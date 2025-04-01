package maz

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/AzureAD/microsoft-authentication-library-for-go/apps/cache"
	"github.com/AzureAD/microsoft-authentication-library-for-go/apps/confidential"
	"github.com/AzureAD/microsoft-authentication-library-for-go/apps/public"
	"github.com/golang-jwt/jwt/v5"
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

	// Retry configuration
	maxRetries := 3
	retryDelay := 2 * time.Second

	for attempt := 1; attempt <= maxRetries; attempt++ {
		ctx := context.Background()

		// Note we're using constant ConstAzPowerShellClientId for interactive login
		app, err := public.New(ConstAzPowerShellClientId,
			public.WithAuthority(authorityUrl),
			public.WithCache(cacheAccessor))
		if err != nil {
			Log("Attempt %d: %v\n", attempt, err)
			if attempt == maxRetries {
				return "", err
			}
			time.Sleep(retryDelay)
			continue
		}

		// Look for cached account
		var targetAccount public.Account
		accounts, err := app.Accounts(ctx)
		if err != nil {
			Log("Attempt %d: %v\n", attempt, err)
			if attempt == maxRetries {
				return "", err
			}
			time.Sleep(retryDelay)
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
			return result.AccessToken, nil
		}
		Log("Attempt %d silent failed: %v\n", attempt, err)

		// Fall back to interactive which uses the default web browser to select
		// the account and then acquire a security token from the authority.
		result, err = app.AcquireTokenInteractive(ctx, scopes)
		if err == nil {
			return result.AccessToken, nil
		}
		Log("Attempt %d interactive failed: %v\n", attempt, err)

		// AcquireTokenInteractive may not work if user is within a VM environment,
		// so finally fallback to device code
		fmt.Println("\nFalling back to device code flow...")
		devCode, err := app.AcquireTokenByDeviceCode(ctx, scopes)
		if err != nil {
			Log("Attempt %d device code failed: %v\n", attempt, err)
			if attempt == maxRetries {
				return "", err
			}
			time.Sleep(retryDelay)
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
			return result.AccessToken, nil
		}
		Log("Attempt %d device auth failed: %v\n", attempt, err)

		if attempt < maxRetries {
			time.Sleep(retryDelay)
		}
	}

	return "", fmt.Errorf("authentication failed after %d attempts", maxRetries)
}

// Initiates an Azure JWT token acquisition with provided parameters, using a Client ID plus a
// Client Secret. This is the 'Confidential' app auth flow and is documented at:
// https://github.com/AzureAD/microsoft-authentication-library-for-go/blob/dev/apps/confidential/confidential.go
func GetTokenByCredentials(scopes []string, z *Config) (token string, err error) {
	// func GetTokenByCredentials(scopes []string, confDir, tokenFile, authorityUrl, clientId, clientSecret string) (token string, err error) {
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
		Log("%v\n", err)
		return "", err
	}

	// Automated login obviously uses the registered app client_id (App ID)
	app, err := confidential.New(authorityUrl, clientId, cred, confidential.WithCache(cacheAccessor))
	if err != nil {
		Log("%v\n", err)
		return "", err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Try getting cached token 1st
	// targetAccount not required, as it appears to locate existing cached tokens without it
	result, err := app.AcquireTokenSilent(ctx, scopes)
	if err != nil {
		Log("%v\n", err)
		// If for whatever reason getting a cached token didn't work, then let's get a fresh token
		result, err = app.AcquireTokenByCredential(ctx, scopes)
		// AcquireTokenByCredential acquires a security token from the authority, using the client credentials grant
		if err != nil {
			Log("%v\n", err)
			return "", err
		}
	}
	return result.AccessToken, nil // Return only the AccessToken, which is of type string
}

// Validates a JWT token *string format* as defined in https://tools.ietf.org/html/rfc7519
func IsValidTokenFormat(tokenString string) (bool, string) {
	if tokenString == "" {
		return false, "token is empty"
	}
	if !strings.HasPrefix(tokenString, "eyJ") {
		return false, "token does not start with 'eyJ'"
	}
	if !strings.Contains(tokenString, ".") {
		return false, "token does not contain any '.'"
	}
	return true, ""
}

