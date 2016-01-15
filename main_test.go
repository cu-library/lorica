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
)

func init() {
	l.Set(l.ErrorMessage)
	log.SetOutput(ioutil.Discard)
}

func TestProxyHanderPreflightCorrect(t *testing.T) {
	req, err := http.NewRequest("OPTIONS", "/", nil)
	if err != nil {
		t.Fatal(err)
	}

	req.Header.Add("Access-Control-Request-Method", "GET")

	w := httptest.NewRecorder()
	proxyHandler(w, req)

	if w.Code != http.StatusOK {
		t.Error("Preflight request handler didn't handle properly formatted request.")
	}

}

func TestBuildHeader(t *testing.T) {

	// Using the values from http://api.summon.serialssolutions.com/help/api/authentication
	apiRequestURL, _ := url.Parse("http://api.summon.serialssolutions.com/2.0.0/search?s.q=forest&s.ff=ContentType,or,1,15")
	accept := "application/xml"
	timestamp := "Tue, 30 Jun 2009 12:10:24 GMT"

	// Override the command line flags
	accessIDVal := "test"
	accessID = &accessIDVal
	secretKeyVal := "ed2ee2e0-65c1-11de-8a39-0800200c9a66"
	secretKey = &secretKeyVal

	header := buildHeader(apiRequestURL, accept, timestamp)
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
	setACAOHeader(w, r)
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

	// Override the command line flags
	allowedOriginsVal := "http://test.com"
	allowedOrigins = &allowedOriginsVal

	setACAOHeader(w, r)
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

	// Override the command line flags
	allowedOriginsVal := "http://test.com"
	allowedOrigins = &allowedOriginsVal

	setACAOHeader(w, r)
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

	// Override the command line flags
	allowedOriginsVal := "http://test.com;http://test2.com"
	allowedOrigins = &allowedOriginsVal

	setACAOHeader(w, r)
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

	// Override the command line flags
	allowedOriginsVal := "http://test.com;http://test2.com"
	allowedOrigins = &allowedOriginsVal

	setACAOHeader(w, r)
	if w.HeaderMap.Get("Access-Control-Allow-Origin") != "http://test2.com" {
		t.Error("Access-Control-Allow-Origin not set properly.")
	}
}
