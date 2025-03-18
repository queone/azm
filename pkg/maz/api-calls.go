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

type JsonObject map[string]interface{} // Local syntactic sugar, for easier reading
type StringMap map[string]string

// ApiCall alias to do a GET
func ApiGet(apiUrl string, z *Config, params StringMap) (JsonObject, int, error) {
	return ApiCall("GET", apiUrl, z, nil, params, false) // false = quiet
}

// ApiCall alias to do a GET with debugging on
func ApiGetVerbose(apiUrl string, z *Config, params StringMap) (JsonObject, int, error) {
	return ApiCall("GET", apiUrl, z, nil, params, true) // true = verbose, for debugging
}

// ApiCall alias to do a PATCH
func ApiPatch(apiUrl string, z *Config, payload JsonObject, params StringMap) (JsonObject, int, error) {
	return ApiCall("PATCH", apiUrl, z, payload, params, false) // false = quiet
}

// ApiCall alias to do a PATCH with debugging on
func ApiPatchVerbose(apiUrl string, z *Config, payload JsonObject, params StringMap) (JsonObject, int, error) {
	return ApiCall("PATCH", apiUrl, z, payload, params, true) // true = verbose, for debugging
}

// ApiCall alias to do a POST
func ApiPost(apiUrl string, z *Config, payload JsonObject, params StringMap) (JsonObject, int, error) {
	return ApiCall("POST", apiUrl, z, payload, params, false) // false = quiet
}

// ApiCall alias to do a POST with debugging on
func ApiPostVerbose(apiUrl string, z *Config, payload JsonObject, params StringMap) (JsonObject, int, error) {
	return ApiCall("POST", apiUrl, z, payload, params, true) // true = verbose, for debugging
}

// ApiCall alias to do a PUT
func ApiPut(apiUrl string, z *Config, payload JsonObject, params StringMap) (JsonObject, int, error) {
	return ApiCall("PUT", apiUrl, z, payload, params, false) // false = quiet
}

// ApiCall alias to do a PUT with debugging on
func ApiPutVerbose(apiUrl string, z *Config, payload JsonObject, params StringMap) (JsonObject, int, error) {
	return ApiCall("PUT", apiUrl, z, payload, params, true) // true = verbose, for debugging
}

// ApiCall alias to do a DELETE
func ApiDelete(apiUrl string, z *Config, params StringMap) (JsonObject, int, error) {
	return ApiCall("DELETE", apiUrl, z, nil, params, false) // false = quiet
}

// ApiCall alias to do a DELETE with debugging on
func ApiDeleteVerbose(apiUrl string, z *Config, params StringMap) (JsonObject, int, error) {
	return ApiCall("DELETE", apiUrl, z, nil, params, true) // true = verbose, for debugging
}