// Decode and dump token string, trusting without formal verification and validation
func DecodeJwtToken(tokenString string) {

	// A JSON Web Token (JWT) string consists of three parts joined together with dot(.):
	// "<Header>.<Payload>.<Signature>"
	// Header: It indicates the token’s type and which signing algorithm has been used.
	// Payload: It consists of the claims. And claims comprise of application’s data(
	//   email id, username, role), the expiration period of a token (Exp), and so on.
	// Signature: It is generated using the secret (provided by the user), encoded
	// header, and payload.
	//
	// Token struct fields:
	//   Raw       string                 // The raw token. Populated when you Parse a token
	//   Method    SigningMethod          // The signing method used or to be used
	//   Header    map[string]interface{} // The first segment of the token
	//   Claims    Claims                 // The second segment of the token
	//   Signature string                 // The third segment of the token. Populated when you Parse a token
	//   Valid     bool                   // Is the token valid? Populated when you Parse/Verify a token

	valid, errMsg := IsValidTokenFormat(tokenString)
	if !valid {
		utl.Die("%s\n", utl.Red(fmt.Sprintf("Invalid token: %s", errMsg)))
	}

	// Parse the token without verifying the signature
	claims := jwt.MapClaims{} // claims are actually a map[string]interface{}
	token, _ := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		return []byte("<YOUR VERIFICATION KEY>"), nil
	})
	// // Below no yet needed, since this is only printing claims in an unverified way
	// if err != nil {
	// 	fmt.Println(utl.Red("Token is invalid: " + err.Error()))
	// }
	// if token == nil {
	// 	fmt.Println(utl.Red("Error parsing token: " + err.Error()))
	// }

	fmt.Println(utl.Blu("header") + ":")

	sortedKeys := utl.SortObjStringKeys(token.Header)
	for _, k := range sortedKeys {
		v := token.Header[k]
		fmt.Printf("  %s:%s %s\n", utl.Blu(k), utl.PadSpaces(20, len(k)), utl.Gre(v))
	}

	fmt.Println(utl.Blu("claims") + ":")
	sortedKeys = utl.SortObjStringKeys(token.Claims.(jwt.MapClaims))
	for _, k := range sortedKeys {
		v := token.Claims.(jwt.MapClaims)[k]

		switch v := v.(type) {
		case string:
			fmt.Printf("  %s:%s %s\n", utl.Blu(k), utl.PadSpaces(20, len(k)), utl.Gre(v))
		case float64:
			t := time.Unix(int64(v), 0)
			vStr := utl.Yel(t.Format("2006-Jan-02 15:04:05"))
			vStr += utl.Gra(fmt.Sprintf("  # %d", int64(v)))
			fmt.Printf("  %s:%s %s\n", utl.Blu(k), utl.PadSpaces(20, len(k)), vStr)
		case []interface{}:
			vList := v
			vStr := ""
			for _, i := range vList {
				vStr += utl.Str(i) + " "
			}
			fmt.Printf("  %s:%s %s\n", utl.Blu(k), utl.PadSpaces(20, len(k)), utl.Gre(vStr))
		}
	}

	fmt.Println(utl.Blu("signature") + ":")
	if string(token.Signature) != "" {
		k := "signature"
		// Display the base64 encoded signature
		fmt.Printf("  %s:%s %s\n", utl.Blu(k), utl.PadSpaces(20, len(k)),
			utl.Gre(base64.StdEncoding.EncodeToString([]byte(token.Signature))))
	}

	fmt.Println(utl.Blu("status") + ":")
	k := "valid"
	vStr := ""
	if token.Valid {
		vStr = utl.Gre("true")
	} else {
		vStr = utl.Gre("false") + utl.Gra("  # Since this parsing isn't really verifying it")
	}
	fmt.Printf("  %s:%s %s\n", utl.Blu(k), utl.PadSpaces(20, len(k)), vStr)

	os.Exit(0)
}

// Returns DEBUG tokens string
func DebugTokenString(z *Config) string {
	az := utl.Str(z.AzToken)
	if len(az) < 4 {
		az = "AZ_" + "none"
	} else {
		az = "AZ_" + az[len(az)-4:]
	}
	mg := utl.Str(z.MgToken)
	if len(mg) < 4 {
		mg = "MG_" + "none"
	} else {
		mg = "MG_" + mg[len(mg)-4:]
	}
	return fmt.Sprintf("%s %s", az, mg)
}
