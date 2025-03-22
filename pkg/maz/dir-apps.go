package maz

import (
	"fmt"
	"strings"
	"time"

	"github.com/queone/utl"
)

const (
	NeitherExists = iota // 0: Neither App nor SP exists
	OnlySPExists         // 1: Only SP exists
	OnlyAppExists        // 2: Only App exists
	BothExist            // 3: Both App and SP exist
)

// Prints application object in YAML-like format
func PrintApp(x AzureObject, z *Config) {
	id := utl.Str(x["id"])
	if id == "" {
		return
	}

	// Print the most important attributes first
	appDisplayName := utl.Str(x["displayName"])
	fmt.Printf("%s\n", utl.Gra("# Application"))
	fmt.Printf("%s: %s\n", utl.Blu("display_name"), utl.Gre(appDisplayName))
	fmt.Printf("%s: %s\n", utl.Blu("object_id"), utl.Gre(id))
	fmt.Printf("%s: %s\n", utl.Blu("client_id"), utl.Gre(utl.Str(x["appId"])))

	// Print certificates keys
	keyCredentials := utl.Slice(x["keyCredentials"]) // Cast to a slice
	PrintCertificateList(keyCredentials)

	// Print secret list & expiry details, not actual secretText (which cannot be retrieve anyway)
	passwordCredentials := utl.Slice(x["passwordCredentials"]) // Cast to a slice
	PrintSecretList(passwordCredentials)

	// Print federated credentials
	apiUrl := ConstMgUrl + "/v1.0/applications/" + id + "/federatedIdentityCredentials"
	resp, statusCode, _ := ApiGet(apiUrl, z, nil)
	fedCreds := utl.Slice(resp["value"]) // Cast to a slice
	if statusCode == 200 && len(fedCreds) > 0 {
		fmt.Printf("%s:\n", utl.Blu("federated_credentials"))
		for _, item := range fedCreds {
			cred := utl.Map(item) // Try casting to a map
			if cred == nil {
				fmt.Printf("  %s\n", utl.Gre("(unable to read cred)"))
				continue
			}
			iId := utl.Gre(utl.Str(cred["id"]))
			name := utl.Gre(utl.Str(cred["name"]))
			sub := utl.Gre(utl.Str(cred["subject"]))
			iss := utl.Gre(utl.Str(cred["issuer"]))
			var audiences []string
			audList := utl.Slice(cred["audiences"])
			for _, audience := range audList {
				audiences = append(audiences, utl.Str(audience)) // Convert and append to the string slice
			}
			aud := utl.Gre(strings.Join(audiences, ", "))
			// TODO: Fix the coloring padding
			//fmt.Printf("  %-36s  %-40s  %-40s  %-40s  %s\n", iId, name, sub, iss, aud)
			fmt.Printf("  %-36s  %-20s  %s  %s  %s\n", iId, name, sub, iss, aud)
		}
	}

	// Print any owners
	apiUrl = ConstMgUrl + "/beta/applications/" + id + "/owners"
	resp, statusCode, _ = ApiGet(apiUrl, z, nil)
	owners := utl.Slice(resp["value"]) // Cast to a slice
	if statusCode == 200 {
		PrintOwners(owners)
	}

	// Print any oAuth2 permission scopes
	api := utl.Map(x["api"]) // Cast to a map
	if api != nil {
		oauth2PermissionScopes := utl.Slice(api["oauth2PermissionScopes"]) // Cast to a slice
		scopeValueMap := make(map[string]string)
		if len(oauth2PermissionScopes) > 0 {
			fmt.Printf("%s:\n", utl.Blu("oauth2_permission_scopes"))
			for _, item := range oauth2PermissionScopes {
				if scope := utl.Map(item); scope != nil {
					// Process if casting to a map works
					scopeId := utl.Str(scope["id"])
					enabledStat := "Disabled"
					if utl.Str(scope["isEnabled"]) == "true" {
						enabledStat = "Enabled"
					}
					apiName := appDisplayName
					scopeType := "Delegated"
					scopeValue := utl.Str(scope["value"])
					scopeValueMap[scopeId] = scopeValue // Keep building scopeValueMap (to be used for preAuthApp below)
					fmt.Printf("  %s%s  %s%s  %s%s  %s%s  %s\n",
						utl.Gre(scopeId), utl.PadSpaces(38, len(scopeId)),
						utl.Gre(enabledStat), utl.PadSpaces(10, len(enabledStat)),
						utl.Gre(apiName), utl.PadSpaces(50, len(apiName)),
						utl.Gre(scopeType), utl.PadSpaces(12, len(scopeType)),
						utl.Gre(scopeValue))
				}
			}
		}
		// Also print any pre-authorized applications
		preAuthorizedApplications := utl.Slice(api["preAuthorizedApplications"]) // Cast to a slice
		if len(preAuthorizedApplications) > 0 {
			fmt.Printf("%s:\n", utl.Blu("  pre_authorized_applications"))
			for _, item := range preAuthorizedApplications {
				app := utl.Map(item)
				clientId := utl.Str(app["appId"])
				permissionIds := utl.Slice(app["permissionIds"])
				if len(permissionIds) > 0 {
					for _, j := range permissionIds {
						scopeId := utl.Str(j)
						scopeValue := scopeValueMap[scopeId]
						fmt.Printf("    %s%s  %s\n",
							utl.Gre(clientId), utl.PadSpaces(38, len(clientId)),
							utl.Gre(scopeValue))
					}
				}
			}
		}
	}

	// Print API permissions that have been ASSIGNED to this application
	// - https://learn.microsoft.com/en-us/entra/identity-platform/app-objects-and-service-principals?tabs=browser
	// - https://learn.microsoft.com/en-us/entra/identity-platform/permissions-consent-overview
	// Just look under the object's 'requiredResourceAccess' attribute
	APIs := utl.Slice(x["requiredResourceAccess"]) // Cast to a slice
	if len(APIs) > 0 {
		fmt.Printf("%s:\n", utl.Blu("api_permissions_assigned"))
		for _, item := range APIs {
			api := utl.Map(item) // Try casting to a map
			// Getting this API's name and permission value such as Directory.Read.All is a 2-step process:
			// 1) Get all the roles for given API and put their id/value pairs in a map, then
			// 2) Use that map to enumerate and print them

			// Let's drill down into the permissions for this API
			resAppId := utl.Str(api["resourceAppId"])
			if resAppId == "" {
				fmt.Printf("  %-50s %s\n", "Unknown API", "Missing resourceAppId")
				continue // Skip this API, move to next one
			}

			// Get this API's SP object with all relevant attributes
			params := map[string]string{"$filter": "appId eq '" + resAppId + "'"}
			apiUrl := ConstMgUrl + "/beta/servicePrincipals"
			resp, _, _ := ApiGet(apiUrl, z, params)
			SPs := utl.Slice(resp["value"]) // Cast to a slice
			// It's a list because this could be a multi-tenant app, having multiple SPs
			// TODO: Handle multiple SPs
			if len(SPs) > 1 {
				utl.Die("  %-50s %s\n", resAppId, "Error. Multiple SPs for this AppId. Aborting.")
			} else if len(SPs) < 1 {
				fmt.Printf("  %-50s %s\n", resAppId, "Unable to get Resource App object. Skipping this API.")
				continue
			}

			// Currently only handling the expected single-tenant entry
			sp := utl.Map(SPs[0]) // Try casting to a map

			// 1. Put all API role id:name pairs into roleMap list
			roleMap := make(map[string]string)
			// These are for Application types
			appRoles := utl.Slice(sp["appRoles"])
			for _, item := range appRoles {
				if role := utl.Map(item); role != nil {
					id := utl.Str(role["id"])
					value := utl.Str(role["value"])
					if id != "" && value != "" {
						roleMap[id] = value // Add entry to map
					}
				}
			}
			// These are for Delegated types
			publishedPermissionScopes := utl.Slice(sp["publishedPermissionScopes"])
			for _, item := range publishedPermissionScopes {
				if role := utl.Map(item); role != nil {
					id := utl.Str(role["id"])
					value := utl.Str(role["value"])
					if id != "" && value != "" {
						roleMap[id] = value // Add entry to map
					}
				}
			}
			if len(roleMap) < 1 {
				fmt.Printf("  %-50s %s\n", resAppId, "Error getting list of appRoles.")
				continue
			}

			// 2. Parse this app permissions, and use roleMap to display permission value
			Perms := utl.Slice(api["resourceAccess"])
			if len(Perms) > 0 {
				apiName := utl.Str(sp["displayName"]) // This API's name
				for _, item := range Perms {          // Iterate through perms
					if perm := utl.Map(item); perm != nil {
						pid := utl.Str(perm["id"])
						var pType string = "?"
						if utl.Str(perm["type"]) == "Role" {
							pType = "Application"
						} else {
							pType = "Delegated"
						}
						fmt.Printf("  %s%s  %s%s  %s\n", utl.Gre(apiName), utl.PadSpaces(40, len(apiName)),
							utl.Gre(pType), utl.PadSpaces(14, len(pType)), utl.Gre(roleMap[pid]))
					}
				}
			} else {
				fmt.Printf("  %-50s %s\n", resAppId, "Error getting list of appRoles.")
			}
		}
	}
}

