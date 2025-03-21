package config

import (
	"os"
	"testing"
)

func TestDevelopment(t *testing.T) {
	orgEnv, exists := os.LookupEnv("ENV")

	defer func() {
		if exists {
			os.Setenv("ENV", orgEnv)
		} else {
			os.Unsetenv("ENV")
		}
	}()

	tests := []struct {
		name string
		env  string
		want bool
	}{
		{"developmen env", "development", true},
		{"production env", "production", false},
		{"unset", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.env == "" {
				os.Unsetenv("ENV")
			} else {
				os.Setenv("ENV", tt.env)
			}

			// 環境変数を変更した後、ReloadConfig()を呼び出して設定を再読み込みする
			ReloadConfig()

			if IsDevelopment != tt.want {
				t.Errorf("IsDevelopment = %v, want %v", IsDevelopment, tt.want)
			}
		})
	}
}

func TestRequestTimeoutSecond(t *testing.T) {
	orgTimeout, timeoutExists := os.LookupEnv("REQUEST_TIMEOUT_SECONDS")

	defer func() {
		if timeoutExists {
			os.Setenv("REQUEST_TIMEOUT_SECONDS", orgTimeout)
		} else {
			os.Unsetenv("REQUEST_TIMEOUT_SECONDS")
		}
	}()

	tests := []struct {
		name    string
		timeout string
		want    int
	}{
		{"with timeout specified", "60", 60},
		{"invalid value", "invalid", 30}, // back to default value
		{"unset", "", 30},                // default value
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.timeout == "" {
				os.Unsetenv("REQUEST_TIMEOUT_SECONDS")
			} else {
				os.Setenv("REQUEST_TIMEOUT_SECONDS", tt.timeout)
			}

			// 環境変数を変更した後、ReloadConfig()を呼び出して設定を再読み込みする
			// これにより、重複したコードが不要になり、実際の実装をテストすることができる
			RequestTimeoutSecond = 30 // デフォルト値に戻す
			ReloadConfig()

			if RequestTimeoutSecond != tt.want {
				t.Errorf("RequestTimeoutSecond = %v, want %v", RequestTimeoutSecond, tt.want)
			}
		})
	}
}
