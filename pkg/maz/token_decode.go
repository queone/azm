package maz

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/queone/utl"
)

// Decode and validate the given JWT token string, print all decoded fields and final validation status.
func DecodeAndValidateToken(tokenString string) {
	Logf("Decoding and validating the given JWT token string\n")
	parts, err := SplitJWT(tokenString)
	if err != nil {
		utl.Die("Error: %s\n", err)
	}

	PrintTokenComponents(parts)

	fmt.Printf("%s:\n", utl.Blu("status")) // Token status block

	Logf("Verifying token signature\n")
	valid, err := VerifyAzureJwt(tokenString)
	fmt.Printf("  %s: ", utl.Blu("valid")) // Printout will be newlined below
	if valid {
		tokenType := GetApiTokenType(parts)
		if tokenType == MgApiToken {
			msg := "# Token PASSED iss/tid structure verification. Note, it's not signature verification."
			fmt.Printf("%s  %s\n", utl.Gre("true"), utl.Gra(msg))
		} else {
			fmt.Printf("%s  %s\n", utl.Gre("true"), utl.Gra("# Token PASSED signature verification"))
		}
	} else {
		fmt.Printf("%s  %s\n", utl.Red("false"), utl.Gra("# Token FAILED signature verification"))
		Logf("Token signature verification error: %s", utl.Red(err))
	}

	os.Stdout.Sync()
	os.Exit(0)
}

// Decode and print the header, claims, and signature components of the JWT token.
func PrintTokenComponents(parts []string) {
	headerJSON, _ := base64.RawURLEncoding.DecodeString(parts[0])
	claimsJSON, _ := base64.RawURLEncoding.DecodeString(parts[1])
	signature := parts[2]

	var header map[string]interface{}
	var claims map[string]interface{}
	json.Unmarshal(headerJSON, &header)
	json.Unmarshal(claimsJSON, &claims)

	fmt.Printf("%s:\n", utl.Blu("header"))
	for k, v := range header {
		fmt.Printf("  %s: %s\n", utl.Blu(k), utl.Gre(v))
	}

	fmt.Printf("%s:\n", utl.Blu("claims"))
	for k, v := range claims {
		switch k {
		case "exp":
			vInt := utl.Int64(v)
			msg := "# " + UnixDateTimeString(vInt) + " <= Expires"
			fmt.Printf("  %s: %s  %s\n", utl.Blu(k), utl.Gre(vInt), utl.Gra(msg))
		case "nbf":
			vInt := utl.Int64(v)
			msg := "# " + UnixDateTimeString(vInt) + " <= Not before"
			fmt.Printf("  %s: %s  %s\n", utl.Blu(k), utl.Gre(vInt), utl.Gra(msg))
		case "iat":
			vInt := utl.Int64(v)
			msg := "# " + UnixDateTimeString(vInt) + " <= Issued at"
			fmt.Printf("  %s: %s  %s\n", utl.Blu(k), utl.Gre(vInt), utl.Gra(msg))
		case "xms_tcdt":
			vInt := utl.Int64(v)
			msg := "# " + UnixDateTimeString(vInt) + " <= Token Creation DateTime"
			fmt.Printf("  %s: %s  %s\n", utl.Blu(k), utl.Gre(vInt), utl.Gra(msg))
		default:
			fmt.Printf("  %s: %s\n", utl.Blu(k), utl.Gre(v))
		}
	}

	Logf("Printing token signature block\n")
	fmt.Printf("%s: %s  %s\n", utl.Blu("signature"), utl.Gre(signature),
		utl.Gra("# In base64 format"))
}