// Checks to see whether the App and SP objects exist. Another preprocessing helper function.
func CheckAppSpExistence(identifier string, z *Config) (app, sp AzureObject, code int) {
	// Check if the App exists
	app = PreFetchAzureObject("ap", identifier, z)
	if app != nil {
		// App exists
		appId := utl.Str(app["appId"])

		// Use appId/ClientID to check if the corresponding SP exists
		sp = PreFetchAzureObject("sp", appId, z)
		if sp != nil {
			return app, sp, BothExist // Both exist
		}
		return app, nil, OnlyAppExists // Only App exists
	}

	// App does not exist, check if SP exists
	sp = PreFetchAzureObject("sp", identifier, z)
	if sp != nil {
		// SP exists, check if its associated App exists using its appId/ClientID
		appId := utl.Str(sp["appId"])
		app = PreFetchAzureObject("ap", appId, z)
		if app != nil {
			return app, sp, BothExist // Both exist
		}
		return nil, sp, OnlySPExists // Only SP exists
	}

	// Neither exists
	return nil, nil, NeitherExists
}

// Creates an App/SP object pair by name, if they don't already exist.
func CreateAppSpByName(force bool, displayName string, z *Config) error {
	app, sp, state := CheckAppSpExistence(displayName, z)
	switch state {
	case NeitherExists:
		// Create both App and SP
		obj := AzureObject{
			"displayName":    displayName,
			"signInAudience": "AzureADMyOrg",
		}

		// Confirmation
		utl.PrintYamlColor(obj)
		if !force {
			msg := utl.Yel("Create App/SP pair with above parameters? y/n ")
			if utl.PromptMsg(msg) != 'y' {
				utl.Die("%s\n", "Operation aborted by user")
			}
		} else {
			fmt.Println("Creating App/SP pair with above parameters...")
		}

		app, err := CreateDirObjectInAzure("ap", obj, z)
		if err != nil {
			utl.Die("%s\n", err.Error())
		}
		appId := utl.Str(app["appId"])
		spObj := AzureObject{"appId": appId}
		_, err = CreateDirObjectInAzure("sp", spObj, z)
		if err != nil {
			utl.Die("%s\n", err.Error())
		}
	case OnlySPExists:
		idSp := utl.Str(sp["id"])
		appId := utl.Str(sp["appId"])
		utl.Die("SP (%s) named '%s' exists, and the associated AppID/ClientID is %s.\n",
			idSp, displayName, appId)
	case OnlyAppExists:
		idApp := utl.Str(app["id"])
		appId := utl.Str(app["appId"])
		fmt.Printf("App (%s) named '%s' exists, its AppID/ClientID is %s.\n", idApp, displayName, appId)
		if !force {
			msg := utl.Yel("Create corresponding SP? y/n ")
			if utl.PromptMsg(msg) != 'y' {
				utl.Die("%s\n", "Operation aborted by user")
			}
		} else {
			fmt.Println("Creating corresponding SP...")
		}
		spObj := AzureObject{"appId": appId}
		_, err := CreateDirObjectInAzure("sp", spObj, z)
		if err != nil {
			utl.Die("%s\n", err.Error())
		}
	case BothExist:
		idApp := utl.Str(app["id"])
		appId := utl.Str(app["appId"])
		idSp := utl.Str(sp["id"])
		utl.Die("Both App (%s) and SP (%s) named '%s' exist. They share appId '%s'.\n", idApp, idSp, displayName, appId)
	default:
		return fmt.Errorf("unexpected app/sp existence state")
	}

	return nil
}

