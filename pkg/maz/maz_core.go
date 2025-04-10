// Package maz is a library of functions for interacting with essential Azure APIs via
// REST calls. Currently it supports two APIs, the Azure Resource Management (ARM) API
// and the MS Graph API, but can be extended to support additional APIs. This package
// obviously also includes code to get an Azure JWT token using the MSAL library, to
// then use against either the 2 currently supported Azure APIs.

package maz

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/queone/utl"
)

const (
	ConfigBaseDir   = ".maz"
	CredentialsFile = "credentials.yaml"
	TokenCacheFile  = "token_cache.json"
	ConstAuthUrl    = "https://login.microsoftonline.com/"
	ConstMgUrl      = "https://graph.microsoft.com"
	ConstAzUrl      = "https://management.azure.com"
	AzApiToken      = "AzApiToken"
	MgApiToken      = "MgApiToken"
	UnknownApiToken = "UnknownApiToken"

	ConstAzPowerShellClientId = "1950a258-227b-4e31-a9cf-717495945fc2" // 'Microsoft Azure PowerShell' ClientId
	//ConstAzPowerShellClientId = "04b07795-8ddb-461a-bbee-02f9e1bf7b46" // 'Microsoft Azure CLI' ClientId
	// Interactive login can use either of above ClientIds. See below references:
	//   - https://learn.microsoft.com/en-us/troubleshoot/azure/active-directory/verify-first-party-apps-sign-in
	//   - https://stackoverflow.com/questions/30454771/how-does-azure-powershell-work-with-username-password-based-auth

	rUp = "\x1B[2K\r" // Clears the line completely and move cursor to the start of the line
	// See https://stackoverflow.com/questions/1508490/erase-the-current-printed-console-line

	ConstMgCacheFileAgePeriod = 1800  // Half hour
	ConstAzCacheFileAgePeriod = 86400 // One day

	YamlFormat = "yaml"
	JsonFormat = "json"

	// Maz object type strings
	ResRoleDefinition = "d"  // Azure resource role definition
	ResRoleAssignment = "a"  // Azure resource role assignment
	Subscription      = "s"  // Azure resource subscription
	ManagementGroup   = "m"  // Azure resource management group
	DirectoryUser     = "u"  // Azure directory user
	DirectoryGroup    = "g"  // Azure directory group
	Application       = "ap" // Azure directory application
	ServicePrincipal  = "sp" // Azure directory service principal
	DirRoleDefinition = "dr" // Azure directory role definition
	DirRoleAssignment = "da" // Azure directory role assignment
	UnknownObject     = ""
	AllMazObjects     = "x"
)

var (
	MazConfigDir string // Global configuration directory, see init()

	MazTypes = []string{
		ResRoleDefinition,
		ResRoleAssignment,
		Subscription,
		ManagementGroup,
		DirectoryUser,
		DirectoryGroup,
		Application,
		ServicePrincipal,
		DirRoleDefinition,
		DirRoleAssignment,
	}
	MazTypeNames = map[string]string{
		ResRoleDefinition: "resource role definition",
		ResRoleAssignment: "resource role assignment",
		Subscription:      "resource subscription",
		ManagementGroup:   "resource management group",
		DirectoryUser:     "directory user",
		DirectoryGroup:    "directory group",
		Application:       "directory application",
		ServicePrincipal:  "directory service principal",
		DirRoleDefinition: "directory role definition",
		DirRoleAssignment: "directory role assignment",
	}
	CacheSuffix = map[string]string{
		ResRoleDefinition: "_res-role-defs",
		ResRoleAssignment: "_res-role-asgns",
		Subscription:      "_res-subs",
		ManagementGroup:   "_res-mgmt-groups",
		DirectoryUser:     "_dir-users",
		DirectoryGroup:    "_dir-groups",
		Application:       "_dir-apps",
		ServicePrincipal:  "_dir-sps",
		DirRoleDefinition: "_dir-role-defs",
		DirRoleAssignment: "_dir-role-asgns",
	}
	ApiEndpoint = map[string]string{
		ResRoleDefinition: "/providers/Microsoft.Authorization/roleDefinitions",
		ResRoleAssignment: "/providers/Microsoft.Authorization/roleAssignments",
		Subscription:      "/subscriptions",
		ManagementGroup:   "/providers/Microsoft.Management/managementGroups",
		DirectoryUser:     "/v1.0/users",
		DirectoryGroup:    "/v1.0/groups",
		Application:       "/v1.0/applications",
		ServicePrincipal:  "/v1.0/servicePrincipals",
		DirRoleDefinition: "/v1.0/roleManagement/directory/roleDefinitions",
		DirRoleAssignment: "/v1.0/roleManagement/directory/roleAssignments",
	}
	mazEnvironmentVars = map[string]string{
		"MAZ_TENANT_ID":     "",
		"MAZ_USERNAME":      "",
		"MAZ_INTERACTIVE":   "",
		"MAZ_CLIENT_ID":     "",
		"MAZ_CLIENT_SECRET": "",
		"MAZ_MG_TOKEN":      "",
		"MAZ_AZ_TOKEN":      "",
	}
)

