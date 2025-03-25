package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"regexp"
	"strings"
	"unicode"

	"gopkg.in/yaml.v3"
)

type RoleAssignment struct {
	Properties struct {
		PrincipalId      string `yaml:"principalId"`
		RoleDefinitionId string `yaml:"roleDefinitionId"`
		Scope            string `yaml:"scope"`
	} `yaml:"properties"`
}

// sanitizeName converts a name to lowercase and replaces spaces with hyphens
func sanitizeName(name string) string {
	name = strings.TrimSpace(name)
	name = strings.ToLower(name)
	return strings.ReplaceAll(name, " ", "-")
}

// getFromGraph makes a GET request to Microsoft Graph and extracts the displayName
func getFromGraph(url, token string) string {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return ""
	}

	req.Header.Add("Authorization", "Bearer "+token)
	req.Header.Add("ConsistencyLevel", "eventual")

	resp, err := http.DefaultClient.Do(req)
	if err != nil || resp.StatusCode != http.StatusOK {
		return ""
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return ""
	}

	// Parse the JSON response
	var result struct {
		DisplayName       string `json:"displayName"`
		UserPrincipalName string `json:"userPrincipalName"`
		Mail              string `json:"mail"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return ""
	}

	// Prefer displayName, fall back to userPrincipalName or mail
	if result.DisplayName != "" {
		return sanitizeName(result.DisplayName)
	}
	if result.UserPrincipalName != "" {
		return sanitizeName(strings.Split(result.UserPrincipalName, "@")[0])
	}
	if result.Mail != "" {
		return sanitizeName(strings.Split(result.Mail, "@")[0])
	}

	return ""
}

// getPrincipalName attempts to get a principal name from Microsoft Graph API
func getPrincipalName(targetId string) string {
	// First check if we have an access token
	token := os.Getenv("MG_TOKEN")
	if token == "" {
		return "UNKNOWN"
	}

	// Try user endpoint first
	if name := getFromGraph(fmt.Sprintf("https://graph.microsoft.com/v1.0/users/%s", targetId), token); name != "" {
		return name
	}

	// Try group endpoint next
	if name := getFromGraph(fmt.Sprintf("https://graph.microsoft.com/v1.0/groups/%s", targetId), token); name != "" {
		return name
	}

	// Try service principal last
	if name := getFromGraph(fmt.Sprintf("https://graph.microsoft.com/v1.0/servicePrincipals/%s", targetId), token); name != "" {
		return name
	}

	return "UNKNOWN"
}

// getSubscriptionName gets the subscription display name from Azure Management API
func getSubscriptionName(targetId string) string {
	// Get Azure token from environment
	token := os.Getenv("AZ_TOKEN")
	if token == "" {
		return "UNKNOWN"
	}

	// Create request to Azure Management API
	url := fmt.Sprintf("https://management.azure.com/subscriptions/%s?api-version=2016-06-01", targetId)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "UNKNOWN"
	}

	// Add authorization header
	req.Header.Add("Authorization", "Bearer "+token)

	// Make the request
	resp, err := http.DefaultClient.Do(req)
	if err != nil || resp.StatusCode != http.StatusOK {
		return "UNKNOWN"
	}
	defer resp.Body.Close()

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "UNKNOWN"
	}

	// Parse the JSON response
	var result struct {
		DisplayName string `json:"displayName"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return "UNKNOWN"
	}

	// Return sanitized name or UNKNOWN if empty
	if result.DisplayName == "" {
		return "UNKNOWN"
	}
	return sanitizeName(result.DisplayName)
}

// getRoleDefinitionName gets the role definition name from Azure Authorization API
func getRoleDefinitionName(targetId string) string {
	// Get Azure token from environment
	token := os.Getenv("AZ_TOKEN")
	if token == "" {
		return "UNKNOWN"
	}

	// Create request to Azure Authorization API
	url := fmt.Sprintf("https://management.azure.com/providers/Microsoft.Authorization/roleDefinitions/%s?api-version=2022-04-01", targetId)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "UNKNOWN"
	}

	// Add authorization header
	req.Header.Add("Authorization", "Bearer "+token)

	// Make the request
	resp, err := http.DefaultClient.Do(req)
	if err != nil || resp.StatusCode != http.StatusOK {
		return "UNKNOWN"
	}
	defer resp.Body.Close()

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "UNKNOWN"
	}

	// Parse the JSON response
	var result struct {
		Properties struct {
			RoleName string `json:"roleName"`
		} `json:"properties"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return "UNKNOWN"
	}

	// Return sanitized name or UNKNOWN if empty
	if result.Properties.RoleName == "" {
		return "UNKNOWN"
	}
	return sanitizeName(result.Properties.RoleName)
}

func isRootManagementGroup(scope string) bool {
	return strings.HasPrefix(scope, "/providers/Microsoft.Management/managementGroups/") &&
		regexp.MustCompile(`[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`).MatchString(scope)
}

func sanitizePart(s string) string {
	s = strings.TrimSpace(s)
	var sb strings.Builder
	prevHyphen := false

	for _, r := range s {
		switch {
		case unicode.IsLetter(r) || unicode.IsDigit(r):
			sb.WriteRune(unicode.ToLower(r))
			prevHyphen = false
		case unicode.IsSpace(r), !unicode.IsPrint(r):
			if !prevHyphen {
				sb.WriteRune('-')
				prevHyphen = true
			}
		default:
			if !prevHyphen {
				sb.WriteRune('-')
				prevHyphen = true
			}
		}
	}

	return strings.Trim(sb.String(), "-")
}

// getScopeName determines the appropriate name part based on the scope
func getScopeName(scope string) string {
	switch {
	case isRootManagementGroup(scope):
		return "mg-root"
	case strings.HasPrefix(scope, "/subscriptions/"):
		subId := path.Base(scope)
		return sanitizePart(getSubscriptionName(subId))
	default:
		return sanitizePart(scope)
	}
}

// Update generateFilename to use getScopeName
func generateFilename(ra RoleAssignment) (string, error) {
	part1 := getScopeName(ra.Properties.Scope)
	if part1 == "" {
		return "", errors.New("invalid scope")
	}

	part2 := sanitizePart(getPrincipalName(ra.Properties.PrincipalId))
	part3 := sanitizePart(getRoleDefinitionName(ra.Properties.RoleDefinitionId))

	if part2 == "" || part3 == "" {
		return "", errors.New("invalid principal or role name")
	}

	filename := fmt.Sprintf("%s_%s_%s.yaml", part1, part2, part3)

	if _, err := os.Stat(filename); err == nil {
		return "", fmt.Errorf("file %q already exists", filename)
	}

	return filename, nil
}

func main() {
	flag.Parse()
	if flag.NArg() < 1 {
		fmt.Fprintln(os.Stderr, "Error: missing input file argument")
		os.Exit(1)
	}

	yamlData, err := os.ReadFile(flag.Arg(0))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading file: %v\n", err)
		os.Exit(1)
	}

	var ra RoleAssignment
	if err := yaml.Unmarshal(yamlData, &ra); err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing YAML: %v\n", err)
		os.Exit(1)
	}

	filename, err := generateFilename(ra)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Println(filename)
}
