// Copyright 2016 Carleton University Library All rights reserved.
// Use of this source code is governed by the MIT
// license that can be found in the LICENSE file.

/*
Package lorica provides a web server which proxies
queries to the Summon API.
*/
package main

import (
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base64"
	"flag"
	"fmt"
	l "github.com/cu-library/lorica/loglevel"
	"github.com/didip/tollbooth"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"
)

const (
	// EnvPrefix is the prefix for the environment variables.
	EnvPrefix string = "LORICA_"

	// DefaultAddress is the default address to serve from.
	DefaultAddress string = ":8877"

	// DefaultLogLevel is the default log level.
	DefaultLogLevel = "WARN"

	// DefaultSummonAPIURL is the default Summon API URL.
	DefaultSummonAPIURL = "https://api.summon.serialssolutions.com"

	// DefaultMaxAge is the default number of seconds for the Access-Control-Max-Age header.
	DefaultMaxAge = "604800"

	// DefaultSummonAPITimeout is the number of seconds this service will wait for a response from Summon.
	DefaultSummonAPITimeout = 10

	// DefaultMaxRequestsPerSecond is the maximum number of requests that will be processed from one IP in a second.
	DefaultMaxRequestsPerSecond = 1
)

var (
	address        = flag.String("address", DefaultAddress, "Address for the server to bind on.")
	apiURL         = flag.String("summonapi", DefaultSummonAPIURL, "Summon API URL.")
	accessID       = flag.String("accessid", "", "Access ID")
	secretKey      = flag.String("secretkey", "", "Secret Key")
	allowedOrigins = flag.String("allowedorigins", "", "A list of allowed origins for CORS, delimited by the ; character. "+
		"To allow any origin to connect, use *.")
	logLevel = flag.String("loglevel", "warn", "The maximum log level which will be logged. "+
		"error < warn < info < debug < trace. "+
		"For example, trace will log everything, info will log info, warn, and error.")
	timeout     = flag.Int("timeout", DefaultSummonAPITimeout, "The number of seconds to wait for a response from Summon.")
	rateLimit   = flag.Bool("ratelimit", true, "Enable and disable rate limiting.")
	maxRequests = flag.Float64("maxrequests", DefaultMaxRequestsPerSecond, "The maximum number of requests accepted from "+
		"one client per one second interval.")
	checkProxyHeaders = flag.Bool("checkproxyheaders", false, "Have the rate limiter use the IP address from the "+
		"X-Forwarded-For and X-Real-IP header first. You may need this if you are running Lorica behind a proxy.")

	// A version flag, which should be overwritten when building using ldflags.
	version = "devel"
)

func init() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Lorica: An authenticating proxy for the Summon API\nVersion %v\n\n", version)
		flag.PrintDefaults()
		fmt.Fprintln(os.Stderr, "  The possible environment variables:")

		flag.VisitAll(func(f *flag.Flag) {
			uppercaseName := strings.ToUpper(f.Name)
			fmt.Fprintf(os.Stderr, "  %v%v\n", EnvPrefix, uppercaseName)
		})
	}
}