// Config holds configuration and credentials for various APIs and the calling programs themselves.
type Config struct {
	TenantId     string
	ClientId     string
	ClientSecret string
	Interactive  bool
	Username     string
	// --- For MS Graph API
	MgToken   string
	MgHeaders map[string]string
	// --- For ARM API
	AzToken   string
	AzHeaders map[string]string
	// --- Add other API token/headers here...
}

// Initialize MazConfigDir to the user's home directory in a cross-platform way.
func init() {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		utl.Die("Could not determine user home directory: %v", err)
	}

	MazConfigDir = filepath.Join(homeDir, ConfigBaseDir)

	// Ensure the configuration directory exists
	if _, err := os.Stat(MazConfigDir); os.IsNotExist(err) {
		if err := os.Mkdir(MazConfigDir, 0700); err != nil {
			utl.Die("Failed to create '%s' config directory: %v",
				utl.Yel(MazConfigDir), err)
		}
	}
}

// Constructs, initializes, and returns a pointer to a Config instance.
// The returned pointer can be used as a global configuration object to store
// credentials, tokens, and other API-related details for the application.
func NewConfig() *Config {
	return &Config{
		MgHeaders: make(map[string]string),
		AzHeaders: make(map[string]string),
	}
}

// Adds a Microsoft Graph API header.
func (m *Config) AddMgHeader(key, value string) *Config {
	m.MgHeaders[key] = value
	return m
}

// Adds an Azure Resource Management API header.
func (m *Config) AddAzHeader(key, value string) *Config {
	m.AzHeaders[key] = value
	return m
}

// Deletes current credentials and token files
func DeleteCurrentCredentials(z *Config) {
	utl.RemoveFile(filepath.Join(MazConfigDir, TokenCacheFile))  // Remove token file
	utl.RemoveFile(filepath.Join(MazConfigDir, CredentialsFile)) // Remove credentials file
	os.Exit(0)
}

// Purges the cache files for given maz object type(s)
func PurgeMazObjectCacheFiles(mazType string, z *Config) {
	var hasError bool // Flag to track if any errors occurred

	if mazType == AllMazObjects {
		for mazType, mazTypeName := range MazTypeNames {
			hasError = true // Set the flag to true if an error occurs
			if err := PurgeCacheFiles(mazType, z); err != nil {
				fmt.Printf("Error removing %s cache files: %v\n", utl.Red(mazTypeName), err)
			}
		}
	} else {
		if err := PurgeCacheFiles(mazType, z); err != nil {
			hasError = true
			fmt.Printf("Error removing %s cache files: %v\n", utl.Red(MazTypeNames[mazType]), err)
		}
	}

	if hasError {
		os.Exit(1) // Exit with code 1 if any errors occurred
	} else {
		os.Exit(0) // Exit with code 0 if everything was successful
	}
}

