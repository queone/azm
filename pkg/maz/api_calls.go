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
func ApiGet(apiUrl string, z *Config, params map[string]string) (map[string]interface{}, int, error) {
	return ApiCall("GET", apiUrl, z, nil, params, false) // false = quiet
}

// ApiCall alias to do a GET with debugging on
func ApiGetVerbose(apiUrl string, z *Config, params map[string]string) (map[string]interface{}, int, error) {
	return ApiCall("GET", apiUrl, z, nil, params, true) // true = verbose, for debugging
}

// ApiCall alias to do a PATCH
func ApiPatch(apiUrl string, z *Config, payload map[string]interface{}, params map[string]string) (map[string]interface{}, int, error) {
	return ApiCall("PATCH", apiUrl, z, payload, params, false) // false = quiet
}

// ApiCall alias to do a PATCH with debugging on
func ApiPatchVerbose(apiUrl string, z *Config, payload map[string]interface{}, params map[string]string) (map[string]interface{}, int, error) {
	return ApiCall("PATCH", apiUrl, z, payload, params, true) // true = verbose, for debugging
}

// ApiCall alias to do a POST
func ApiPost(apiUrl string, z *Config, payload map[string]interface{}, params map[string]string) (map[string]interface{}, int, error) {
	return ApiCall("POST", apiUrl, z, payload, params, false) // false = quiet
}

// ApiCall alias to do a POST with debugging on
func ApiPostVerbose(apiUrl string, z *Config, payload map[string]interface{}, params map[string]string) (map[string]interface{}, int, error) {
	return ApiCall("POST", apiUrl, z, payload, params, true) // true = verbose, for debugging
}

// ApiCall alias to do a PUT
func ApiPut(apiUrl string, z *Config, payload map[string]interface{}, params map[string]string) (map[string]interface{}, int, error) {
	return ApiCall("PUT", apiUrl, z, payload, params, false) // false = quiet
}

// ApiCall alias to do a PUT with debugging on
func ApiPutVerbose(apiUrl string, z *Config, payload map[string]interface{}, params map[string]string) (map[string]interface{}, int, error) {
	return ApiCall("PUT", apiUrl, z, payload, params, true) // true = verbose, for debugging
}

// ApiCall alias to do a DELETE
func ApiDelete(apiUrl string, z *Config, params map[string]string) (map[string]interface{}, int, error) {
	return ApiCall("DELETE", apiUrl, z, nil, params, false) // false = quiet
}

// ApiCall alias to do a DELETE with debugging on
func ApiDeleteVerbose(apiUrl string, z *Config, params map[string]string) (map[string]interface{}, int, error) {
	return ApiCall("DELETE", apiUrl, z, nil, params, true) // true = verbose, for debugging
}

// // Makes an API call and returns the result object, statusCode, and error.
// func ApiCall(method, apiUrl string, z *Config, payload map[string]interface{}, params map[string]string, verbose bool) (map[string]interface{}, int, error) {
// 	if !strings.HasPrefix(apiUrl, "http") {
// 		return nil, 0, fmt.Errorf("%s Error: Bad URL, %s", utl.Trace(), apiUrl)
// 	}

// 	// Map headers to corresponding API endpoint
// 	var headers map[string]string
// 	if strings.HasPrefix(apiUrl, ConstMgUrl) {
// 		headers = z.MgHeaders
// 	} else if strings.HasPrefix(apiUrl, ConstAzUrl) {
// 		headers = z.AzHeaders
// 	}

// 	// Set up new HTTP request client
// 	client := &http.Client{Timeout: time.Second * 60} // One minute timeout
// 	var req *http.Request
// 	var err error
// 	switch strings.ToUpper(method) {
// 	case "GET":
// 		req, err = http.NewRequest("GET", apiUrl, nil)
// 	case "PATCH":
// 		jsonData, err2 := json.Marshal(payload)
// 		if err2 != nil {
// 			return nil, 0, fmt.Errorf("failed to marshal payload: %w", err2)
// 		}
// 		req, err = http.NewRequest("PATCH", apiUrl, bytes.NewBuffer(jsonData))
// 	case "POST":
// 		jsonData, err2 := json.Marshal(payload)
// 		if err2 != nil {
// 			return nil, 0, fmt.Errorf("failed to marshal payload: %w", err2)
// 		}
// 		req, err = http.NewRequest("POST", apiUrl, bytes.NewBuffer(jsonData))
// 	case "PUT":
// 		jsonData, err2 := json.Marshal(payload)
// 		if err2 != nil {
// 			return nil, 0, fmt.Errorf("failed to marshal payload: %w", err2)
// 		}
// 		req, err = http.NewRequest("PUT", apiUrl, bytes.NewBuffer(jsonData))
// 	case "DELETE":
// 		req, err = http.NewRequest("DELETE", apiUrl, nil)
// 	default:
// 		return nil, 0, fmt.Errorf("%s Error: Unsupported HTTP method", utl.Trace())
// 	}
// 	if err != nil {
// 		return nil, 0, fmt.Errorf("failed to create HTTP request: %w", err)
// 	}

// 	// Set up the headers
// 	for h, v := range headers {
// 		req.Header.Add(h, v)
// 	}

