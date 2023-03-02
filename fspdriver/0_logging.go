package fspdriver

import (
	"log"
	"os"
)

var (
    WARNINGLogger *log.Logger
    INFOLogger    *log.Logger
    ERRORLogger   *log.Logger
	DEBUGLogger   *log.Logger
)

var (
	LOG_LEVEL = 20  // default log level
	DEBUG_LEVEL = 10
	INFO_LEVEL = 20
	WARNING_LEVEL = 30
	ERROR_LEVEL = 40	
)

func init() {

	logLevelStr := os.Getenv("LOG_LEVEL")
	if logLevelStr != "" {
		switch (logLevelStr) {
		case "DEBUG":
			LOG_LEVEL = 10
		case "INFO":
			LOG_LEVEL = 20
		case "WARNING":
			LOG_LEVEL = 30
		case "ERROR":
			LOG_LEVEL = 40
		default:
			WARNINGLogger.Printf("Unrecognized LOG_LEVEL env variable value: %s. Keeping LOG_LEVEL at level INFO (20)", logLevelStr)
		}
	}

	flag := log.Ldate|log.Ltime|log.Lmicroseconds|log.Lmsgprefix|log.Lshortfile

    INFOLogger = log.New(os.Stderr, "INFO ", flag)
    WARNINGLogger = log.New(os.Stderr, "WARNING ", flag)
    ERRORLogger = log.New(os.Stderr, "ERROR ", flag)
	DEBUGLogger = log.New(os.Stderr, "DEBUG ", flag)
}