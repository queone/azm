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
// cache accessor implementation, and it can be based on below provided example.

// ==== Direct from Microsoft code ================================================
// https://github.com/AzureAD/microsoft-authentication-library-for-go/blob/main/apps/cache/cache.go
// https://github.com/AzureAD/microsoft-authentication-library-for-go/blob/main/apps/tests/devapps/sample_cache_accessor.go
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

// ==== Remainining code is part of the maz package ================================================

type cacheAccessorMemory struct {
	cache.Unmarshaler
}

func (m *cacheAccessorMemory) Unmarshal(data []byte) error {
	// Do nothing special — just test if it errors
	return nil
}

// Validate that the token cache file exists, is non-empty, and contains valid MSAL-formatted data.
func validateTokenCache(cacheAccessor *TokenCache) bool {
	if cacheAccessor == nil {
		Logf("Token cache accessor is nil\n")
		return false
	}

	data, err := os.ReadFile(cacheAccessor.file)
	if err != nil {
		if os.IsNotExist(err) {
			Logf("Token cache file does not exist: %s\n", cacheAccessor.file)
		} else {
			Logf("Error reading token cache file: %v\n", err)
		}
		return false
	}

	if len(data) == 0 {
		Logf("Token cache file is empty: %s\n", cacheAccessor.file)
		return false
	}

	// Try unmarshalling using a dummy in-memory cache
	dummy := &cacheAccessorMemory{}

	err = dummy.Unmarshal(data)
	if err != nil {
		Logf("Token cache unmarshal failed: %v\n", err)
		return false
	}

	return true
}

// Returns the service API name based on the scope
func getServiceApiName(scopes []string) string {
	service := "MS Graph (MG)" // ConstMgUrl
	for _, scope := range scopes {
		if strings.HasPrefix(scope, ConstAzUrl) {
			service = "Azure ARM (AZ)"
		}
	}
	return service
}

// Acquire Azure JWT token with Username via a browser popup window.
func GetTokenInteractively(scopes []string, z *Config) (token string, err error) {
	// See https://github.com/AzureAD/microsoft-authentication-library-for-go/blob/main/apps/public/public.go
	authorityUrl := ConstAuthUrl + z.TenantId
	username := z.Username

	// Set up and validate token cache file and accessor
	tokenFile := filepath.Join(MazConfigDir, TokenCacheFile)
	cacheAccessor := &TokenCache{tokenFile}

	if !validateTokenCache(cacheAccessor) {
		Logf("Invalid token cache. Deleting file %s to force refresh\n", tokenFile)
		if err := os.Remove(tokenFile); err != nil {
			Logf("Removal of token cache %s failed: %v\n", tokenFile, err)
		}
	}
	Logf("Token cache is valid\n")

	service := getServiceApiName(scopes)
	Logf("Getting access token for service %s\n", utl.Cya(service))

	// Retry configuration with backoff
	maxRetries := 3
	retryDelays := []time.Duration{2 * time.Second, 5 * time.Second, 10 * time.Second}
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
		Logf("Found %d accounts in cache:\n", len(accounts))
		for i, account := range accounts {
			Logf("  Account %d: HomeAccountID=%s, Username=%s\n", i+1, account.HomeAccountID, account.PreferredUsername)
		}

		Logf("Looking for account with PreferredUsername matching: %s\n", strings.ToLower(username))
		for _, account := range accounts {
			if strings.ToLower(account.PreferredUsername) == username {
				targetAccount = account
				break
			}
		}

		ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
		defer cancel()

		Logf("First, try getting token from cache (AcquireTokenSilent)\n")
		result, err := app.AcquireTokenSilent(ctx, scopes, public.WithSilentAccount(targetAccount))
		if err == nil {
			token := result.AccessToken // Actual token

			msg := fmt.Sprintf("Successfully got token from cache (attempt %d)", attempt)
			Logf("%s\n", utl.Cya(msg))

			// Verifying the newly acquired token
			if valid, err := VerifyAzureJwt(token); valid {
				Logf("%s\n", utl.Cya("Token verification passed"))
				return token, err
			} else {
				Logf("%s\n", utl.Red2("Token verification failed"))
				Logf("Trying again...\n")
				continue
			}
		} else {
			msg := fmt.Sprintf("Failed to get token from cache (attempt %d)", attempt)
			Logf("%s: %v\n", utl.Red2(msg), err)
		}

		Logf("Fallback to getting a token interactively from Microsoft identity platform (AcquireTokenInteractive)\n")
		result, err = app.AcquireTokenInteractive(ctx, scopes)
		if err == nil {
			Logf("Successfully acquired token interactively (attempt %d)\n", attempt)
			return result.AccessToken, nil
		}
		Logf("Interactive attempt %d failed: %v\n", attempt, err)

		// Final fallback to device code
		Logf("Fallback to getting a token via device code flow (AcquireTokenByDeviceCode) (attempt %d)\n", attempt)
		devCode, err := app.AcquireTokenByDeviceCode(ctx, scopes)
		if err != nil {
			Logf("Device code flow attempt %d failed: %v\n", attempt, err)
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
			Logf("Successfully acquired token via device code flow (attempt %d)\n", attempt)
			return result.AccessToken, nil
		}
		Logf("Device code flow attempt %d failed: %v\n", attempt, err)

		if attempt < maxRetries {
			time.Sleep(retryDelays[attempt-1])
		}
	}

	return "", fmt.Errorf("all interactive authentication methods failed after %d attempts", maxRetries)
}

