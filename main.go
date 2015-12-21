// Copyright 2015 Carleton University Library All rights reserved.
// Use of this source code is governed by the MIT
// license that can be found in the LICENSE file.

/*
Package lorica provides a web application which creates
authorization headers for the Summon API.
*/
package main

import (
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	l "github.com/cu-library/lorica/loglevel"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"sort"
	"strings"
	"time"
)

const (
	// The prefix for all the environment variables
	EnvPrefix string = "LORICA_"

	// The default address to serve from
	DefaultAddress string = ":8877"

	// The default log level
	DefaultLogLevel = "WARN"

	// The default summon api URL
	DefaultSummonAPIURL = "api.summon.serialssolutions.com"
)

var (
	address        = flag.String("address", DefaultAddress, "Address for the server to bind on.")
	apiURL         = flag.String("summonapi", DefaultSummonAPIURL, "API url.")
	accessID       = flag.String("accessid", "", "Access ID")
	secretKey      = flag.String("secretkey", "", "Secret Key")
	allowedOrigins = flag.String("allowedorigins", "", "A list of allowed origins for CORS, delimited by the ; character. ")
	logLevel       = flag.String("loglevel", "warn", "The maximum log level which will be logged. error < warn < info < debug < trace. "+
		"For example, trace will log everything, info will log info, warn, and error.")
)

func init() {
	flag.Usage = func() {
		fmt.Fprint(os.Stderr, "Lorica: Generate an authorization header for the Summon API\nVersion 0.1.0\n\n")
		flag.PrintDefaults()
		fmt.Fprintln(os.Stderr, "  The possible environment variables:")

		flag.VisitAll(func(f *flag.Flag) {
			uppercaseName := strings.ToUpper(f.Name)
			fmt.Fprintf(os.Stderr, "  %v%v\n", EnvPrefix, uppercaseName)
		})
	}
}

func main() {

	// Process the flags
	flag.Parse()

	// If any flags have not been set, see if there are
	// environment variables that set them.
	overrideUnsetFlagsFromEnvironmentVariables()

	// Set the loglevel in the loglevel subpackage
	level, err := l.ParseLogLevel(*logLevel)
	if err != nil {
		log.Fatal("FATAL: Unable to parse log level.")
	}
	l.Set(level)

	// Greet the user
	l.Log(l.InfoMessage, "Starting Lorica")
	l.Log(l.InfoMessage, "Serving on address: "+*address)
	l.Log(l.InfoMessage, "Using API URL: "+*apiURL)
	l.Log(l.InfoMessage, "Allowed Origins for CORS: "+*allowedOrigins)

	if *accessID == "" {
		log.Fatal("FATAL: An access ID for the Summon API is required.")
	} else if *secretKey == "" {
		log.Fatal("FATAL: An secret key for the Summon API is required.")
	} else if *allowedOrigins == "*" {
		log.Fatal("FATAL: A defined list of allowed origins is required.")
	}

	http.HandleFunc("/", homeHandler)
	http.HandleFunc("/buildheader", buildheaderHandler)

	log.Fatalf("FATAL: %v", http.ListenAndServe(*address, nil))
}

func homeHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		sendErrorToClient(w, http.StatusNotFound, "")
		return
	}
	w.Header().Set("Content-Type", "text/html")
	l.Log(l.TraceMessage, "Home Handler visited.")
	fmt.Fprint(w, "<html><head></head><body><h1>Welcome to Lorica.</h1></body></html>")
}

func buildheaderHandler(w http.ResponseWriter, r *http.Request) {

	setACAOHeader(w, r, *allowedOrigins)

	// We don't need to worry about Preflight requests, since
	// our server only supports Simple Cross-Origin Requests

	if r.Method != "GET" {
		sendErrorToClient(w, http.StatusMethodNotAllowed, "Only GET HTTP method supported.")
		return
	}

	// This is a bit tricky. We're not looking at the 'Accept' header in the request.
	// We're looking for a query value called accept.
	accept := r.URL.Query().Get("accept")
	if accept != "application/json" && accept != "application/xml" {
		sendErrorToClient(w, http.StatusBadRequest, "Bad value for accept parameter.")
		return
	}

	path := r.URL.Query().Get("path")
	if path == "" {
		sendErrorToClient(w, http.StatusBadRequest, "Path parameter required.")
		return
	}

	timestampRFC2616 := time.Now().UTC().Format(http.TimeFormat)

	header := buildHeader(r.URL.Query(), accept, path, timestampRFC2616)

	payload := struct {
		TimestampRFC2616    string `json:"timestamprfc2616"`
		AuthorizationHeader string `json:"authorizationheader"`
	}{
		timestampRFC2616,
		header,
	}

	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		sendErrorToClient(w, http.StatusInternalServerError, "JSON encoding error.")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(jsonPayload)
}