// Deletes Azure AppSP pair from given command-line arguments.
func DeleteAppSpByIdentifier(force bool, identifier string, z *Config) {
	app, sp, state := CheckAppSpExistence(identifier, z)
	switch state {
	case NeitherExists:
		utl.Die("No App or SP found with identifier '%s'\n", identifier)
	case OnlySPExists:
		// Delete SP only
		// Confirmation prompt
		utl.PrintYamlColor(sp.TrimForCache("sp"))
		if !force {
			msg := utl.Yel("Delete above SP? y/n ")
			if utl.PromptMsg(msg) != 'y' {
				utl.Die("Operation aborted by user.\n")
			}
		} else {
			fmt.Println("Deleting above SP...")
		}
		idSp := utl.Str(sp["id"])
		DeleteDirObjectInAzure("sp", idSp, z)
	case OnlyAppExists:
		// Delete App only
		// Confirmation prompt
		utl.PrintYamlColor(app.TrimForCache("ap"))
		if !force {
			msg := utl.Yel("Delete above App? y/n ")
			if utl.PromptMsg(msg) != 'y' {
				utl.Die("Operation aborted by user.\n")
			}
		} else {
			fmt.Println("Deleting above App...")
		}
		idApp := utl.Str(app["id"])
		DeleteDirObjectInAzure("ap", idApp, z)
	case BothExist:
		// Delete both
		utl.PrintYamlColor(app.TrimForCache("ap"))
		fmt.Println(utl.Gra("and corresponding SP..."))
		utl.PrintYamlColor(sp.TrimForCache("sp"))
		if !force {
			msg := utl.Yel("Delete above App/SP pair? y/n ")
			if utl.PromptMsg(msg) != 'y' {
				utl.Die("Operation aborted by user.\n")
			}
		} else {
			fmt.Println("Deleting above App/SP pair...")
		}
		idSp := utl.Str(sp["id"])
		idApp := utl.Str(app["id"])
		DeleteDirObjectInAzure("sp", idSp, z)
		DeleteDirObjectInAzure("ap", idApp, z)
	default:
		utl.Die("Unexpected App/SP existence state.\n")
	}
}

