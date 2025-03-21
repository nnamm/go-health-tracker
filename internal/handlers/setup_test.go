package handlers

import (
	"os"
	"testing"
)

func TestMain(m *testing.M) {
	os.Setenv("ENV", "development")

	exitCode := m.Run()

	os.Unsetenv("ENV")

	os.Exit(exitCode)
}
