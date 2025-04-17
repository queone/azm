package maz

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httputil"
	"net/url"
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
	return ApiCall("GET", apiUrl, z, nil, params, false)
}

// ApiCall alias to do a GET with debugging on
func ApiGetVerbose(
	apiUrl string,
	z *Config,
	params map[string]string,
) (map[string]interface{}, int, error) {
	return ApiCall("GET", apiUrl, z, nil, params, true)
}

// ApiCall alias to do a PATCH
func ApiPatch(
	apiUrl string,
	z *Config,
	payload map[string]interface{},
	params map[string]string,
) (map[string]interface{}, int, error) {
	return ApiCall("PATCH", apiUrl, z, payload, params, false)
}

// ApiCall alias to do a PATCH with debugging on
func ApiPatchVerbose(
	apiUrl string,
	z *Config,
	payload map[string]interface{},
	params map[string]string,
) (map[string]interface{}, int, error) {
	return ApiCall("PATCH", apiUrl, z, payload, params, true)
}

// ApiCall alias to do a POST
func ApiPost(
	apiUrl string,
	z *Config,
	payload map[string]interface{},
	params map[string]string,
) (map[string]interface{}, int, error) {
	return ApiCall("POST", apiUrl, z, payload, params, false)
}

// ApiCall alias to do a POST with debugging on
func ApiPostVerbose(
	apiUrl string,
	z *Config,
	payload map[string]interface{},
	params map[string]string,
) (map[string]interface{}, int, error) {
	return ApiCall("POST", apiUrl, z, payload, params, true)
}

// ApiCall alias to do a PUT
func ApiPut(
	apiUrl string,
	z *Config,
	payload map[string]interface{},
	params map[string]string,
) (map[string]interface{}, int, error) {
	return ApiCall("PUT", apiUrl, z, payload, params, false)
}

// ApiCall alias to do a PUT with debugging on
func ApiPutVerbose(
	apiUrl string,
	z *Config,
	payload map[string]interface{},
	params map[string]string,
) (map[string]interface{}, int, error) {
	return ApiCall("PUT", apiUrl, z, payload, params, true)
}

// ApiCall alias to do a DELETE
func ApiDelete(
	apiUrl string,
	z *Config,
	params map[string]string,
) (map[string]interface{}, int, error) {
	return ApiCall("DELETE", apiUrl, z, nil, params, false)
}

// ApiCall alias to do a DELETE with debugging on
func ApiDeleteVerbose(
	apiUrl string,
	z *Config,
	params map[string]string,
) (map[string]interface{}, int, error) {
	return ApiCall("DELETE", apiUrl, z, nil, params, true)
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

// Prints HTTP headers specific to API calls. Simplifies ApiCall function.
func PrintHeaders(headers http.Header) {
	if headers == nil {
		return
	}
	fmt.Println(utl.Blu("headers") + ":")
	for k, v := range headers {
		fmt.Printf("  %s:\n", utl.Blu(k))
		count := len(v) // Array of string
		if count == 1 {
			fmt.Printf("    - %s\n", utl.Gre(string(v[0]))) // In YAML-like output, 1st entry gets the dash
		}
		if count > 2 {
			for _, i := range v[1:] {
				fmt.Printf("      %s\n", utl.Gre(string(i)))
			}
		}
	}
}

// Prints HTTP parameters specific to API calls. Simplifies ApiCall function.
func PrintParams(params url.Values) {
	if params == nil {
		return
	}
	fmt.Println(utl.Blu("params") + ":")
	for k, v := range params {
		fmt.Printf("  %s:\n", utl.Blu(k))
		count := len(v) // Array of string
		if count == 1 {
			fmt.Printf("    - %s\n", utl.Gre(string(v[0]))) // In YAML-like output, 1st entry gets the dash
		}
		if count > 2 {
			for _, i := range v[1:] {
				fmt.Printf("      %s\n", utl.Gre(string(i)))
			}
		}
	}
}

// Makes an API call and returns the result object, statusCode, and error.
func ApiCall(
	method string,
	apiUrl string,
	z *Config,
	payload map[string]interface{},
	params map[string]string,
	verbose bool,
) (map[string]interface{}, int, error) {
	Logf("%s %s\n", method, apiUrl) // Basic logging info

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

	// Log request details if verbose mode is enabled
	if verbose {
		logRequestDetails(method, apiUrl, req, payload, params)
	}

	client := &http.Client{Timeout: time.Second * 30} // Thirty second timeout
	resp, err := client.Do(req)
	if err != nil {
		// Note, this only captures network HTTP errors making the request, NOT errors
		// related to the actual API request itself. See next step for such details.
		Logf("%s\n", utl.Red2(fmt.Sprintf("Failed to execute API HTTP request: %s", err)))
		return nil, 0, fmt.Errorf("failed to execute API HTTP request: %w", err)
	}
	defer resp.Body.Close()

	result, err := processResponse(resp, verbose)
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

// Helper function to log request details in verbose mode
func logRequestDetails(
	method string,
	apiUrl string,
	req *http.Request,
	payload map[string]interface{},
	params map[string]string,
) {
	fmt.Println(utl.Blu("==== REQUEST ================================="))
	fmt.Println(method + " " + apiUrl)
	PrintHeaders(req.Header)
	PrintParams(req.URL.Query())
	if payload != nil {
		fmt.Println(utl.Blu("payload") + ":")
		utl.PrintJsonColor(payload)
	}
}

// Helper function to process the HTTP response
func processResponse(resp *http.Response, verbose bool) (map[string]interface{}, error) {
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

	// Log response details if verbose mode is enabled
	if verbose {
		logResponseDetails(resp, result)
	}

	return result, nil
}

// Helper function to log response details in verbose mode
func logResponseDetails(resp *http.Response, result map[string]interface{}) {
	fmt.Println(utl.Blu("==== RESPONSE ================================"))
	fmt.Printf("%s: %d %s\n", utl.Blu("status"), resp.StatusCode, http.StatusText(resp.StatusCode))
	fmt.Println(utl.Blu("result") + ":")
	utl.PrintJsonColor(result)
	resHeaders, err := httputil.DumpResponse(resp, false)
	if err != nil {
		fmt.Println("failed to dump response:", err)
		return
	}
	fmt.Println(utl.Blu("headers") + ":")
	fmt.Println(string(resHeaders))
}
