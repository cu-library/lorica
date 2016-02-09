// Copyright 2015 Carleton University Library All rights reserved.
// Use of this source code is governed by the MIT
// license that can be found in the LICENSE file.

package main

import (
	"fmt"
	l "github.com/cu-library/lorica/loglevel"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"
)

func init() {
	l.Set(l.ErrorMessage)
	log.SetOutput(ioutil.Discard)
}

// Test that a correctly formatted preflight request
// works as expected.
func TestProxyHanderPreflightCorrect(t *testing.T) {
	req, err := http.NewRequest("OPTIONS", "/", nil)
	if err != nil {
		t.Fatal(err)
	}

	req.Header.Add("Access-Control-Request-Method", "GET")
	req.Header.Add("Origin", "http://test.com")

	// Override the command line flags
	oldAllowedOrigins := *allowedOrigins
	*allowedOrigins = "http://test.com"
	defer func() { *allowedOrigins = oldAllowedOrigins }()

	w := httptest.NewRecorder()
	proxyHandler(w, req)

	if w.Code != http.StatusOK {
		t.Error("Preflight request handler didn't handle properly formatted request.")
	}
	if w.Header().Get("Access-Control-Allow-Methods") != "GET" {
		t.Errorf("Access-Control-Allow-Methods header had %v, expected GET",
			w.Header().Get("Access-Control-Allow-Methods"))
	}
	if w.Header().Get("Access-Control-Allow-Headers") != "x-summon-session-id" {
		t.Errorf("Access-Control-Allow-Headers header had %v, expected x-summon-session-id",
			w.Header().Get("Access-Control-Allow-Headers"))
	}
	if w.Header().Get("Access-Control-Max-Age") != DefaultMaxAge {
		t.Errorf("Access-Control-Max-Age header had %v, expected %v",
			w.Header().Get("Access-Control-Max-Age"), DefaultMaxAge)
	}
	if w.Header().Get("Access-Control-Allow-Origin") != "http://test.com" {
		t.Errorf("Access-Control-Allow-Origin header had %v, expected http://test.com",
			w.Header().Get("Access-Control-Allow-Methods"))
	}

}

// Test that a preflight request with no request method
// fails as expected.
func TestProxyHanderPreflightNoMethodHeader(t *testing.T) {
	req, err := http.NewRequest("OPTIONS", "/", nil)
	if err != nil {
		t.Fatal(err)
	}

	req.Header.Add("Origin", "http://test.com")

	w := httptest.NewRecorder()
	proxyHandler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Error("Preflight request with Access-Control-Request-Method set to POST should have failed.")
	}
	bodyString := w.Body.String()
	if !strings.Contains(bodyString, "Access-Control-Request-Method header should be set for OPTIONS request.") {
		t.Errorf("Didn't get the right message from bad preflight request, got %v.", bodyString)
	}

}

// Test that a preflight request with a bad request method
// fails as expected.
func TestProxyHanderPreflightBadMethodHeader(t *testing.T) {
	req, err := http.NewRequest("OPTIONS", "/", nil)
	if err != nil {
		t.Fatal(err)
	}

	req.Header.Add("Access-Control-Request-Method", "POST")
	req.Header.Add("Origin", "http://test.com")

	w := httptest.NewRecorder()
	proxyHandler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Error("Preflight request with Access-Control-Request-Method set to POST should have failed.")
	}
	bodyString := w.Body.String()
	if !strings.Contains(bodyString, "Access-Control-Request-Method header should only be GET") {
		t.Errorf("Didn't get the right message from bad preflight request, got %v.", bodyString)
	}

}

// Test that a preflight request with a bad request header
// fails as expected.
func TestProxyHanderPreflightBadRequestHeader(t *testing.T) {
	req, err := http.NewRequest("OPTIONS", "/", nil)
	if err != nil {
		t.Fatal(err)
	}

	req.Header.Add("Access-Control-Request-Method", "GET")
	req.Header.Add("Access-Control-Request-Header", "bad-news")
	req.Header.Add("Origin", "http://test.com")

	w := httptest.NewRecorder()
	proxyHandler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Error("Preflight request with Access-Control-Request-Header set to bad-news should have failed.")
	}
	bodyString := w.Body.String()
	if !strings.Contains(bodyString, "Access-Control-Request-Header header should only contain x-summon-session-id.") {
		t.Errorf("Didn't get the right message from bad preflight request, got %v.", bodyString)
	}

}

