package utils

import (
	"log"
)

// ReportError logs an error to the centralized logging system.
// This function should be used for all error reporting.
func ReportError(err error, msg string) {
	if err == nil {
		return
	}
	if msg != "" {
		log.Printf("[error] %s: %v", msg, err)
	} else {
		log.Printf("[error] %v", err)
	}
}
