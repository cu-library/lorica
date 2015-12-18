// Copyright 2015 Carleton University Library All rights reserved.
// Use of this source code is governed by the MIT
// license that can be found in the LICENSE file.

package loglevel

import (
	"bytes"
	"fmt"
	"log"
	"strings"
	"testing"
)

func TestLogLevelParse(t *testing.T) {

	//Our expected results, in maps
	logLevelToString := map[LogLevel]string{
		ErrorMessage: "ERROR",
		WarnMessage:  "WARN",
		InfoMessage:  "INFO",
		DebugMessage: "DEBUG",
		TraceMessage: "TRACE",
	}

	stringToLogLevel := map[string]LogLevel{
		"error": ErrorMessage,
		"warn":  WarnMessage,
		"info":  InfoMessage,
		"debug": DebugMessage,
		"trace": TraceMessage,
		"ERROR": ErrorMessage,
		"WARN":  WarnMessage,
		"INFO":  InfoMessage,
		"DEBUG": DebugMessage,
		"TRACE": TraceMessage,
	}

	for logLevel, logLevelExpectedString := range logLevelToString {
		if logLevel.String() != logLevelExpectedString {
			t.Errorf("Unable to parse log level %v properly", logLevel)
		}
	}

	for parseString, logLevel := range stringToLogLevel {
		if level, _ := ParseLogLevel(parseString); level != logLevel {
			t.Errorf("Unable to parse log level string %v properly", parseString)
		}
	}

	level, err := ParseLogLevel("blahblahblah")
	if level != TraceMessage {
		t.Error("Default case for string to log level broken.")
	}
	if err == nil {
		t.Error("ParseLogLevel doesn't return error on bad input.")
	}

}

func TestLogFormatting(t *testing.T) {

	testMessage := "Test Message"

	logLevels := []LogLevel{
		ErrorMessage,
		WarnMessage,
		InfoMessage,
		DebugMessage,
		TraceMessage,
	}

	for _, level := range logLevels {
		Set(level)
		b := new(bytes.Buffer)
		log.SetOutput(b)
		Log(level, testMessage)
		if !strings.HasSuffix(b.String(), fmt.Sprintf("%v: %v\n", level, testMessage)) {
			t.Errorf("The log level %v logged the wrong message.", level)
		}
	}
}

func TestLogLevel(t *testing.T) {

	logLevelToExpectedLength := map[LogLevel]int{
		ErrorMessage: 2,
		WarnMessage:  3,
		InfoMessage:  4,
		DebugMessage: 5,
		TraceMessage: 6,
	} //One more than expected, because of empty string at end of Split()

	logLevels := []LogLevel{
		ErrorMessage,
		WarnMessage,
		InfoMessage,
		DebugMessage,
		TraceMessage,
	}

	for _, level := range logLevels {
		b := new(bytes.Buffer)
		Set(level)
		for _, messageLevel := range logLevels {
			log.SetOutput(b)
			Log(messageLevel, "x")
		}
		if len(strings.Split(b.String(), "\n")) != logLevelToExpectedLength[level] {
			t.Logf("%#v", strings.Split(b.String(), "\n"))
			t.Errorf("The log level %v logged the wrong number of messages.", level)
		}
	}
}