func main() {

	// Process the flags.
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

	// Is the apiURL parseable?
	_, err = url.Parse(*apiURL)
	if err != nil {
		log.Fatalf("FATAL: Unable to parse Summon API URL.")
	}

	// Greet the user.
	l.Log(l.InfoMessage, "Serving on address: "+*address)
	l.Log(l.InfoMessage, "Using API URL: "+*apiURL)
	l.Log(l.InfoMessage, "Allowed Origins for CORS: "+*allowedOrigins)
	l.Log(l.InfoMessage, "Summon API Timeout: "+strconv.Itoa(*timeout)+" seconds")

	// If any of the required flags are not set, exit.
	if *accessID == "" {
		log.Fatal("FATAL: An access ID for the Summon API is required.")
	} else if *secretKey == "" {
		log.Fatal("FATAL: An secret key for the Summon API is required.")
	}

	// Warn if the allowedOrigins flag is empty.
	if *allowedOrigins == "" {
		l.Log(l.WarnMessage, "No Allowed Origins for CORS! No CORS requests will be processed.")
	}

	// HTTP handler. All requests are proxied to the Summon API.
	if *rateLimit {
		l.Log(l.InfoMessage, "Rate Limiting Enabled: Max "+strconv.FormatFloat(*maxRequests, 'f', -1, 64)+" request(s) per second.")
		if *checkProxyHeaders {
			l.Log(l.InfoMessage, "Using client IP from headers.")
		}
		limiter := tollbooth.NewLimiter(*maxRequests, nil)
		if *checkProxyHeaders {
			limiter.SetIPLookups([]string{"X-Forwarded-For", "X-Real-IP", "RemoteAddr"})
		}
		http.Handle("/", tollbooth.LimitFuncHandler(limiter, proxyHandler))
	} else {
		l.Log(l.InfoMessage, "Rate Limiting Disabled!")
		http.HandleFunc("/", proxyHandler)
	}

	// Run the HTTP server. If ListenAndServe returns,
	// then there was an error.
	l.Log(l.TraceMessage, "Starting server.")
	log.Fatalf("FATAL: %v", http.ListenAndServe(*address, nil))
}

// proxyHandler is responsible for the duties of a CORS
// server and proxying requests to the Summon API.
func proxyHandler(w http.ResponseWriter, r *http.Request) {

	// If the Origin header is set, this might be a CORS request.
	if r.Header.Get("Origin") != "" {
		if r.Method == "OPTIONS" {
			// If this is an OPTIONS request and the Access-Control-Request-Method
			// header isn't set, it isn't accepted.
			preflightRequestMethod := r.Header.Get("Access-Control-Request-Method")
			if preflightRequestMethod == "" {
				sendError(w, http.StatusBadRequest,
					"Access-Control-Request-Method header "+
						"should be set for OPTIONS request.")
				return
			}
			// Otherwise, this is a preflight request.
			// The Access-Control-Request-Method must be GET.
			if preflightRequestMethod != "GET" {
				sendError(w, http.StatusBadRequest,
					"Access-Control-Request-Method header "+
						"should only be GET.")
				return
			}
			// The Access-Control-Request-Header should not be set or
			// only contain x-summon-session-id
			preflightRequestHeader := r.Header.Get("Access-Control-Request-Header")
			if preflightRequestHeader != "" && preflightRequestHeader != "x-summon-session-id" {
				sendError(w, http.StatusBadRequest,
					"Access-Control-Request-Header header "+
						"should only contain x-summon-session-id.")
				return
			}
			w.Header().Set("Access-Control-Allow-Methods", "GET")
			w.Header().Set("Access-Control-Allow-Headers", "x-summon-session-id")
			w.Header().Set("Access-Control-Max-Age", DefaultMaxAge)
			setACAOHeader(w, r)

			l.Logf(l.TraceMessage, "Sending preflight response %#v.", w.Header())

			// Write an empty body.
			w.Write([]byte{})
			return
		}

		// Not a preflight request, so it has to be a GET request.
		if r.Method != "GET" {
			sendError(w, http.StatusMethodNotAllowed,
				"Only GET requests accepted.")
			return
		}

		// Set the Access-Control-Allow-Origin header.
		setACAOHeader(w, r)

	}

	// Build the auth headers and send a request to the Summon API.
	client := new(http.Client)

	// Add a timeout to the http client
	client.Timeout = time.Duration(*timeout) * time.Second

	// Build the API Request.
	apiRequestURL, err := url.Parse(*apiURL)
	if err != nil {
		// This should never happen, since we already parsed in main.
		sendError(w, http.StatusInternalServerError, "Unable to parse API URL.")
		return
	}
	apiRequestURL.Path = r.URL.Path
	apiRequestURL.RawQuery = r.URL.RawQuery

	// Create the request struct.
	apiRequest, err := http.NewRequest("GET", apiRequestURL.String(), nil)
	if err != nil {
		sendError(w, http.StatusInternalServerError,
			"Unable to build API Request.")
		return
	}

	// Close the connection after sending the request.
	apiRequest.Close = true

	// Add the accept header from the client.
	accept := r.Header.Get("Accept")
	apiRequest.Header.Add("Accept", accept)

	// Add the timestamp
	timestampRFC2616 := time.Now().UTC().Format(http.TimeFormat)
	apiRequest.Header.Add("x-summon-date", timestampRFC2616)

	// Add the session id from the client, if available.
	sessionID := r.Header.Get("x-summon-session-id")
	if sessionID != "" {
		apiRequest.Header.Add("x-summon-session-id", sessionID)
	}

	// Call the helper function to build the accept header.
	apiRequest.Header.Add("Authorization", buildHeader(apiRequestURL, accept, timestampRFC2616))

	l.Logf(l.TraceMessage, "Sending request to Summon API %#v", apiRequest)

	// Send the response to the Summon API.
	apiResp, err := client.Do(apiRequest)
	if err != nil {
		sendError(w, http.StatusInternalServerError,
			fmt.Sprintf("Error sending API Request: %v", err))
		return
	}

	l.Logf(l.TraceMessage, "Received response from Summon API: %#v", apiResp)

	// Send the client important Summon API headers
	proxiedHeaders := []string{
		"Content-Type",
	}

	for _, proxiedHeader := range proxiedHeaders {
		if apiResp.Header.Get(proxiedHeader) != "" {
			w.Header().Add(proxiedHeader, apiResp.Header.Get(proxiedHeader))
		}
	}

	l.Logf(l.TraceMessage, "Sending response to client with headers: %v", w.Header())

	w.WriteHeader(apiResp.StatusCode)
	io.Copy(w, apiResp.Body)
	apiResp.Body.Close()

}