// Dumps configured login values
func DumpLoginValues(z *Config) {
	fmt.Printf("%s: %s  %s\n", utl.Blu("config_dir"), utl.Gre(MazConfigDir),
		utl.Gra("# Config and cache directory"))

	fmt.Printf("%s:\n", utl.Blu("config_vars"))
	comment := "  # 1. MS Graph and Azure ARM tokens can be supplied directly via MAZ_MG_TOKEN and\n" +
		"  #    MAZ_AZ_TOKEN environment variables, and they have the highest precedence.\n" +
		"  #    Note, MAZ_TENANT_ID is still required when using these 2.\n" +
		"  # 2. Credentials supplied via environment variables have precedence over those\n" +
		"  #    provided via credentials file.\n" +
		"  # 3. The MAZ_USERNAME + MAZ_INTERACTIVE combo have priority over the MAZ_CLIENT_ID\n" +
		"  #    + MAZ_CLIENT_SECRET combination.\n"
	fmt.Print(utl.Gra(comment))
	fmt.Printf("  %s: %s\n", utl.Blu("MAZ_TENANT_ID"), utl.Gre(os.Getenv("MAZ_TENANT_ID")))
	fmt.Printf("  %s: %s\n", utl.Blu("MAZ_USERNAME"), utl.Gre(os.Getenv("MAZ_USERNAME")))
	fmt.Printf("  %s: %s\n", utl.Blu("MAZ_INTERACTIVE"), utl.Mag(os.Getenv("MAZ_INTERACTIVE")))
	fmt.Printf("  %s: %s\n", utl.Blu("MAZ_CLIENT_ID"), utl.Gre(os.Getenv("MAZ_CLIENT_ID")))
	fmt.Printf("  %s: %s\n", utl.Blu("MAZ_CLIENT_SECRET"), utl.Gre(os.Getenv("MAZ_CLIENT_SECRET")))
	fmt.Printf("  %s: %s\n", utl.Blu("MAZ_MG_TOKEN"), utl.Gre(os.Getenv("MAZ_MG_TOKEN")))
	fmt.Printf("  %s: %s\n", utl.Blu("MAZ_AZ_TOKEN"), utl.Gre(os.Getenv("MAZ_AZ_TOKEN")))
	fmt.Printf("%s:\n", utl.Blu("config_creds_file"))
	credsFile := filepath.Join(MazConfigDir, CredentialsFile)
	fmt.Printf("  %s: %s\n", utl.Blu("file_path"), utl.Gre(credsFile))
	credsRaw, err := utl.LoadFileYaml(credsFile)
	if err != nil {
		utl.Die("  %s\n", utl.Red("Credentials file does not exists yet."))
	}
	if creds := utl.Map(credsRaw); creds != nil {
		fmt.Printf("  %s: %s\n", utl.Blu("tenant_id"), utl.Gre(utl.Str(creds["tenant_id"])))
		if strings.ToLower(utl.Str(creds["interactive"])) == "true" {
			fmt.Printf("  %s: %s\n", utl.Blu("username"), utl.Gre(utl.Str(creds["username"])))
			fmt.Printf("  %s: %s\n", utl.Blu("interactive"), utl.Mag("true"))
		} else {
			fmt.Printf("  %s: %s\n", utl.Blu("client_id"), utl.Gre(utl.Str(creds["client_id"])))
			fmt.Printf("  %s: %s\n", utl.Blu("client_secret"), utl.Gre(utl.Str(creds["client_secret"])))
		}
	} else {
		utl.Die("  %s\n", utl.Red("Error reading credentials file."))
	}
	os.Exit(0)
}

// Configure maz credentials file for interactive login
func ConfigureCredsFileForInterativeLogin(z *Config) {
	credsFile := filepath.Join(MazConfigDir, CredentialsFile)
	if !utl.ValidUuid(z.TenantId) {
		utl.Die("Error. TENANT_ID is an invalid UUID.\n")
	}
	content := fmt.Sprintf("%-14s %s\n%-14s %s\n%-14s %s\n", "tenant_id:", z.TenantId,
		"username:", z.Username, "interactive:", "true")
	if err := os.WriteFile(credsFile, []byte(content), 0600); err != nil { // Write string to file
		panic(err.Error())
	}
	fmt.Printf("Updated %s file\n", utl.Yel(credsFile))
	os.Exit(0)
}