// Makes an API call and returns the result object, statusCode, and error. For a more clear
// explanation of how to interpret the JSON responses see https://eager.io/blog/go-and-json/
// This function is the cornerstone of the maz package, extensively handling all API interactions.
func ApiCall(method, apiUrl string, z *Config, payload JsonObject, params StringMap, verbose bool) (JsonObject, int, error) {
	if !strings.HasPrefix(apiUrl, "http") {
		return nil, 0, fmt.Errorf("%s Error: Bad URL, %s", utl.Trace(), apiUrl)
	}

	// Map headers to corresponding API endpoint
	var headers StringMap
	if strings.HasPrefix(apiUrl, ConstMgUrl) {
		headers = z.MgHeaders
	} else if strings.HasPrefix(apiUrl, ConstAzUrl) {
		headers = z.AzHeaders
	}

	// Set up new HTTP request client
	client := &http.Client{Timeout: time.Second * 60} // One minute timeout
	var req *http.Request
	var err error
	switch strings.ToUpper(method) {
	case "GET":
		req, err = http.NewRequest("GET", apiUrl, nil)
	case "PATCH":
		jsonData, err2 := json.Marshal(payload)
		if err2 != nil {
			return nil, 0, fmt.Errorf("failed to marshal payload: %w", err2)
		}
		req, err = http.NewRequest("PATCH", apiUrl, bytes.NewBuffer(jsonData))
	case "POST":
		jsonData, err2 := json.Marshal(payload)
		if err2 != nil {
			return nil, 0, fmt.Errorf("failed to marshal payload: %w", err2)
		}
		req, err = http.NewRequest("POST", apiUrl, bytes.NewBuffer(jsonData))
	case "PUT":
		jsonData, err2 := json.Marshal(payload)
		if err2 != nil {
			return nil, 0, fmt.Errorf("failed to marshal payload: %w", err2)
		}
		req, err = http.NewRequest("PUT", apiUrl, bytes.NewBuffer(jsonData))
	case "DELETE":
		req, err = http.NewRequest("DELETE", apiUrl, nil)
	default:
		return nil, 0, fmt.Errorf("%s Error: Unsupported HTTP method", utl.Trace())
	}
	if err != nil {
		return nil, 0, fmt.Errorf("failed to create HTTP request: %w", err)
	}

	// Set up the headers
	for h, v := range headers {
		req.Header.Add(h, v)
	}

	// Set up the query parameters and encode
	reqParams := req.URL.Query()
	for p, v := range params {
		reqParams.Add(p, v)
	}
	req.URL.RawQuery = reqParams.Encode()

	// === MAKE THE CALL ============
	if verbose {
		fmt.Println(utl.Blu("==== REQUEST ================================="))
		fmt.Println(method + " " + apiUrl)
		PrintHeaders(req.Header)
		PrintParams(reqParams)
		if payload != nil {
			fmt.Println(utl.Blu("payload") + ":")
			utl.PrintJsonColor(payload)
		}
	}
	// Make the call
	resp, err := client.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to execute HTTP request: %w", err)
	}
	defer resp.Body.Close()

	// Read the response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to read response body: %w", err)
	}

	// This function caters to Microsoft Azure REST API calls. Note that variable 'body' is of type
	// []uint8, which is essentially a long string that evidently can be either: 1) a single integer
	// number, or 2) a JSON object string that needs unmarshalling. Below conditional is based on
	// this interpretation, but may need confirmation and improved handling.

	var result JsonObject // JSON object to be returned
	if intValue, err := strconv.ParseInt(string(body), 10, 64); err == nil {
		// It's an integer, probably an API object count value
		result = make(map[string]interface{})
		result["value"] = intValue
	} else {
		// It's a regular JSON result, or null
		if len(body) > 0 { // Make sure we have something to unmarshal, else return nil
			if err = json.Unmarshal(body, &result); err != nil {
				return nil, 0, fmt.Errorf("failed to unmarshal response body: %w", err)
			}
		}
		// If it's null, returning r.StatusCode below will let caller know
	}

	statCode := resp.StatusCode
	if verbose {
		fmt.Println(utl.Blu("==== RESPONSE ================================"))
		fmt.Printf("%s: %d %s\n", utl.Blu("status"), statCode, http.StatusText(statCode))
		fmt.Println(utl.Blu("result") + ":")
		utl.PrintJsonColor(result)
		resHeaders, err := httputil.DumpResponse(resp, false)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to dump response: %w", err)
		}
		fmt.Println(utl.Blu("headers") + ":")
		fmt.Println(string(resHeaders))
	}

	return result, statCode, nil
}

// Returns API error message string.
func ApiErrorMsg(obj map[string]interface{}) string {
	if err, ok := obj["error"].(map[string]interface{}); ok {
		msg, msgOk := err["message"].(string)
		if !msgOk {
			msg = "Unknown message"
		}
		return msg
	}
	return ""
}

// Checks for errors in API results and prints them out.
func CheckApiError(trace utl.TraceInfo, result map[string]interface{}, statusCode int, err error) {
	caller := fmt.Sprintf("%s\n  %s:%d", trace.FuncName, trace.File, trace.Line)
	apiError := result["error"] != nil || (300 <= statusCode && statusCode <= 599)
	if apiError {
		e := result["error"].(map[string]interface{})
		eMsg := e["message"].(string)
		msg := fmt.Sprintf("%s\n    HTTP %d : %s", caller, statusCode, eMsg)
		fmt.Printf("%s\n", utl.Yel(msg))
	}
}

// Prints API error messages in 2 parts separated by a newline: A header, then a JSON byte slice
func PrintApiErrMsg(msg string) {
	parts := strings.Split(msg, "\n")
	fmt.Println(utl.Red(parts[0])) // Print error header

	// Check if there is a second part
	if len(parts) > 1 {
		errorBytes := []byte(parts[1])
		yamlError, _ := utl.BytesToYamlObject(errorBytes)
		utl.PrintYamlColor(yamlError) // Print error
		// errorMsg, _ := utl.JsonBytesReindent(errorBytes, 2)
		// utl.PrintYamlBytesColor(errorMsg) // Print error
	} else {
		// Handle the case where there is no second part
		fmt.Println(utl.Red("No error details available."))
	}
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
