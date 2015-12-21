// Copyright 2015 Carleton University Library All rights reserved.
// Use of this source code is governed by the MIT
// license that can be found in the LICENSE file.

package main

import (
	l "github.com/cu-library/lorica/loglevel"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"encoding/json"
	"strings"
)

func init() {
	l.Set(l.ErrorMessage)
	log.SetOutput(ioutil.Discard)
}

func TestHomeHandler(t *testing.T) {

	req, err := http.NewRequest("GET", "/", nil)
	if err != nil {
		t.Fatal(err)
	}

	w := httptest.NewRecorder()
	homeHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Home handler didn't return %v", http.StatusOK)
	}
}

func TestHomeHandler404(t *testing.T) {

	req, err := http.NewRequest("GET", "/badurlnocookie", nil)
	if err != nil {
		t.Fatal(err)
	}

	w := httptest.NewRecorder()
	homeHandler(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Home handler didn't return %v for url which should not exist.", http.StatusNotFound)
	}
}

func TestBuildHeaderHandlerPostFails(t *testing.T) {
	req, err := http.NewRequest("POST", "/buildheader", nil)
	if err != nil {
		t.Fatal(err)
	}

	w := httptest.NewRecorder()
	buildheaderHandler(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("Build header handler didn't return %v for method which should not be allowed.", http.StatusMethodNotAllowed)
	}
}

func TestBuildHeaderHandlerHeadFails(t *testing.T) {
	req, err := http.NewRequest("HEAD", "/buildheader", nil)
	if err != nil {
		t.Fatal(err)
	}

	w := httptest.NewRecorder()
	buildheaderHandler(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("Build header handler didn't return %v for method which should not be allowed.", http.StatusMethodNotAllowed)
	}
}

func TestBuildHeaderHandlerNoAcceptParam(t *testing.T) {
	req, err := http.NewRequest("GET", "/buildheader", nil)
	if err != nil {
		t.Fatal(err)
	}

	w := httptest.NewRecorder()
	buildheaderHandler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Build header handler didn't return %v when no accept query param provided.", http.StatusBadRequest)
	}
}

func TestBuildHeaderHandlerBadAcceptParam(t *testing.T) {
	req, err := http.NewRequest("GET", "/buildheader?accept=badvalue", nil)
	if err != nil {
		t.Fatal(err)
	}

	w := httptest.NewRecorder()
	buildheaderHandler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Build header handler didn't return %v when a bad accept query param was provided.", http.StatusBadRequest)
	}
	if !strings.Contains(w.Body.String(), "Bad value for accept parameter") {
		t.Error("Wrong error returned by build header handler.")
	}
}

func TestBuildHeaderHandlerNoPathParam(t *testing.T) {
	req, err := http.NewRequest("GET", "/buildheader?accept=application%2Fjson", nil)
	if err != nil {
		t.Fatal(err)
	}

	w := httptest.NewRecorder()
	buildheaderHandler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Build header handler didn't return %v when no path query param provided.", http.StatusBadRequest)
	}
	if !strings.Contains(w.Body.String(), "Path parameter required") {
		t.Error("Wrong error returned by build header handler.")
	}
}

func TestBuildHeaderHandlerExpectedResults(t *testing.T) {
	req, err := http.NewRequest("GET", "/buildheader?accept=application%2Fjson&path=%2Ftest", nil)
	if err != nil {
		t.Fatal(err)
	}

	w := httptest.NewRecorder()
	buildheaderHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Build header handler didn't return %v.", http.StatusOK)
	}

	if w.HeaderMap.Get("Content-Type") != "application/json" {
		t.Error("Build header handler didn't set the content type header to application/json")
	}

	var payload struct {
		TimestampRFC2616    string `json:"timestamprfc2616"`
		AuthorizationHeader string `json:"authorizationheader"`
	}

	err = json.Unmarshal(w.Body.Bytes(), &payload)
	if err != nil {
		t.Error("Unable to unmarshal json payload from handler.")
	}

}

func TestBuildHeader(t *testing.T) {

	// Using the values from http://api.summon.serialssolutions.com/help/api/authentication
	path := "/2.0.0/search"
	timestamp := "Tue, 30 Jun 2009 12:10:24 GMT"
	accept := "application/xml"
	values := url.Values{}
	values.Set("s.q", "forest")
	values.Set("s.ff", "ContentType,or,1,15")

	// Override the command line flags
	apiURLVal := "api.summon.serialssolutions.com"
	apiURL = &apiURLVal
	accessIDVal := "test"
	accessID = &accessIDVal
	secretKeyVal := "ed2ee2e0-65c1-11de-8a39-0800200c9a66"
	secretKey = &secretKeyVal

	header := buildHeader(values, accept, path, timestamp)
	goodheader := "Summon test;3a4+j0Wrrx6LF8X4iwOLDetVOu4="

	if header != goodheader {
		t.Errorf("buildheader did not build the right header!\n%v\ninstead of\n%v", header, goodheader)
	}

}

//The default case. Don't set the header at all.
func TestSetACAOHeaderNoConfig(t *testing.T) {
	r, err := http.NewRequest("GET", "/", nil)
	if err != nil {
		t.Fatal(err)
	}
	w := httptest.NewRecorder()
	setACAOHeader(w, r, "")
	if w.HeaderMap.Get("Access-Control-Allow-Origin") != "" {
		t.Error("Access-Control-Allow-Origin shouldn't be set.")
	}
}


//Set the ACAO config to a single origin which doesn't match.
func TestSetACAOHeaderNotMatchOnSingle(t *testing.T) {
	r, err := http.NewRequest("GET", "/", nil)
	if err != nil {
		t.Fatal(err)
	}
	w := httptest.NewRecorder()
	setACAOHeader(w, r, "http://test.com")
	if w.HeaderMap.Get("Access-Control-Allow-Origin") != "" {
		t.Error("Access-Control-Allow-Origin shouldn't be set.")
	}
}

//Set the ACAO config to a single origin which does match.
func TestSetACAOHeaderMatchOnSingle(t *testing.T) {
	r, err := http.NewRequest("GET", "/", nil)
	if err != nil {
		t.Fatal(err)
	}
	r.Header.Set("Origin", "http://test.com")
	w := httptest.NewRecorder()
	setACAOHeader(w, r, "http://test.com")
	if w.HeaderMap.Get("Access-Control-Allow-Origin") != "http://test.com" {
		t.Error("Access-Control-Allow-Origin not set properly.")
	}
}

//Set the ACAO config to a one of a list of origins, none of which match.
func TestSetACAOHeaderNoMatchOnList(t *testing.T) {
	r, err := http.NewRequest("GET", "/", nil)
	if err != nil {
		t.Fatal(err)
	}
	r.Header.Set("Origin", "http://test3.com")
	w := httptest.NewRecorder()
	setACAOHeader(w, r, "http://test.com;http://test2.com;")
	if w.HeaderMap.Get("Access-Control-Allow-Origin") != "" {
		t.Error("Access-Control-Allow-Origin shouldn't be set.")
	}
}

//Set the ACAO config to a one of a list of origins, one of which does match.
func TestSetACAOHeaderMatchOnList(t *testing.T) {
	r, err := http.NewRequest("GET", "/", nil)
	if err != nil {
		t.Fatal(err)
	}
	r.Header.Set("Origin", "http://test2.com")
	w := httptest.NewRecorder()
	setACAOHeader(w, r, "http://test.com;http://test2.com;")
	if w.HeaderMap.Get("Access-Control-Allow-Origin") != "http://test2.com" {
		t.Error("Access-Control-Allow-Origin not set properly.")
	}
}
