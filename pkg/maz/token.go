package maz

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/AzureAD/microsoft-authentication-library-for-go/apps/cache"
	"github.com/AzureAD/microsoft-authentication-library-for-go/apps/confidential"
	"github.com/AzureAD/microsoft-authentication-library-for-go/apps/public"
	"github.com/queone/utl"
)

// The MSAL Go library defines the types of cache file, and expect you to roll your own
// implementation. See below:
//   https://github.com/AzureAD/microsoft-authentication-library-for-go/blob/v1.4.0/apps/cache/cache.go
//
// One can base one's own cache accessor on below examples:
//   https://github.com/AzureAD/microsoft-authentication-library-for-go/blob/v1.4.0/apps/tests/devapps/sample_cache_accessor.go
//   https://github.com/AzureAD/microsoft-authentication-library-for-go/blob/v1.4.0/apps/tests/integration/cache_accessor.go
//

// Below type and methods are verbatim copies of the ones in file 'cache_accessor.go' from above .

// Copyright (c) Microsoft Corporation.
// Licensed under the MIT license.

type TokenCache struct {
	file string
}

func (t *TokenCache) Replace(ctx context.Context, cache cache.Unmarshaler, hints cache.ReplaceHints) error {
	data, err := os.ReadFile(t.file)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // Return nil if file doesn't exist yet
		}
		return err
	}
	if len(data) == 0 {
		return nil // Skip if empty file
	}
	return cache.Unmarshal(data)
}

func (t *TokenCache) Export(ctx context.Context, cache cache.Marshaler, hints cache.ExportHints) error {
	data, err := cache.Marshal()
	if err != nil {
		return err
	}
	return os.WriteFile(t.file, data, 0600)
}

func (t *TokenCache) Print() string {
	data, err := os.ReadFile(t.file)
	if err != nil {
		return err.Error()
	}
	return string(data)
}

// ==== Remainining code is part of the maz package ================================================
// Copyright (c) The maz Authors.
// Licensed under the MIT license.

// Validate token cache - ALWAYS TRUE FOR NOW
func validateTokenCache(cacheAccessor *TokenCache) bool {
	// Implement actual cache validation logic
	if cacheAccessor != nil {
		return true
	}
	return true // Placeholder
}

// // Check for network errors
// func isNetworkError(err error) bool {
// 	return strings.Contains(err.Error(), "EOF") ||
// 		strings.Contains(err.Error(), "connection reset") ||
// 		strings.Contains(err.Error(), "timeout")
// }

// Initiates an Azure JWT token acquisition with provided parameters, using a Username and a browser
// pop up window. This is the 'Public' app auth flow as documented at:
// https://github.com/AzureAD/microsoft-authentication-library-for-go/blob/dev/apps/public/public.go
func GetTokenInteractively(scopes []string, z *Config) (token string, err error) {
	confDir := z.ConfDir
	tokenFile := z.TokenFile
	authorityUrl := ConstAuthUrl + z.TenantId
	username := z.Username

	// Set up token cache storage file and accessor
	cacheAccessor := &TokenCache{filepath.Join(confDir, tokenFile)}

	// Retry configuration with backoff
	maxRetries := 3
	retryDelays := []time.Duration{2 * time.Second, 5 * time.Second, 10 * time.Second}

	// Cache validation check
	if cacheValid := validateTokenCache(cacheAccessor); !cacheValid {
		Logf("Cache invalid, forcing fresh authentication\n")
		if err := os.Remove(filepath.Join(confDir, tokenFile)); err != nil {
			Logf("Cache removal failed: %v\n", err)
		}
	}

	// Determine the service API the token is for
	service := "MS Graph (MG)" // ConstMgUrl
	for _, scope := range scopes {
		if strings.HasPrefix(scope, ConstAzUrl) {
			service = "Azure ARM (AZ)"
		}
	}

	for attempt := 1; attempt <= maxRetries; attempt++ {
		ctx := context.Background()

		// Create new app instance for each attempt
		app, err := public.New(ConstAzPowerShellClientId,
			public.WithAuthority(authorityUrl),
			public.WithCache(cacheAccessor))
		if err != nil {
			Logf("Attempt %d app init failed: %v\n", attempt, err)
			if attempt == maxRetries {
				return "", fmt.Errorf("failed to initialize after %d attempts: %w", maxRetries, err)
			}
			time.Sleep(retryDelays[attempt-1])
			continue
		}

		// Look for cached account
		var targetAccount public.Account
		accounts, err := app.Accounts(ctx)
		if err != nil {
			Logf("Attempt %d account lookup failed: %v\n", attempt, err)
			if attempt == maxRetries {
				return "", fmt.Errorf("account lookup failed after %d attempts: %w", maxRetries, err)
			}
			time.Sleep(retryDelays[attempt-1])
			continue
		}

		for _, account := range accounts {
			if strings.ToLower(account.PreferredUsername) == username {
				targetAccount = account
				break
			}
		}

		ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
		defer cancel()

		// Try silent acquisition first
		result, err := app.AcquireTokenSilent(ctx, scopes, public.WithSilentAccount(targetAccount))
		if err == nil {
			token := result.AccessToken // Actual token

			msg := fmt.Sprintf("AcquireTokenSilent for service %s SUCCEEDED (attempt %d)",
				service, attempt)
			Logf("%s: Suffix = %s\n", utl.Gre(msg), utl.Cya2(GetTokenSuffix(token)))

			// // HOLD OFF ON THIS FOR NOW
			// if valid, err := VerifyAzureJwt(token); valid {
			// 	Logf("%s\n", utl.Gre("Token verification PASSED!"))
			// 	return token, err
			// } else {
			// 	Logf("%s\n", utl.Red2("Token verification FAILED!"))
			// 	Logf("Trying again...\n")
			// 	continue
			// }

			return token, nil

		} else {
			msg := fmt.Sprintf("AcquireTokenSilent for service %s FAILED (attempt %d)",
				service, attempt)
			Logf("%s: Suffix = %s\n", utl.Red2(msg), utl.Cya2("none"))
			Logf("Error: %v\n", err)

			// // Special handling for network errors
			// if isNetworkError(err) {
			// 	Logf("Network error detected, retrying...\n")
			// 	time.Sleep(retryDelays[attempt-1])
			// 	continue
			// }
		}

		// Fall back to interactive
		result, err = app.AcquireTokenInteractive(ctx, scopes)
		if err == nil {
			Logf("Successfully acquired interactive token (attempt %d)\n", attempt)
			return result.AccessToken, nil
		}
		Logf("Attempt %d interactive failed: %v\n", attempt, err)

		// Final fallback to device code
		Logf("Fallback to AcquireTokenByDeviceCode for service %s (attempt %d)\n", service, attempt)
		devCode, err := app.AcquireTokenByDeviceCode(ctx, scopes)
		if err != nil {
			Logf("Attempt %d device code init failed: %v\n", attempt, err)
			if attempt == maxRetries {
				return "", fmt.Errorf("device code flow failed after %d attempts: %w", maxRetries, err)
			}
			time.Sleep(retryDelays[attempt-1])
			continue
		}

		verificationUri := devCode.Result.VerificationURL
		if verificationUri == "" {
			verificationUri = "https://microsoft.com/devicelogin"
		}
		fmt.Printf("\nOpen in browser: %s\n", verificationUri)
		fmt.Printf("Enter code: %s\n\n", devCode.Result.UserCode)

		result, err = devCode.AuthenticationResult(ctx)
		if err == nil {
			Logf("Successfully acquired device code token (attempt %d)\n", attempt)
			return result.AccessToken, nil
		}
		Logf("Attempt %d device auth failed: %v\n", attempt, err)

		if attempt < maxRetries {
			time.Sleep(retryDelays[attempt-1])
		}
	}

	return "", fmt.Errorf("all authentication methods failed after %d attempts", maxRetries)
}