// Renames Azure App/SP pair
func RenameAppSp(force bool, identifier, newName string, z *Config) {
	app, sp, state := CheckAppSpExistence(identifier, z)
	switch state {
	case NeitherExists:
		utl.Die("No App or SP found with identifier '%s'\n", identifier)
	case OnlySPExists:
		// Rename SP only
		idSp := utl.Str(sp["id"])
		// Confirmation prompt
		if !force {
			displayName := utl.Str(sp["displayName"])
			msg := utl.Yel("Rename SP "+idSp+"\n  from \"") + utl.Blu(displayName) +
				utl.Yel("\"\n    to \"") + utl.Blu(newName) + utl.Yel("\"\n? y/n ")
			if utl.PromptMsg(msg) != 'y' {
				utl.Die("Aborted.\n")
			}
		} else {
			fmt.Println("Renaming SP...")
		}
		obj := AzureObject{"displayName": newName}
		if err := UpdateDirObjectInAzure("sp", idSp, obj, z); err != nil {
			utl.Die("%s", err.Error())
		}
	case OnlyAppExists:
		// Rename App only
		idApp := utl.Str(app["id"])
		if !force {
			displayName := utl.Str(app["displayName"])
			msg := utl.Yel("Rename App "+idApp+"\n  from \"") + utl.Blu(displayName) +
				utl.Yel("\"\n    to \"") + utl.Blu(newName) + utl.Yel("\"\n? y/n ")
			if utl.PromptMsg(msg) != 'y' {
				utl.Die("Aborted.\n")
			}
		} else {
			fmt.Println("Renaming App...")
		}
		obj := AzureObject{"displayName": newName}
		if err := UpdateDirObjectInAzure("ap", idApp, obj, z); err != nil {
			utl.Die("%s", err.Error())
		}
	case BothExist:
		// Rename both
		idApp := utl.Str(app["id"])
		appId := utl.Str(app["appId"])
		idSp := utl.Str(sp["id"])
		if !force {
			displayName := utl.Str(app["displayName"])
			msg := utl.Yel("Rename App/SP pair with appId "+appId+"\n  from \"") + utl.Blu(displayName) +
				utl.Yel("\"\n    to \"") + utl.Blu(newName) + utl.Yel("\"\n? y/n ")
			if utl.PromptMsg(msg) != 'y' {
				utl.Die("Aborted.\n")
			}
		} else {
			fmt.Println("Renaming App/SP pair...")
		}
		obj := AzureObject{"displayName": newName}
		if err := UpdateDirObjectInAzure("ap", idApp, obj, z); err != nil {
			utl.Die("%s", err.Error())
		}
		if err := UpdateDirObjectInAzure("sp", idSp, obj, z); err != nil {
			utl.Die("%s", err.Error())
		}
	default:
		utl.Die("Unexpected App/SP existence state.\n")
	}
}

