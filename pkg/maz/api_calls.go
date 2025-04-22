package maz

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httputil"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/queone/utl"
)

// ApiCall alias to do a GET
func ApiGet(
	apiUrl string,
	z *Config,
	params map[string]string,
) (map[string]interface{}, int, error) {
	return ApiCall("GET", apiUrl, z, nil, params)
}

// ApiCall alias to do a PATCH
func ApiPatch(
	apiUrl string,
	z *Config,
	payload map[string]interface{},
	params map[string]string,
) (map[string]interface{}, int, error) {
	return ApiCall("PATCH", apiUrl, z, payload, params)
}

// ApiCall alias to do a POST
func ApiPost(
	apiUrl string,
	z *Config,
	payload map[string]interface{},
	params map[string]string,
) (map[string]interface{}, int, error) {
	return ApiCall("POST", apiUrl, z, payload, params)
}

// ApiCall alias to do a PUT
func ApiPut(
	apiUrl string,
	z *Config,
	payload map[string]interface{},
	params map[string]string,
) (map[string]interface{}, int, error) {
	return ApiCall("PUT", apiUrl, z, payload, params)
}

// ApiCall alias to do a DELETE
func ApiDelete(
	apiUrl string,
	z *Config,
	params map[string]string,
) (map[string]interface{}, int, error) {
	return ApiCall("DELETE", apiUrl, z, nil, params)
}

// Makes an API call and returns the result object, statusCode, and error.
func ApiCall(
	method string,
	apiUrl string,
	z *Config,
	payload map[string]interface{},
	params map[string]string,
) (map[string]interface{}, int, error) {
	// Validate URL
	if !strings.HasPrefix(apiUrl, "http") {
		Logf("%s\n", utl.Red2("Error: Bad URL"))
		return nil, 0, fmt.Errorf("%s error: Bad URL, %s", utl.Trace(), apiUrl)
	}

	// Set headers based on the API URL
	headers := getHeadersForApi(apiUrl, z)

	// Create HTTP request
	req, err := createHttpRequest(method, apiUrl, payload)
	if err != nil {
		Logf("%s\n", utl.Red2(fmt.Sprintf("Failed to create HTTP request: %s", err)))
		return nil, 0, fmt.Errorf("failed to create HTTP request: %w", err)
	}

	// Add headers and query parameters to the request
	setRequestHeaders(req, headers)
	setQueryParameters(req, params)

	logRequestDetails(req, payload, params)

	client := &http.Client{Timeout: time.Second * 30} // Thirty second timeout
	resp, err := client.Do(req)
	if err != nil {
		// Note, this only captures network HTTP errors making the request, NOT errors
		// related to the actual API request itself. See next step for such details.
		Logf("%s\n", utl.Red2(fmt.Sprintf("Failed to execute API HTTP request: %s", err)))
		return nil, 0, fmt.Errorf("failed to execute API HTTP request: %w", err)
	}
	defer resp.Body.Close()

	result, err := processResponse(resp)
	if err != nil {
		Logf("%s\n", utl.Red2(fmt.Sprintf("Failed to process API response: %s", err)))
		return nil, 0, fmt.Errorf("failed to process API response: %w", err)
	}

	return result, resp.StatusCode, nil
}

// Helper function to get headers based on the API URL
func getHeadersForApi(apiUrl string, z *Config) map[string]string {
	if strings.HasPrefix(apiUrl, ConstMgUrl) {
		return z.MgHeaders
	} else if strings.HasPrefix(apiUrl, ConstAzUrl) {
		return z.AzHeaders
	}
	return nil
}

// Helper function to create an HTTP request
func createHttpRequest(method, apiUrl string, payload map[string]interface{}) (*http.Request, error) {
	switch strings.ToUpper(method) {
	case "GET":
		return http.NewRequest("GET", apiUrl, nil)
	case "PATCH", "POST", "PUT":
		jsonData, err := json.Marshal(payload)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal payload: %w", err)
		}
		return http.NewRequest(strings.ToUpper(method), apiUrl, bytes.NewBuffer(jsonData))
	case "DELETE":
		return http.NewRequest("DELETE", apiUrl, nil)
	default:
		return nil, fmt.Errorf("%s Error: Unsupported HTTP method", utl.Trace())
	}
}

// Helper function to set headers on the request
func setRequestHeaders(req *http.Request, headers map[string]string) {
	for h, v := range headers {
		req.Header.Add(h, v)
	}
}

// Helper function to set query parameters on the request
func setQueryParameters(req *http.Request, params map[string]string) {
	reqParams := req.URL.Query()
	for p, v := range params {
		reqParams.Add(p, v)
	}
	req.URL.RawQuery = reqParams.Encode()
}