// Validate the given Azure or MS Graph JWT token, including key fetching and issuer structure.
func VerifyAzureJwt(tokenString string) (bool, error) {
	parts, err := SplitJWT(tokenString)
	if err != nil {
		return false, fmt.Errorf("invalid token format: %w", err)
	}

	headerJSON, _ := base64.RawURLEncoding.DecodeString(parts[0])
	claimsJSON, _ := base64.RawURLEncoding.DecodeString(parts[1])

	var header map[string]interface{}
	var claims map[string]interface{}
	json.Unmarshal(headerJSON, &header)
	json.Unmarshal(claimsJSON, &claims)

	kid, _ := header["kid"].(string)
	iss, _ := claims["iss"].(string)
	aud, _ := claims["aud"].(string)
	tid, _ := claims["tid"].(string)

	jwksURL := fmt.Sprintf("%s/discovery/v2.0/keys", strings.TrimRight(iss, "/"))
	resp, err := http.Get(jwksURL)
	if err != nil {
		return false, fmt.Errorf("failed to fetch JWKS: %w", err)
	}
	defer resp.Body.Close()

	var jwks struct {
		Keys []map[string]interface{} `json:"keys"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&jwks); err != nil {
		return false, fmt.Errorf("failed to decode JWKS: %w", err)
	}

	var pubKey *rsa.PublicKey
	for _, key := range jwks.Keys {
		if key["kid"] == kid {
			if x5cArr, ok := key["x5c"].([]interface{}); ok && len(x5cArr) > 0 {
				decodedCert, _ := base64.StdEncoding.DecodeString(x5cArr[0].(string))
				cert, err := x509.ParseCertificate(decodedCert)
				if err == nil {
					pubKey = cert.PublicKey.(*rsa.PublicKey)
				}
			} else if nStr, okN := key["n"].(string); okN {
				nBytes, _ := base64.RawURLEncoding.DecodeString(nStr)
				eBytes, _ := base64.RawURLEncoding.DecodeString(key["e"].(string))
				n := new(big.Int).SetBytes(nBytes)
				e := int(new(big.Int).SetBytes(eBytes).Int64())
				pubKey = &rsa.PublicKey{N: n, E: e}
			}
			break
		}
	}

	if pubKey == nil {
		return false, fmt.Errorf("no matching public key found for KID: %s", kid)
	}

	tokenType := GetApiTokenType(parts)
	switch tokenType {
	case AzApiToken:
		Logf("Verifying AZ token\n")
		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			return pubKey, nil
		}, jwt.WithAudience(aud), jwt.WithIssuer(iss))

		if err != nil || !token.Valid {
			return false, fmt.Errorf("signature validation failed: %w", err)
		}
		return true, nil
	case MgApiToken:
		Logf("Verifying MG token\n")
		if validateIssuerStructure(iss, tid) {
			return true, nil
		}
		return false, fmt.Errorf("issuer structure or tid validation failed")
	default:
		return false, fmt.Errorf("unrecognized or unsupported audience: %s", aud)
	}
}

// Validate whether the issuer URL structure matches the tenant ID.
func validateIssuerStructure(iss string, tid string) bool {
	// For more details about what is happening here please read below doc
	// https://que.one/azure/ms-token-validation.html and source it references.
	Logf("Validating iss/tid structure\n")
	if tid == "" || iss == "" {
		fmt.Println("Missing tid or iss")
		return false
	}
	if strings.Contains(strings.ToLower(iss), "{tenantid}") {
		expected := strings.ReplaceAll(strings.ToLower(iss), "{tenantid}", strings.ToLower(tid))
		if expected != strings.ToLower(iss) {
			return false
		}
	}
	if !strings.Contains(strings.ToLower(iss), strings.ToLower(tid)) {
		return false
	}
	path := strings.Trim(strings.SplitN(iss, "://", 2)[1], "/")
	segments := strings.Split(path, "/")
	return len(segments) > 0 && strings.EqualFold(segments[len(segments)-1], tid)
}

// Determine whether the JWT is for Azure or Microsoft Graph based on the aud field.
func GetApiTokenType(parts []string) string {
	if len(parts) != 3 {
		return UnknownApiToken
	}

	payloadB64 := parts[1]
	payloadJSON, err := base64.RawURLEncoding.DecodeString(payloadB64)
	if err != nil {
		return UnknownApiToken
	}

	var claims map[string]interface{}
	if err := json.Unmarshal(payloadJSON, &claims); err != nil {
		return UnknownApiToken
	}

	aud, ok := claims["aud"].(string)
	if !ok {
		return UnknownApiToken
	}

	switch aud {
	case "https://management.azure.com":
		return AzApiToken
	case "https://graph.microsoft.com":
		return MgApiToken
	default:
		return UnknownApiToken
	}
}

// Return the last 4 characters of the token string, for display/debugging.
func GetTokenSuffix(tokenString string) string {
	if len(tokenString) < 4 {
		return "none"
	}
	return tokenString[len(tokenString)-4:]
}

// Convert a Unix timestamp (float, int, etc.) to a readable date-time string.
func UnixDateTimeString(exp interface{}) string {
	var ts int64

	switch v := exp.(type) {
	case float64:
		ts = int64(v)
	case int64:
		ts = v
	case int:
		ts = int64(v)
	default:
		return "invalid timestamp type"
	}

	t := time.Unix(ts, 0)
	return t.Format("2006-Jan-02 15:04:05")
}