// A helper function that uses a HMAC with SHA1 to build the Authorization header.
func buildHeader(apiRequestURL *url.URL, accept, timestampRFC2616 string) string {

	// The slice which holds the pieces of the identification string.
	idComponents := make([]string, 5)
	idComponents[0] = accept
	idComponents[1] = timestampRFC2616
	idComponents[2] = apiRequestURL.Host
	idComponents[3] = apiRequestURL.Path

	// Build a list of query parameters.
	var queryStrings []string
	for key, values := range apiRequestURL.Query() {
		for _, value := range values {
			queryStrings = append(queryStrings, key+"="+value)
		}
	}

	// Sort that list in place.
	sort.Strings(queryStrings)

	// Concatinate the list with &, and add it to idComponents.
	idComponents[4] = strings.Join(queryStrings, "&")

	l.Logf(l.DebugMessage, "Authorizing %v", idComponents)

	// Make the id string from the slice of values.
	idString := strings.Join(idComponents, "\n") + "\n"

	// Hash using sha1, then base64 encode.
	hmacsha1 := hmac.New(sha1.New, []byte(*secretKey))
	io.WriteString(hmacsha1, idString)
	encodedHash := base64.StdEncoding.EncodeToString(hmacsha1.Sum(nil))

	// Build the final auth header.
	return fmt.Sprintf("Summon %v;%v", *accessID, encodedHash)
}

// Send an error to the client, and log the error.
func sendError(w http.ResponseWriter, statuscode int, message string) {

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(statuscode)
	fmt.Fprintf(w, "<html><head></head><body><pre>%v %v - %v</pre></body></html>",
		statuscode, http.StatusText(statuscode), message)
	l.Logf(l.ErrorMessage, "%v - %v", statuscode, message)
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
	for k := range listOfUnsetFlags {

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

// Set the Access-Control-Allow-Origin header
func setACAOHeader(w http.ResponseWriter, r *http.Request) {

	if *allowedOrigins == "*" {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		return
	}

	if *allowedOrigins != "" {
		possibleOrigins := strings.Split(*allowedOrigins, ";")
		for _, okOrigin := range possibleOrigins {
			okOrigin = strings.TrimSpace(okOrigin)
			if (okOrigin != "") && (okOrigin == r.Header.Get("Origin")) {
				w.Header().Set("Access-Control-Allow-Origin", okOrigin)
				return
			}
		}
	}
}