func buildHeader(values url.Values, accept, path, timestampRFC2616 string) string {

	var idStringSlice []string
	idStringSlice = append(idStringSlice, accept)
	idStringSlice = append(idStringSlice, timestampRFC2616)
	idStringSlice = append(idStringSlice, *apiURL)
	idStringSlice = append(idStringSlice, path)

	// Sort by query parameter key and concatinate.
	var queryKeys []string
	var queryParams []string
	for key := range values {
		if key != "path" && key != "accept" {
			queryKeys = append(queryKeys, key)
		}
	}
	sort.Strings(queryKeys)
	for _, key := range queryKeys {
		queryParams = append(queryParams,
			fmt.Sprintf("%v=%v", key, values.Get(key)))
	}
	idStringSlice = append(idStringSlice, strings.Join(queryParams, "&"))

	l.Logf(l.DebugMessage, "Authorizing %v", idStringSlice)

	// Make the id string from the slice of values
	idString := strings.Join(idStringSlice, "\n") + "\n"

	// Hash using sha1, then base64 encode.
	hmacsha1 := hmac.New(sha1.New, []byte(*secretKey))
	io.WriteString(hmacsha1, idString)
	encodedHash := base64.StdEncoding.EncodeToString(hmacsha1.Sum(nil))

	// Build the final auth header
	return fmt.Sprintf("Summon %v;%v", *accessID, encodedHash)
}

func sendErrorToClient(w http.ResponseWriter, statuscode int, message string) {
	w.Header().Set("Content-Type", "text/html")
	w.WriteHeader(statuscode)
	fmt.Fprintf(w, "<html><head></head><body><pre>%v %v - %v</pre></body></html>",
		statuscode, http.StatusText(statuscode), message)
	l.Logf(l.ErrorMessage, "%v in buildheaderHandler, %v", statuscode, message)
}

// If any flags are not set, use environment variables to set them.
func overrideUnsetFlagsFromEnvironmentVariables() {

	// A map of pointers to unset flags.
	listOfUnsetFlags := make(map[*flag.Flag]bool)

	// flag.Visit calls a function on "only those flags that have been set."
	// flag.VisitAll calls a function on "all flags, even those not set."
	// No way to ask for "only unset flags". So, we add all, then
	// delete the set flags.

	// First, visit all the flags, and add them to our map.
	flag.VisitAll(func(f *flag.Flag) { listOfUnsetFlags[f] = true })

	// Then delete the set flags.
	flag.Visit(func(f *flag.Flag) { delete(listOfUnsetFlags, f) })

	// Loop through our list of unset flags.
	// We don't care about the values in our map, only the keys.
	for k, _ := range listOfUnsetFlags {

		// Build the corresponding environment variable name for each flag.
		uppercaseName := strings.ToUpper(k.Name)
		environmentVariableName := fmt.Sprintf("%v%v", EnvPrefix, uppercaseName)

		// Look for the environment variable name.
		// If found, set the flag to that value.
		// If there's a problem setting the flag value,
		// there's a serious problem we can't recover from.
		environmentVariableValue := os.Getenv(environmentVariableName)
		if environmentVariableValue != "" {
			err := k.Value.Set(environmentVariableValue)
			if err != nil {
				log.Fatalf("FATAL: Unable to set configuration option %v from environment variable %v, "+
					"which has a value of \"%v\"",
					k.Name, environmentVariableName, environmentVariableValue)
			}
		}
	}
}

// Set the Access-Control-Allow-Origin
func setACAOHeader(w http.ResponseWriter, r *http.Request, allowedOrigins string) {
	if allowedOrigins != "" {
		possibleOrigins := strings.Split(allowedOrigins, ";")
		for _, okOrigin := range possibleOrigins {
			okOrigin = strings.TrimSpace(okOrigin)
			if (okOrigin != "") && (okOrigin == r.Header.Get("Origin")) {
				w.Header().Set("Access-Control-Allow-Origin", okOrigin)
				return
			}
		}
	}
}
