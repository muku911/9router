package config

import (
	"fmt"
	"runtime"
)

const (
	GoogleTokenURL   = "https://oauth2.googleapis.com/token"
	AntigravityBase  = "https://daily-cloudcode-pa.googleapis.com"
	UserAgentVersion = "1.107.0"
)

var (
	// Internal Request Header names
	InternalRequestHeaderName  = "X-Goog-Api-Client"
	InternalRequestHeaderValue = "cloudcode/1.107.0 client/antigravity"
)

func GetUserAgent() string {
	return fmt.Sprintf("antigravity/%s %s/%s", UserAgentVersion, runtime.GOOS, runtime.GOARCH)
}
