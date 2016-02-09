// Copyright 2015 Carleton University Library All rights reserved.
// Use of this source code is governed by the MIT
// license that can be found in the LICENSE file.

// Package loglevel provides a way to filter log messages
// depending on their log level.
package loglevel

import (
	"fmt"
	"log"
	"strings"
	"sync"
)

// LogLevel defines a hirarchy of levels for classifing
// log messages.
type LogLevel int

// An enumeration of LogLevels.
const (
	ErrorMessage LogLevel = iota
	WarnMessage
	InfoMessage
	DebugMessage
	TraceMessage
)

var logLevelToString = map[LogLevel]string{
	ErrorMessage: "ERROR",
	WarnMessage:  "WARN",
	InfoMessage:  "INFO",
	DebugMessage: "DEBUG",
	TraceMessage: "TRACE",
}

var logMessageLevel = ErrorMessage
var logMessageLevelMutex = new(sync.RWMutex)

// Set the package's message level.
func Set(level LogLevel) {
	logMessageLevelMutex.Lock()
	defer logMessageLevelMutex.Unlock()

	logMessageLevel = level
}

// Log a message if the messagelevel is below or equal to the set LogLevel.
// For example, if the package is set to InfoMessage, then ErrorMessage,
// WarnMessage, and InfoMessage would be printed, but
// DebugMessage and TraceMessage would not be.
func Log(messagelevel LogLevel, message interface{}) {
	logMessageLevelMutex.RLock()
	defer logMessageLevelMutex.RUnlock()

	if messagelevel <= logMessageLevel {
		log.Printf("%v: %v\n", messagelevel, message)
	}
}

// Logf is a wrapper around Log(). It first formats the log message
// using the provided format string.
func Logf(messagelevel LogLevel, format string, a ...interface{}) {
	Log(messagelevel, fmt.Sprintf(format, a...))
}

// Return the string representation of the LogLevel.
func (level LogLevel) String() string {
	return logLevelToString[level]
}

// ParseLogLevel parses a string, returns a log level.
func ParseLogLevel(parseThis string) (LogLevel, error) {
	for logLevel, logLevelString := range logLevelToString {
		if logLevelString == strings.ToUpper(parseThis) {
			return logLevel, nil
		}

	}
	return TraceMessage, fmt.Errorf("Unknown log level '%v', defaulting to TRACE.", parseThis)
}