// Test that a CORS request that isn't preflight should be a GET.
func TestProxyHanderBadCORSMethod(t *testing.T) {
	req, err := http.NewRequest("POST", "/", nil)
	if err != nil {
		t.Fatal(err)
	}

	req.Header.Add("Origin", "http://test.com")

	w := httptest.NewRecorder()
	proxyHandler(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Error("CORS request that isn't GET should fail.")
	}
	bodyString := w.Body.String()
	if !strings.Contains(bodyString, "Only GET requests accepted.") {
		t.Errorf("Didn't get the right message from bad CORS method request, got %v.", bodyString)
	}

}

// Mock the Summon API, and test that proxyHandler works as expected.
func TestProxyHanderAPICall(t *testing.T) {

	// The mock of the Sierra API.
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		// The request should have the right path
		if r.URL.Path != "/2.0.0/search" {
			t.Error("Sierra API got the wrong path.")
		}
		// The request should have the right query
		if r.URL.RawQuery != "s.q=test" {
			t.Error("Sierra API got the wrong query.")
		}
		// The request should have an x-summon-date header
		if r.Header.Get("x-summon-date") == "" {
			t.Error("Sierra API didn't recieve x-summon-date header.")
		}
		// The request should have an Authorization header
		if r.Header.Get("Authorization") == "" {
			t.Error("Sierra API didn't recieve Authorization header.")
		}

		fmt.Fprintln(w, "")
	}))
	defer ts.Close()

	// Override the command line flags
	oldAPIURL := *apiURL
	*apiURL = ts.URL
	defer func() { *apiURL = oldAPIURL }()

	// The request from the client.
	req, err := http.NewRequest("GET", "/2.0.0/search?s.q=test", nil)
	if err != nil {
		t.Fatal(err)
	}

	// The response to the client.
	w := httptest.NewRecorder()

	proxyHandler(w, req)

}

// Test the build header with data from the Summon API
func TestBuildHeader(t *testing.T) {

	// Using the values from http://api.summon.serialssolutions.com/help/api/authentication
	apiRequestURL, _ := url.Parse("http://api.summon.serialssolutions.com/2.0.0/search?s.q=forest&s.ff=ContentType,or,1,15")
	accept := "application/xml"
	timestamp := "Tue, 30 Jun 2009 12:10:24 GMT"

	// Override the command line flags
	oldAccessID := *accessID
	*accessID = "test"
	defer func() { *accessID = oldAccessID }()

	oldSecretKey := *secretKey
	*secretKey = "ed2ee2e0-65c1-11de-8a39-0800200c9a66"
	defer func() { *secretKey = oldSecretKey }()

	header := buildHeader(apiRequestURL, accept, timestamp)
	goodheader := "Summon test;3a4+j0Wrrx6LF8X4iwOLDetVOu4="

	if header != goodheader {
		t.Errorf("buildheader did not build the right header!\n%v\ninstead of\n%v", header, goodheader)
	}

}

// sendError should return the right errors.
func TestSendError(t *testing.T) {

	sendErrorTestTable := []struct {
		statuscode int
		message    string
	}{
		{http.StatusBadRequest, "Access-Control-Request-Method header should be set for OPTIONS request."},
		{http.StatusUnauthorized, "You're doing it wrong!"},
		{http.StatusInternalServerError, "We're doing it wrong!"},
	}

	for _, entry := range sendErrorTestTable {
		w := httptest.NewRecorder()
		sendError(w, entry.statuscode, entry.message)
		if w.Code != entry.statuscode {
			t.Errorf("Bad status code, got %v for entry %#v.", w.Code, entry)
		}
		if w.Header().Get("Content-Type") != "text/html; charset=utf-8" {
			t.Errorf("Bad Content-Type for entry %#v.", entry)
		}
		bodyString := w.Body.String()
		if !strings.Contains(bodyString, entry.message) {
			t.Errorf("Didn't get the right contents from error message, got %v for entry %#v.", bodyString, entry)
		}
	}

}

// See if setting an env var overrides an unset flag.
func TestEnvironmentVariableOverrideByFlag(t *testing.T) {
	os.Setenv(EnvPrefix+"ADDRESS", ":8080")
	overrideUnsetFlagsFromEnvironmentVariables()
	if *address != ":8080" {
		t.Error("Setting an environment variable did not override an unset flag.")
	}
}

