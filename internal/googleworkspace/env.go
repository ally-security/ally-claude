package googleworkspace

import (
	"os"
	"strings"
)

func getenvService(serviceID, suffix string) string {
	if v := os.Getenv("GOOGLE_" + strings.ToUpper(serviceID) + "_" + suffix); v != "" {
		return v
	}
	return ""
}