// 	// Set up the query parameters and encode
// 	reqParams := req.URL.Query()
// 	for p, v := range params {
// 		reqParams.Add(p, v)
// 	}
// 	req.URL.RawQuery = reqParams.Encode()

// 	// === MAKE THE CALL ============
// 	if verbose {
// 		fmt.Println(utl.Blu("==== REQUEST ================================="))
// 		fmt.Println(method + " " + apiUrl)
// 		PrintHeaders(req.Header)
// 		PrintParams(reqParams)
// 		if payload != nil {
// 			fmt.Println(utl.Blu("payload") + ":")
// 			utl.PrintJsonColor(payload)
// 		}
// 	}
// 	// Make the call
// 	resp, err := client.Do(req)
// 	if err != nil {
// 		return nil, 0, fmt.Errorf("failed to execute HTTP request: %w", err)
// 	}
// 	defer resp.Body.Close()

// 	// Read the response body
// 	body, err := io.ReadAll(resp.Body)
// 	if err != nil {
// 		return nil, 0, fmt.Errorf("failed to read response body: %w", err)
// 	}

// 	// This function caters to Microsoft Azure REST API calls. Note that variable 'body' is of type
// 	// []uint8, which is essentially a long string that evidently can be either: 1) a single integer
// 	// number, or 2) a JSON object string that needs unmarshalling. Below conditional is based on
// 	// this interpretation, but may need further confirmation and improved handling.

// 	var result map[string]interface{} // JSON object to be returned
// 	if intValue, err := strconv.ParseInt(string(body), 10, 64); err == nil {
// 		// It's an integer, probably an API object count value
// 		result = make(map[string]interface{})
// 		result["value"] = intValue
// 	} else {
// 		// It's a regular JSON result, or null
// 		if len(body) > 0 { // Make sure we have something to unmarshal, else return nil
// 			if err = json.Unmarshal(body, &result); err != nil {
// 				return nil, 0, fmt.Errorf("failed to unmarshal response body: %w", err)
// 			}
// 		}
// 		// If it's null, returning r.StatusCode below will let caller know
// 	}

// 	statCode := resp.StatusCode
// 	if verbose {
// 		fmt.Println(utl.Blu("==== RESPONSE ================================"))
// 		fmt.Printf("%s: %d %s\n", utl.Blu("status"), statCode, http.StatusText(statCode))
// 		fmt.Println(utl.Blu("result") + ":")
// 		utl.PrintJsonColor(result)
// 		resHeaders, err := httputil.DumpResponse(resp, false)
// 		if err != nil {
// 			return nil, 0, fmt.Errorf("failed to dump response: %w", err)
// 		}
// 		fmt.Println(utl.Blu("headers") + ":")
// 		fmt.Println(string(resHeaders))
// 	}

// 	return result, statCode, nil
// }

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

// =========================================================================
// Makes an API call and returns the result object, statusCode, and error.
func ApiCall(method, apiUrl string, z *Config, payload map[string]interface{}, params map[string]string, verbose bool) (map[string]interface{}, int, error) {
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
func logRequestDetails(method, apiUrl string, req *http.Request, payload map[string]interface{}, params map[string]string) {
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

// ==========================================================================
// TODO: Make the alias functions more readable
//
// 1. Use Structs for Complex Parameters
//
//	type ApiCallOptions struct {
//		Method  string
//		Url     string
//		Config  *Config
//		Payload map[string]interface{}
//		Params  map[string]string
//		Verbose bool
//	}
//
//	func ApiCall(opts ApiCallOptions) (map[string]interface{}, int, error) {
//		// Function implementation
//	}
//
// // Caller
//
//	opts := ApiCallOptions{
//	    Method:  "GET",
//	    Url:     "https://api.example.com",
//	    Config:  z,
//	    Payload: nil,
//	    Params:  map[string]string{"param1": "value1"},
//	    Verbose: true,
//	}
//
// result, statusCode, err := ApiCall(opts)
//
// 2. Use Type Definitions for Return Values
//
// type ApiResponse struct {
// 	Result     map[string]interface{}
// 	StatusCode int
// 	Error      error
// }
// func ApiCall(method, apiUrl string, z *Config, payload map[string]interface{}, params map[string]string, verbose bool) ApiResponse {
// 	// Function implementation
// }
// response := ApiCall("GET", "https://api.example.com", z, nil, nil, true)
// if response.Error != nil {
//     fmt.Println("Error:", response.Error)
// } else {
//     fmt.Println("Result:", response.Result)
// }
//
// 3. Break Down Large Functions
//
// func BuildRequest(method, apiUrl string, payload map[string]interface{}, params map[string]string) (*http.Request, error) {
//     // Build and return the HTTP request
// }

// func SendRequest(req *http.Request, verbose bool) (map[string]interface{}, int, error) {
//     // Send the HTTP request and return the response
// }

// func ApiCall(method, apiUrl string, z *Config, payload map[string]interface{}, params map[string]string, verbose bool) (map[string]interface{}, int, error) {
//     req, err := BuildRequest(method, apiUrl, payload, params)
//     if err != nil {
//         return nil, 0, err
//     }
//     return SendRequest(req, verbose)
// }