// Helper function to log request details
func logRequestDetails(req *http.Request, payload map[string]interface{}, params map[string]string) {
	var b strings.Builder

	// Add method and URL line
	b.WriteString("REQUEST\n")

	// Clone the request to avoid modifying the original
	clonedReq := req.Clone(context.Background())

	// Partially redact Authorization header (show first 4 and last 4 chars)
	if authHeader := clonedReq.Header.Get("Authorization"); authHeader != "" {
		redactedAuth := partiallyRedactToken(authHeader)
		clonedReq.Header.Set("Authorization", redactedAuth)
	}

	// Dump and trim request headers (now with redacted auth)
	if reqDump, err := httputil.DumpRequest(clonedReq, false); err == nil {
		rawHeaders := strings.TrimRight(string(reqDump), "\n")
		b.WriteString(rawHeaders)
		b.WriteString("\n")
	} else {
		b.WriteString(fmt.Sprintf("Failed to dump REQUEST headers: %s\n", err))
	}

	// Log full payload if present
	if payload != nil {
		if jsonBytes, err := utl.JsonToBytesIndent(payload, 2); err == nil {
			b.WriteString("Request payload:\n")
			b.WriteString(string(jsonBytes))
			b.WriteString("\n")
		} else {
			b.WriteString(fmt.Sprintf("Failed to print request payload: %s\n", err))
		}
	}

	// Log query parameters if present
	if len(params) > 0 {
		b.WriteString("Query parameters:\n")
		for k, v := range params {
			b.WriteString(fmt.Sprintf("  %s: %s\n", k, v))
		}
	}

	Logf("%s", b.String()) // Single Logf call
}

// Helper function to partially redact a token (shows first 10 and last 4 chars)
func partiallyRedactToken(token string) string {
	if len(token) <= 8 {
		return "***__<REDACTED>__***" // Too short to split meaningfully
	}
	firstPart := token[:10]
	lastPart := token[len(token)-4:]
	return firstPart + "__<REDACTED>__" + lastPart
}

// Helper function to process the HTTP response
func processResponse(resp *http.Response) (map[string]interface{}, error) {
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var result map[string]interface{}
	if intValue, err := strconv.ParseInt(string(body), 10, 64); err == nil {
		// It's an integer, probably an API object count value
		result = make(map[string]interface{})
		result["value"] = intValue
	} else if len(body) > 0 {
		// It's a regular JSON result
		if err := json.Unmarshal(body, &result); err != nil {
			return nil, fmt.Errorf("failed to unmarshal response body: %w", err)
		}
	}

	logResponseDetails(resp, result)

	return result, nil
}

// Helper function to log response details
func logResponseDetails(resp *http.Response, result map[string]interface{}) {
	var b strings.Builder

	// Dump and trim response headers
	if resHeaders, err := httputil.DumpResponse(resp, false); err == nil {
		trimmed := strings.TrimRight(string(resHeaders), "\n")
		b.WriteString("RESPONSE\n")
		b.WriteString(trimmed)
		b.WriteString("\n")
	} else {
		b.WriteString(fmt.Sprintf("RESPONSE\nFailed to dump headers: %s\n", err))
	}

	// Build a simplified version of the result
	simplified := make(map[string]interface{})
	for k, v := range result {
		switch val := v.(type) {
		case []interface{}:
			simplified[k] = fmt.Sprintf("[%d items]", len(val))
		case map[string]interface{}:
			simplified[k] = "{...}"
		default:
			// Special case for @odata.deltaLink abbreviation
			if k == "@odata.deltaLink" {
				strVal := utl.Str(val)
				re := regexp.MustCompile(`(?i)/delta\?(.{20})`)
				if matches := re.FindStringSubmatch(strVal); len(matches) == 2 {
					simplified[k] = fmt.Sprintf(".../delta?%s{__ABRIDGED__}...", matches[1])
					break
				}
			}
			simplified[k] = val
		}
	}

	// Format simplified JSON and append to buffer
	if pretty, err := utl.JsonToBytesIndent(simplified, 2); err == nil {
		b.WriteString("Response body is proper JSON. Showing top-level structure only:\n")
		b.WriteString(string(pretty))
		b.WriteString("\n")
	} else {
		b.WriteString(fmt.Sprintf("Failed to summarize response JSON body: %s\n", err))
	}

	// Single Logf call
	Logf("%s", b.String())
}

// Extracts API error message as "<code> <message>".
func ApiErrorMsg(obj map[string]interface{}) string {
	err := utl.Map(obj["error"])
	if err == nil {
		return ""
	}

	// Prefer details[0] if available
	if details := utl.Slice(err["details"]); len(details) > 0 {
		if d := utl.Map(details[0]); d != nil {
			code, msg := utl.Str(d["code"]), utl.Str(d["message"])
			if code != "" && msg != "" {
				return fmt.Sprintf("%s: %s", code, msg)
			}
		}
	}

	// Fall back to top-level error
	code, msg := utl.Str(err["code"]), utl.Str(err["message"])
	if code != "" || msg != "" {
		return fmt.Sprintf("%s: %s", code, msg)
	}

	return ""
}
