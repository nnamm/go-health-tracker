package config

import (
	"os"
	"strconv"
)

// IsDevelopment is a flag to determine if the application is running in development mode
var IsDevelopment bool

// RequestTimeoutSecond is the default timeout for HTTP requests
var RequestTimeoutSecond = 30

// IsDev returns true if the application is running in development mode
func IsDev() bool {
	return os.Getenv("ENV") == "development"
}

// reloadConfig reloads the configuration from environment variables
func ReloadConfig() {
	IsDevelopment = IsDev()

	if timeout := os.Getenv("REQUEST_TIMEOUT_SECONDS"); timeout != "" {
		if val, err := strconv.Atoi(timeout); err == nil {
			RequestTimeoutSecond = val
		}
	}
}

// init initializes the configuration
func init() {
	ReloadConfig()
}