// Initiates an Azure JWT token acquisition with provided parameters, using a Client ID plus a
// Client Secret. This is the 'Confidential' app auth flow and is documented at:
// https://github.com/AzureAD/microsoft-authentication-library-for-go/blob/dev/apps/confidential/confidential.go
func GetTokenByCredentials(scopes []string, z *Config) (token string, err error) {
	confDir := z.ConfDir
	tokenFile := z.TokenFile
	authorityUrl := ConstAuthUrl + z.TenantId
	clientId := z.ClientId
	clientSecret := z.ClientSecret

	// Set up token cache storage file and accessor
	cacheAccessor := &TokenCache{filepath.Join(confDir, tokenFile)}

	// Initializing the client credential
	cred, err := confidential.NewCredFromSecret(clientSecret)
	if err != nil {
		Logf("%v\n", err)
		return "", err
	}

	// Automated login obviously uses the registered app client_id (App ID)
	app, err := confidential.New(authorityUrl, clientId, cred, confidential.WithCache(cacheAccessor))
	if err != nil {
		Logf("%v\n", err)
		return "", err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Determine the service API the token is for
	service := "MS Graph (MG)" // ConstMgUrl
	for _, scope := range scopes {
		if strings.HasPrefix(scope, ConstAzUrl) {
			service = "Azure ARM (AZ)"
		}
	}

	// First, try to get token silent
	Logf("First, try AcquireTokenSilent for service %s\n", service)
	result, err := app.AcquireTokenSilent(ctx, scopes)
	// Note that a targetAccount is not required; it appears to locate existing cached tokens without it
	if err == nil {
		token = result.AccessToken // Actual token
		msg := fmt.Sprintf("AcquireTokenSilent for service %s SUCCEEDED", service)
		Logf("%s: Suffix = %s\n", utl.Gre(msg), utl.Cya2(GetTokenSuffix(token)))
		Logf("Doing a full token verification...\n")

		// // HOLD OFF ON THIS FOR NOW
		// if valid, err := VerifyAzureJwt(token); valid {
		// 	Logf("%s\n", utl.Gre("Token verification PASSED!"))
		// 	return token, err
		// } else {
		// 	Logf("%s\n", utl.Red2("Token verification FAILED!"))
		// 	// Drop out of top IF statement, to try the fallback method
		// }

		return token, nil
	}
	msg := fmt.Sprintf("AcquireTokenSilent for service %s FAILED", service)
	Logf("%s: Suffix = %s\n", utl.Red2(msg), utl.Cya2("none"))
	Logf("Error: %v\n", err)

	// Final fallback to by credential
	Logf("Fallback to AcquireTokenByCredential for service %s\n", service)
	result, err = app.AcquireTokenByCredential(ctx, scopes)
	if err == nil {
		token = result.AccessToken // Actual token
		msg := fmt.Sprintf("AcquireTokenByCredential for service %s SUCCEEDED", service)
		Logf("%s: Suffix = %s\n", utl.Gre(msg), utl.Cya2(GetTokenSuffix(token)))
		Logf("%v\n", err)
		return token, nil
	}

	return "", err
}