// Configure maz credentials file for automated client_id/secret login
func ConfigureCredsFileForAutomatedLogin(z *Config) {
	credsFile := filepath.Join(MazConfigDir, CredentialsFile)
	if !utl.ValidUuid(z.TenantId) {
		utl.Die("Error. TENANT_ID is an invalid UUID.\n")
	}
	if !utl.ValidUuid(z.ClientId) {
		utl.Die("Error. CLIENT_ID is an invalid UUID.\n")
	}
	content := fmt.Sprintf("%-14s %s\n%-14s %s\n%-14s %s\n", "tenant_id:", z.TenantId,
		"client_id:", z.ClientId, "client_secret:", z.ClientSecret)
	if err := os.WriteFile(credsFile, []byte(content), 0600); err != nil { // Write string to file
		panic(err.Error())
	}
	fmt.Printf("Updated %s file\n", utl.Yel(credsFile))
	os.Exit(0)
}

// Configure variables and API credentials for maz
func SetupMazCredentials(z *Config) {
	// For login credentials precedence is given to environment variables

	// Check if credentials have been provided via environment variables
	usingEnvVars := false // Assume they have not
	for k := range mazEnvironmentVars {
		mazEnvironmentVars[k] = os.Getenv(k)
		// Read all MAZ_* environment variables
		if mazEnvironmentVars[k] != "" {
			// If any are set, then environment variable are being used
			usingEnvVars = true
		}
	}
	if usingEnvVars {
		SetupMazCredentialsFromEnvVars(z)
	} else {
		SetupMazCredentialsFromFile(z)
	}
}

// Configure login credentials from OS environment variables
func SetupMazCredentialsFromEnvVars(z *Config) {
	Logf("Using environment variables for login credentials\n")
	z.TenantId = mazEnvironmentVars["MAZ_TENANT_ID"]
	if !utl.ValidUuid(z.TenantId) {
		utl.Die("Error: Environment variable MAZ_TENANT_ID '%s' is not a valid UUID. "+
			"Cannot continue.\n", z.TenantId)
	}
	Logf("1. Environment variable MAZ_TENANT_ID is set to %s\n", utl.Yel(z.TenantId))

	// Use API login tokens provided via environment variables
	z.AzToken = mazEnvironmentVars["MAZ_AZ_TOKEN"]
	z.MgToken = mazEnvironmentVars["MAZ_MG_TOKEN"]
	_, azErr := SplitJWT(z.AzToken)
	_, mgErr := SplitJWT(z.MgToken)
	bothTokensAreValid := azErr == nil && mgErr == nil
	if bothTokensAreValid {
		Logf("2. Environment variable MAZ_AZ_TOKEN appears to have a valid token: Suffix = %s\n",
			utl.Yel(GetTokenSuffix(z.AzToken)))
		Logf("3. Environment variable MAZ_MG_TOKEN appears to have a valid token: Suffix = %s\n",
			utl.Yel(GetTokenSuffix(z.MgToken)))
		Logf("Attempting %s login\n", utl.Yel("automated token-based"))
		return // Return early since we have all creds for this type of login
	}

	// Assume the 2 API tokens will be acquired using the other variables, so let's check them
	z.Interactive = utl.Bool(mazEnvironmentVars["MAZ_INTERACTIVE"])
	if z.Interactive {
		Logf("2. Environment variable MAZ_INTERACTIVE is set to %s", utl.Yel(z.Interactive))
		z.Username = strings.ToLower(utl.Str(mazEnvironmentVars["MAZ_USERNAME"]))
		if z.Username != "" {
			Logf("3. Environment variable MAZ_USERNAME is set to %s", utl.Yel(z.Username))
			Logf("Attempting %s login\n", utl.Yel("interactive username"))
		} else {
			utl.Die("Error: Environment variable MAZ_USERNAME is blank. Cannot continue " +
				"with interactive login.\n")
		}
	} else {
		z.ClientId = utl.Str(mazEnvironmentVars["MAZ_CLIENT_ID"])
		if !utl.ValidUuid(z.ClientId) {
			utl.Die("Error: The chosen login method appears to be via environment variables, "+
				"but variable MAZ_CLIENT_ID '%s' is not a valid UUID. Cannot continue.\n", z.ClientId)
		}
		Logf("2. Environment variable MAZ_CLIENT_ID is set to %s", utl.Yel(z.ClientId))
		z.ClientSecret = utl.Str(mazEnvironmentVars["MAZ_CLIENT_SECRET"])
		if z.ClientSecret == "" {
			utl.Die("Error: The chosen login method appears to be via environment variables, " +
				"but variable MAZ_CLIENT_SECRET is blank. Cannot continue.\n")
		}
		Logf("3. Environment variable MAZ_CLIENT_SECRET has a value.\n")
		Logf("Attempting %s login\n", utl.Yel("automated client_id/secret"))
	}
}