// Initiates an Azure JWT token acquisition with provided parameters, using a Client ID plus a
// Client Secret. This is the 'Confidential' app auth flow and is documented at:
// https://github.com/AzureAD/microsoft-authentication-library-for-go/blob/dev/apps/confidential/confidential.go
func GetTokenByCredentials(scopes []string, z *Config) (token string, err error) {
	authorityUrl := ConstAuthUrl + z.TenantId
	clientId := z.ClientId
	clientSecret := z.ClientSecret

	// Set up and validate token cache file and accessor
	tokenFile := filepath.Join(MazConfigDir, TokenCacheFile)
	cacheAccessor := &TokenCache{tokenFile}

	if !validateTokenCache(cacheAccessor) {
		Logf("Invalid token cache. Deleting file %s to force refresh\n", tokenFile)
		if err := os.Remove(tokenFile); err != nil {
			Logf("Removal of token cache %s failed: %v\n", tokenFile, err)
		}
	}
	Logf("Token cache is valid\n")

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

	service := getServiceApiName(scopes)
	Logf("Getting access token for service %s\n", utl.Cya(service))

	Logf("First, try getting token from cache (AcquireTokenSilent)\n")
	result, err := app.AcquireTokenSilent(ctx, scopes)
	// Note that a targetAccount is not required; it appears to locate existing cached tokens without it
	if err == nil {
		token = result.AccessToken // Actual token
		Logf("%s\n", utl.Cya("Successfully got token from cache"))

		// Verifying the newly acquired token
		if valid, err := VerifyAzureJwt(token); valid {
			Logf("%s\n", utl.Cya("Token verification passed"))
			return token, err
		} else {
			Logf("%s\n", utl.Red2("Token verification failed!"))
			// Drop out of top IF statement, to try the fallback method
		}
	}
	Logf("%s: %v\n", utl.Red("Failed to get token from cache"), err)

	Logf("Fallback to getting a token direct from Microsoft identity platform (AcquireTokenByCredential)\n")
	result, err = app.AcquireTokenByCredential(ctx, scopes)
	if err == nil {
		Logf("%s\n", utl.Cya("Successfully got token from Microsoft"))
		return result.AccessToken, nil // Return the token string part
	}

	return "", err
}
