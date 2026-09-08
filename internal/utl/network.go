package utl

import (
	"net/http"
	"time"
)

// List of reliable URLs for checking internet connectivity
var reliableURLs = []string{
	"https://dns.google",
	"https://1.1.1.1",
	"https://httpbin.org/ip",
	"https://github.com",
	"https://www.wikipedia.org",
}

// Returns true if given string is a valid IP address. False otherwise.

// Checks if IP_Address:Port string is reachable

// Checks if internet is available by attempting to reach reliable URLs.
func IsInternetAvailable() bool {
	client := http.Client{
		Timeout: 3 * time.Second, // Set a timeout for the request
	}

	for _, url := range reliableURLs {
		resp, err := client.Get(url)
		if err == nil {
			defer resp.Body.Close()               // Ensure the response body is closed
			if resp.StatusCode == http.StatusOK { // Check for a successful response
				return true
			}
		}
	}
	return false // Return false if none of the URLs were reachable
}

// Checks if internet is not available.