// Creates or updates an Azure App/SP pair from given specfile.
func UpsertAppSpFromFile(force bool, filePath string, z *Config) {
	// Abort if specfile is not YAML or does not have a valid AppSP defininition
	formatType, t, mapObj := GetObjectFromFile(filePath)
	if formatType != YamlFormat {
		utl.Die("File is not YAML\n")
	}

	// Capture the object defined in the specfile: Note that we need to treat all attributes
	// in this object as applying to *both* the App and the SP, as long as it's consistent
	// with the MS Graph API.
	obj := AzureObject(mapObj)

	if obj == nil {
		utl.Die("Specfile does not contain a valid App/SP definition.\n")
	}
	if t != "ap" {
		utl.Die("Object defined in specfile is not an App/SP pair.\n")
	}

	// Cannot continue without at least the displayName and signInAudience
	displayName := utl.Str(obj["displayName"])
	signInAudience := utl.Str(obj["displayName"])
	msg := "Specfile object is missing"
	if displayName == "" {
		utl.Die("%s %s\n", msg, utl.Red("displayName"))
	}
	if signInAudience == "" {
		utl.Die("%s %s\n", msg, utl.Red("signInAudience"))
	}

	// Check if either the App or the SP exist and process this specfile accordingly
	app, sp, state := CheckAppSpExistence(displayName, z)
	switch state {
	case NeitherExists:
		// So let's create them both
		utl.PrintYamlColor(obj)
		if !force {
			msg := utl.Yel("Create App/SP pair with above parameters? y/n ")
			if utl.PromptMsg(msg) != 'y' {
				utl.Die("%s\n", "Operation aborted by user")
			}
		} else {
			fmt.Println("Creating App/SP pair with above parameters...")
		}

		app, err := CreateDirObjectInAzure("ap", obj, z)
		if err != nil {
			utl.Die("%s\n", err.Error())
		}
		appId := utl.Str(app["appId"])
		spObj := AzureObject{"appId": appId}
		_, err = CreateDirObjectInAzure("sp", spObj, z)
		if err != nil {
			utl.Die("%s\n", err.Error())
		}
	case OnlySPExists:
		// So let's update the SP and create the App?
		idSp := utl.Str(sp["id"])
		fmt.Printf("There's an existing SP (%s) named '%s'.\n", idSp, displayName)
		utl.Die("%s\n", "This condition is not supported. Aborting.")
	case OnlyAppExists:
		// So let's update the App and create the SP
		idApp := utl.Str(app["id"])
		fmt.Printf("There's an existing App (%s) named '%s'.\n", idApp, displayName)
		utl.Die("%s\n", "This condition is not supported. Aborting.")
	case BothExist:
		// So let's update them both
		idApp := utl.Str(app["id"])
		idSp := utl.Str(sp["id"])
		UpdateDirObject(force, idApp, obj, "ap", z)
		UpdateDirObject(force, idSp, obj, "sp", z)
	default:
		utl.Die("Unexpected App/SP existence state.\n")
	}
}