// Configure login credentials from credentials file
func SetupMazCredentialsFromFile(z *Config) {
	credsFile := filepath.Join(MazConfigDir, CredentialsFile)
	Logf("Using credential file %s\n", utl.Yel(credsFile))
	if !utl.FileUsable(credsFile) {
		utl.Die("Error: Credential file is missing!\n")
	}
	Logf("Credential file exists\n")

	credsRaw, err := utl.LoadFileYaml(credsFile)
	if err != nil {
		utl.Die("Error: %s\n", err)
	}
	Logf("Credential file is valid YAML\n")

	creds := utl.Map(credsRaw)
	if creds == nil {
		utl.Die("Error: Credential file %s values are not formatted properly.\n",
			utl.Red(credsFile))
	}
	Logf("Credential file parameters/values appear to be formatted properly.\n")

	z.TenantId = utl.Str(creds["tenant_id"])
	if !utl.ValidUuid(z.TenantId) {
		utl.Die("Error: Credential file %s parameter %s (%s) is not a valid UUID.\n",
			utl.Red(credsFile), utl.Red("tenant_id"), z.TenantId)
	}
	Logf("1. Credential file parameter 'tenant_id' is set to %s\n", utl.Yel(z.TenantId))

	z.Interactive = utl.Bool(creds["interactive"])
	if z.Interactive {
		Logf("2. Credential file parameter 'interactive' is set to %s\n", utl.Yel(z.Interactive))
		z.Username = strings.ToLower(utl.Str(creds["username"]))
		if z.Username != "" {
			Logf("3. Credential file parameter 'username' is set to %s\n", utl.Yel(z.Username))
			Logf("Attempting %s login\n", utl.Yel("interactive username"))
		} else {
			utl.Die("Error: Credential file parameter 'username' is blank. Cannot " +
				"continue with interactive login.\n")
		}
	} else {
		z.ClientId = utl.Str(creds["client_id"])
		if !utl.ValidUuid(z.ClientId) {
			utl.Die("Error: Credential file parameter %s (%s) is not a valid UUID.\n",
				utl.Red("client_id"), z.ClientId)
		}
		Logf("2. Credential file parameter 'client_id' is set to %s\n", utl.Yel(z.ClientId))

		z.ClientSecret = utl.Str(creds["client_secret"])
		if z.ClientSecret == "" {
			utl.Die("Error: Credential file parameter %s is blank. Cannot continue.\n",
				utl.Red("client_secret"))
		}
		Logf("3. Credential file parameter 'client_secret' has a value.\n")
		Logf("Attempting %s login\n", utl.Yel("automated client_id/secret"))
	}
}

// Initializes all necessary global variables and acquires and sets all API tokens.
func SetupApiTokens(z *Config) {
	SetupMazCredentials(z) // Set up authentication method and required variables

	// This function must initialize ALL service API tokens. A failure to do so for
	// any any token will result in the program aborting.

	// Initialize Azure ARM API token
	SetupAzureArmToken(z)

	// Initialize MS Graph API token
	SetupMsGraphToken(z)

	// Other API tokens can be initialized here...
}

