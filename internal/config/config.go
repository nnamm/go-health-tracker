package config

import (
	"os"
	"strconv"
)

// IsDevelopment is a flag to determine if the application is running in development mode
var IsDevelopment = os.Getenv("ENV") == "development"

// RequestTimeoutSecond is the default timeout for HTTP requests
var RequestTimeoutSecond = 30

// init initializes the configuration
func init() {
	if timeout := os.Getenv("REQUEST_TIMEOUT_SECONDS"); timeout != "" {
		if val, err := strconv.Atoi(timeout); err == nil {
			RequestTimeoutSecond = val
		}
	}
}