// Helper function to check if the object is an App Service Principal
func IsAppSp(obj AzureObject) bool {
	displayName := utl.Str(obj["displayName"])
	signInAudience := utl.Str(obj["signInAudience"])
	return displayName != "" && signInAudience != ""
}

// Adds a new secret to the given App or SP
func AddAppSpSecret(mazType, id, displayName, expiry string, z *Config) {
	if mazType != Application && mazType != ServicePrincipal {
		utl.Die("Error: Secrets can only be added to an App or SP object.\n")
	}
	x := GetObjectFromAzureById(mazType, id, z)
	if x == nil {
		utl.Die("No %s with that ID.\n", MazTypeNames[mazType])
	}

	// Check if a password with the same displayName already exists
	object_id := utl.Str(x["id"]) // NOTE: We call Azure with the OBJECT ID
	apiUrl := ConstMgUrl + ApiEndpoint[mazType] + "/" + object_id + "/passwordCredentials"
	resp, statCode, _ := ApiGet(apiUrl, z, nil)
	if statCode == 200 {
		passwordCredentials := utl.Slice(resp["value"])
		for _, credential := range passwordCredentials {
			credentialMap := utl.Map(credential)
			if credentialMap != nil {
				if utl.Str(credentialMap["displayName"]) == displayName {
					utl.Die("A password named %s already exists.\n", utl.Yel(displayName))
				}
			}
		}
	}

	// Setup expiry for endDateType payload variable
	var endDateTime string
	if expiry != "" {
		if utl.ValidDate(expiry, "2006-01-02") {
			// If user-supplied expiry is a valid date, reformat and use for our purpose
			var err error
			endDateTime, err = utl.ConvertDateFormat(expiry, "2006-01-02", time.RFC3339Nano)
			if err != nil {
				utl.Die("Error converting %s Expiry to RFC3339Nano/ISO8601 format.\n", utl.Yel(expiry))
			}
		} else if days, err := utl.StringToInt64(expiry); err == nil {
			// If expiry not a valid date, see if it's a valid integer number
			expiryTime := utl.GetDateInDays(utl.Int64ToString(days)) // Set expiryTime to 'days' from now
			endDateTime = expiryTime.Format(time.RFC3339Nano)        // Convert to RFC3339Nano/ISO8601 format
		} else {
			utl.Die("Invalid expiry format. Please use YYYY-MM-DD or number of days.\n")
		}
	} else {
		// If expiry is blank, default to 365 days from now
		endDateTime = time.Now().AddDate(0, 0, 365).Format(time.RFC3339Nano)
	}

	// Call Azure to create the new secret
	payload := AzureObject{
		"passwordCredential": map[string]string{
			"displayName": displayName,
			"endDateTime": endDateTime,
		},
	}
	apiUrl = ConstMgUrl + ApiEndpoint[mazType] + "/" + object_id + "/addPassword"
	resp, statCode, _ = ApiPost(apiUrl, z, payload, nil)
	if statCode == 200 {
		if mazType == Application {
			fmt.Printf("%s: %s\n", utl.Blu("app_object_id"), utl.Gre(object_id))
		} else {
			fmt.Printf("%s: %s\n", utl.Blu("sp_object_id"), utl.Gre(object_id))
		}
		fmt.Printf("%s: %s\n", utl.Blu("new_secret_id"), utl.Gre(utl.Str(resp["keyId"])))
		fmt.Printf("%s: %s\n", utl.Blu("new_secret_name"), utl.Gre(displayName))
		fmt.Printf("%s: %s\n", utl.Blu("new_secret_expiry"), utl.Gre(expiry))
		fmt.Printf("%s: %s\n", utl.Blu("new_secret_text"), utl.Gre(utl.Str(resp["secretText"])))
	} else {
		msg := fmt.Sprintf("HTTP %d: %s", statCode, ApiErrorMsg(resp))
		utl.Die("%s\n", utl.Red(msg))
	}
}