// Sets up the Azure Resource Management (ARM) API token
func SetupAzureArmToken(z *Config) {
	// If token is not valid, then lets acquire a new one
	if _, err := SplitJWT(z.AzToken); err != nil {
		Logf("AZ token suffix = %s\n", utl.Cya(GetTokenSuffix(z.AzToken)))
		scope := []string{ConstAzUrl + "/.default"}
		// Appending '/.default' allows using all static and consented permissions of the identity in
		// use. See https://learn.microsoft.com/en-us/azure/active-directory/develop/msal-v1-app-scopes
		var err error
		z.AzToken, err = GetApiToken(scope, z) // Get the Azure ARM token
		if err != nil {
			utl.Die("%s: %v\n", utl.Red("Error"), err)
		}
		Logf("AZ token suffix = %s\n", utl.Cya(GetTokenSuffix(z.AzToken)))
		// Setup the base API headers; token + content type
		z.AddAzHeader("Authorization", "Bearer "+z.AzToken).AddAzHeader("Content-Type", "application/json")
	}
}

// Sets up the Microsoft Graph API token
func SetupMsGraphToken(z *Config) {
	// If token is not valid, then lets acquire a new one
	if _, err := SplitJWT(z.MgToken); err != nil {
		Logf("MG token suffix = %s\n", utl.Cya(GetTokenSuffix(z.MgToken)))
		scope := []string{ConstMgUrl + "/.default"}
		var err error
		z.MgToken, err = GetApiToken(scope, z) // Get the MS Graph token
		if err != nil {
			utl.Die("%s: %v\n", utl.Red("Error"), err)
		}
		Logf("MG token suffix = %s\n", utl.Cya(GetTokenSuffix(z.MgToken)))
		// Setup the base API headers; token + content type
		z.AddMgHeader("Authorization", "Bearer "+z.MgToken).AddMgHeader("Content-Type", "application/json")
	}
}

// Acquires an access token for the given API scope using one of two different methods
func GetApiToken(scope []string, z *Config) (string, error) {
	if z.Interactive {
		// User has configured the utility to do interactive username popup browser login
		return GetTokenInteractively(scope, z)
	} else {
		// User has configured the utility to do automated client_id/secret login
		return GetTokenByCredentials(scope, z)
	}
}

// Checks if a JWT token string is properly formatted and splits it into its three parts.
func SplitJWT(tokenString string) ([]string, error) {
	if tokenString == "" {
		return nil, fmt.Errorf("token is empty")
	}
	if !strings.HasPrefix(tokenString, "eyJ") {
		return nil, fmt.Errorf("token does not appear to start with a JWT header")
	}

	parts := strings.Split(tokenString, ".")
	if len(parts) != 3 {
		return nil, fmt.Errorf("invalid JWT format: expected 3 parts separated by '.'")
	}

	return parts, nil
}

// Check if MAZ_LOG logging is enabled
func isMazLoggingEnabled() bool {
	val := strings.ToLower(os.Getenv("MAZ_LOG"))
	return val == "1" || val == "true" || val == "yes"
}

// Prints colorized, formatted debugging messages to stderr when MAZ_LOG is enabled
func Logf(format string, args ...interface{}) {
	if !isMazLoggingEnabled() {
		return
	}

	// Get caller info
	_, file, line, ok := runtime.Caller(1)
	if !ok {
		file = "???"
		line = 0
	}

	// Format prefix and message
	timestamp := time.Now().Format("2006-Jan-02 15:04:05")
	shortFile := filepath.Base(file)
	prefix := utl.Cya(fmt.Sprintf("MAZ %s %s:%04d", timestamp, shortFile, line))
	msg := fmt.Sprintf(prefix+": "+format, args...)

	// Write to stderr with forced flush
	fmt.Fprint(os.Stderr, msg)
}
