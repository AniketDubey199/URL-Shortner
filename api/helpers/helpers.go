package helpers

import (
	"os"
	"strings"
)

func EnforceHTTP(url string) string {
	if url[:4] != "http" {
		return "http://" + url
	}
	return url
}

func RemoveDomainUrl(url string) bool {

	if url == os.Getenv("DOMAIN") {
		return false
	}
	newurl := strings.Replace(url, "http", "", 1)
	newurl = strings.Replace(newurl, "https", "", 1)
	newurl = strings.Replace(newurl, "www", "", 1)

	if newurl == os.Getenv("DOMAIN") {
		return false
	}

	return true

}