// Set the ACAO header, the default case. Don't set the header at all.
func TestSetACAOHeaderNoConfig(t *testing.T) {
	r, err := http.NewRequest("GET", "/", nil)
	if err != nil {
		t.Fatal(err)
	}
	w := httptest.NewRecorder()
	setACAOHeader(w, r)
	if w.HeaderMap.Get("Access-Control-Allow-Origin") != "" {
		t.Error("Access-Control-Allow-Origin shouldn't be set.")
	}
}

// Set the ACAO config to a single origin which doesn't match.
func TestSetACAOHeaderNotMatchOnSingle(t *testing.T) {
	r, err := http.NewRequest("GET", "/", nil)
	if err != nil {
		t.Fatal(err)
	}
	w := httptest.NewRecorder()

	// Override the command line flags
	oldAllowedOrigins := *allowedOrigins
	*allowedOrigins = "http://test.com"
	defer func() { *allowedOrigins = oldAllowedOrigins }()

	setACAOHeader(w, r)
	if w.HeaderMap.Get("Access-Control-Allow-Origin") != "" {
		t.Error("Access-Control-Allow-Origin shouldn't be set.")
	}
}

// Set the ACAO config to a single origin which does match.
func TestSetACAOHeaderMatchOnSingle(t *testing.T) {
	r, err := http.NewRequest("GET", "/", nil)
	if err != nil {
		t.Fatal(err)
	}
	r.Header.Set("Origin", "http://test.com")
	w := httptest.NewRecorder()

	// Override the command line flags
	oldAllowedOrigins := *allowedOrigins
	*allowedOrigins = "http://test.com"
	defer func() { *allowedOrigins = oldAllowedOrigins }()

	setACAOHeader(w, r)
	if w.HeaderMap.Get("Access-Control-Allow-Origin") != "http://test.com" {
		t.Error("Access-Control-Allow-Origin not set properly.")
	}
}

// Set the ACAO config to a one of a list of origins, none of which match.
func TestSetACAOHeaderNoMatchOnList(t *testing.T) {
	r, err := http.NewRequest("GET", "/", nil)
	if err != nil {
		t.Fatal(err)
	}
	r.Header.Set("Origin", "http://test3.com")
	w := httptest.NewRecorder()

	// Override the command line flags
	oldAllowedOrigins := *allowedOrigins
	*allowedOrigins = "http://test.com;http://test2.com"
	defer func() { *allowedOrigins = oldAllowedOrigins }()

	setACAOHeader(w, r)
	if w.HeaderMap.Get("Access-Control-Allow-Origin") != "" {
		t.Error("Access-Control-Allow-Origin shouldn't be set.")
	}
}

// Set the ACAO config to a one of a list of origins, one of which does match.
func TestSetACAOHeaderMatchOnList(t *testing.T) {
	r, err := http.NewRequest("GET", "/", nil)
	if err != nil {
		t.Fatal(err)
	}
	r.Header.Set("Origin", "http://test2.com")
	w := httptest.NewRecorder()

	// Override the command line flags
	oldAllowedOrigins := *allowedOrigins
	*allowedOrigins = "http://test.com;http://test2.com"
	defer func() { *allowedOrigins = oldAllowedOrigins }()

	setACAOHeader(w, r)
	if w.HeaderMap.Get("Access-Control-Allow-Origin") != "http://test2.com" {
		t.Error("Access-Control-Allow-Origin not set properly.")
	}
}

// Set the ACAO config to the wildcard.
func TestSetACAOHeaderWildcard(t *testing.T) {
	r, err := http.NewRequest("GET", "/", nil)
	if err != nil {
		t.Fatal(err)
	}
	r.Header.Set("Origin", "http://test.com")
	w := httptest.NewRecorder()

	// Override the command line flags
	oldAllowedOrigins := *allowedOrigins
	*allowedOrigins = "*"
	defer func() { *allowedOrigins = oldAllowedOrigins }()

	setACAOHeader(w, r)
	if w.HeaderMap.Get("Access-Control-Allow-Origin") != "*" {
		t.Error("Access-Control-Allow-Origin not set properly.")
	}
}
