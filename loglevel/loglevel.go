// Copyright 2015 Carleton University Library All rights reserved.
// Use of this source code is governed by the MIT
// license that can be found in the LICENSE file.

// This package provides a way to filter log messages
// depending on their log level.
package loglevel

import (
	"fmt"
	"log"
	"strings"
	"sync"
)

type LogLevel int

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

func Set(level LogLevel) {
	logMessageLevelMutex.Lock()
	defer logMessageLevelMutex.Unlock()

	logMessageLevel = level
}

//Log a message if the level is below or equal to the set LogMessageLevel
func Log(messagelevel LogLevel, message interface{}) {
	logMessageLevelMutex.RLock()
	defer logMessageLevelMutex.RUnlock()

	if messagelevel <= logMessageLevel {
		log.Printf("%v: %v\n", messagelevel, message)
	}
}

func Logf(messagelevel LogLevel, format string, a ...interface{}) {
	Log(messagelevel, fmt.Sprintf(format, a), )
}

func (level LogLevel) String() string {
	return logLevelToString[level]
}

func ParseLogLevel(parseThis string) (LogLevel, error) {
	for logLevel, logLevelString := range logLevelToString {
		if logLevelString == strings.ToUpper(parseThis) {
			return logLevel, nil
		}

	}
	return TraceMessage, fmt.Errorf("Unknown log level '%v', defaulting to TRACE.", parseThis)
}
