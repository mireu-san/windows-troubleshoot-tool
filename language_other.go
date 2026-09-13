//go:build !windows

package main

import (
	"os"
	"strings"
)

func systemLanguage() string {
	for _, key := range []string{"LC_ALL", "LC_MESSAGES", "LANG"} {
		value := strings.ToLower(os.Getenv(key))
		if value == "" {
			continue
		}
		if strings.HasPrefix(value, "ko") {
			return "ko"
		}
		if strings.HasPrefix(value, "ja") {
			return "ja"
		}
		return "en"
	}
	return "en"
}