// Removes a secret from the given App or SP object
func RemoveAppSpSecret(mazType, id, keyId string, force bool, z *Config) {
	// TODO: Needs a prompt/force option
	if mazType != Application && mazType != ServicePrincipal {
		utl.Die("Error: Secrets can only be removed from an App or SP object.\n")
	}
	x := GetObjectFromAzureById(mazType, id, z)
	if x == nil {
		utl.Die("No %s with that ID.\n", MazTypeNames[mazType])
	}
	if !utl.ValidUuid(keyId) {
		utl.Die("Secret ID is not a valid UUID.\n")
	}

	// Display object secret details, and prompt for delete confirmation
	pwdCreds := utl.Slice(x["passwordCredentials"]) // Try casting to a slice
	if len(pwdCreds) < 1 {
		utl.Die("App object has no secrets.\n")
	}
	var a AzureObject = nil // Target keyId, Secret ID to be deleted
	for _, item := range pwdCreds {
		if targetKeyId := utl.Map(item); targetKeyId != nil {
			if utl.Str(targetKeyId["keyId"]) == keyId {
				a = targetKeyId
				break
			}
		}
	}
	if a == nil {
		utl.Die("App object does not have this Secret ID.\n")
	}
	cId := utl.Str(a["keyId"])
	cName := utl.Str(a["displayName"])
	cHint := utl.Str(a["hint"]) + "********"
	cStart, err := utl.ConvertDateFormat(utl.Str(a["startDateTime"]), time.RFC3339Nano, "2006-01-02")
	if err != nil {
		utl.Die("%s %s\n", utl.Trace(), err.Error())
	}
	cExpiry, err := utl.ConvertDateFormat(utl.Str(a["endDateTime"]), time.RFC3339Nano, "2006-01-02")
	if err != nil {
		utl.Die("%s %s\n", utl.Trace(), err.Error())
	}

	// Prompt
	fmt.Printf("%s: %s\n", utl.Blu("id"), utl.Gre(utl.Str(x["id"])))
	fmt.Printf("%s: %s\n", utl.Blu("appId"), utl.Gre(utl.Str(x["appId"])))
	fmt.Printf("%s: %s\n", utl.Blu("displayName"), utl.Gre(utl.Str(x["displayName"])))
	fmt.Printf("%s:\n", utl.Yel("secret_to_be_deleted"))
	fmt.Printf("  %-36s  %-30s  %-16s  %-16s  %s\n", utl.Yel(cId), utl.Yel(cName),
		utl.Yel(cHint), utl.Yel(cStart), utl.Yel(cExpiry))
	if utl.PromptMsg(utl.Yel("DELETE above? y/n ")) == 'y' {
		payload := AzureObject{"keyId": keyId}
		object_id := utl.Str(x["id"]) // NOTE: We call Azure with the OBJECT ID
		apiUrl := ConstMgUrl + ApiEndpoint[mazType] + "/" + object_id + "/removePassword"
		resp, statCode, _ := ApiPost(apiUrl, z, payload, nil)
		if statCode == 204 {
			utl.Die("Successfully deleted secret.\n")
		} else {
			msg := fmt.Sprintf("HTTP %d: %s", statCode, ApiErrorMsg(resp))
			utl.Die("%s\n", utl.Red(msg))
		}
	} else {
		utl.Die("Aborted.\n")
	}
}
