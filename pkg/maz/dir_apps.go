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
	fmt.Printf("%s\n", utl.Gra("# Application"))
	displayName := utl.Str(x["displayName"])
	fmt.Printf("%s: %s\n", utl.Blu("displayName"), utl.Gre(displayName))
	fmt.Printf("%s: %s\n", utl.Blu("id"), utl.Gre(id))
	fmt.Printf("%s: %s\n", utl.Blu("appId"), utl.Gre(utl.Str(x["appId"])))

	// Print certificates details
	keyCredentials := utl.Slice(x["keyCredentials"])
	PrintCertificateList(keyCredentials)

	// Print secrets details
	passwordCredentials := utl.Slice(x["passwordCredentials"])
	PrintSecretList(passwordCredentials)

	// Print federated credentials details
	PrintFederatedCredentials(id, z)

	// Print any owners
	apiUrl := ConstMgUrl + "/beta/applications/" + id + "/owners"
	resp, statCode, _ := ApiGet(apiUrl, z, nil)
	if statCode != 200 {
		Logf("%s\n", utl.Red2(fmt.Sprintf("HTTP %d: %s", statCode, ApiErrorMsg(resp))))
	}
	owners := utl.Slice(resp["value"]) // Cast to a slice
	if statCode == 200 {
		PrintOwners(owners)
	}

	// Print OAuth2 permission scopes
	api := utl.Map(x["api"])
	PrintOAuth2PermissionScopes(api, displayName)

	// Print API permissions that have already been assigned to this application
	// Just look under the object's 'requiredResourceAccess' attribute
	requiredResourceAccess := utl.Slice(x["requiredResourceAccess"])
	PrintAssignedApiPermissions(requiredResourceAccess, z)
}

// Prints federated credentials list stanza for App objects
func PrintFederatedCredentials(id string, z *Config) {
	apiUrl := ConstMgUrl + "/v1.0/applications/" + id + "/federatedIdentityCredentials"
	resp, statCode, _ := ApiGet(apiUrl, z, nil)
	if statCode != 200 {
		Logf("%s\n", utl.Red2(fmt.Sprintf("HTTP %d: %s", statCode, ApiErrorMsg(resp))))
	}
	fedCreds := utl.Slice(resp["value"]) // Cast to a slice
	if statCode == 200 && len(fedCreds) > 0 {
		fmt.Printf("%s:\n", utl.Blu("federated_credentials"))
		for _, item := range fedCreds {
			cred := utl.Map(item) // Try casting to a map
			if cred == nil {
				fmt.Printf("  - %s\n", utl.Gre("(unable to read credential)"))
				continue
			}
			iId := utl.Gre(utl.Str(cred["id"]))
			name := utl.Gre(utl.Str(cred["name"]))
			sub := utl.Gre(utl.Str(cred["subject"]))
			iss := utl.Gre(utl.Str(cred["issuer"]))

			// Derive audiences string
			var audiences []string
			audList := utl.Slice(cred["audiences"])
			for _, audience := range audList {
				audiences = append(audiences, utl.Str(audience)) // Convert and append to the string slice
			}
			aud := utl.Gre(strings.Join(audiences, ", "))

			fmt.Printf("  - %s: %s\n    %s: %s\n    %s: %s\n    %s: %s\n    %s: %s\n",
				utl.Blu("id"), iId, utl.Blu("name"), name, utl.Blu("subject"), sub,
				utl.Blu("issuer"), iss, utl.Blu("audiences"), aud)

			// Old printout style
			//fmt.Printf("  %-36s  %-20s  %s  %s  %s\n", iId, name, sub, iss, aud)
		}
	}
}

// Prints OAuth2 permission scopes and pre-authorized applications
func PrintOAuth2PermissionScopes(api map[string]interface{}, displayName string) {
	oauth2PermissionScopes := utl.Slice(api["oauth2PermissionScopes"]) // Cast to a slice
	scopeValueMap := make(map[string]string)
	if len(oauth2PermissionScopes) > 0 {
		fmt.Printf("%s:\n", utl.Blu("oauth2_permission_scopes"))
		for _, item := range oauth2PermissionScopes {
			if scope := utl.Map(item); scope != nil {
				// Process if casting to a map works
				scopeId := utl.Str(scope["id"])
				isEnabled := utl.Bool(scope["isEnabled"])

				// enabledStat := "Disabled"
				// if utl.Str(scope["isEnabled"]) == "true" {
				// 	enabledStat = "Enabled"
				// }

				apiName := displayName
				scopeType := "Delegated"
				scopeValue := utl.Str(scope["value"])
				scopeValueMap[scopeId] = scopeValue
				// Keep building scopeValueMap (to be used for preAuthApp below)

				fmt.Printf("  - %s: %s\n    %s: %s\n    %s: %s\n    %s: %s\n    %s: %s\n",
					utl.Blu("id"), utl.Gre(scopeId), utl.Blu("isEnabled"), utl.Gre(isEnabled),
					utl.Blu("apiName"), utl.Gre(apiName), utl.Blu("type"), utl.Gre(scopeType),
					utl.Blu("value"), utl.Gre(scopeValue))

				// Old printout style
				// fmt.Printf("  %s%s  %s%s  %s%s  %s%s  %s\n",
				// 	utl.Gre(scopeId), utl.PadSpaces(38, len(scopeId)),
				// 	utl.Gre(enabledStat), utl.PadSpaces(10, len(enabledStat)),
				// 	utl.Gre(apiName), utl.PadSpaces(50, len(apiName)),
				// 	utl.Gre(scopeType), utl.PadSpaces(12, len(scopeType)),
				// 	utl.Gre(scopeValue))
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

					fmt.Printf("    - %s: %s\n      %s: %s\n",
						utl.Blu("appId"), utl.Gre(clientId), utl.Blu("permissionIds"), utl.Gre(scopeValue))

					// Old printout style
					// fmt.Printf("    %s%s  %s\n",
					// 	utl.Gre(clientId), utl.PadSpaces(38, len(clientId)),
					// 	utl.Gre(scopeValue))
				}
			}
		}
	}
}

// Prints API permissions that have already been assigned to this application
func PrintAssignedApiPermissions(requiredResourceAccess interface{}, z *Config) {
	// learn.microsoft.com/en-us/entra/identity-platform/app-objects-and-service-principals
	// learn.microsoft.com/en-us/entra/identity-platform/permissions-consent-overview

	APIs := utl.Slice(requiredResourceAccess) // Cast to a slice
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
			resp, statCode, _ := ApiGet(apiUrl, z, params)
			if statCode != 200 {
				Logf("%s\n", utl.Red2(fmt.Sprintf("HTTP %d: %s", statCode, ApiErrorMsg(resp))))
			}
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
	app = PreFetchAzureObject(Application, identifier, z)
	if app != nil {
		// App exists
		appId := utl.Str(app["appId"])

		// Use appId/ClientID to check if the corresponding SP exists
		sp = PreFetchAzureObject(ServicePrincipal, appId, z)
		if sp != nil {
			return app, sp, BothExist // Both exist
		}
		return app, nil, OnlyAppExists // Only App exists
	}

	// App does not exist, check if SP exists
	sp = PreFetchAzureObject(ServicePrincipal, identifier, z)
	if sp != nil {
		// SP exists, check if its associated App exists using its appId/ClientID
		appId := utl.Str(sp["appId"])
		app = PreFetchAzureObject(Application, appId, z)
		if app != nil {
			return app, sp, BothExist // Both exist
		}
		return nil, sp, OnlySPExists // Only SP exists
	}

	// Neither exists
	return nil, nil, NeitherExists
}

// Creates an App/SP object pair by name, if they don't already exist.
func CreateAppSpByName(force bool, displayName string, z *Config) {
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
			fmt.Printf("Creating App/SP pair with above parameters...\n")
		}

		app := CreateDirObjectInAzure(Application, obj, z)
		if app == nil {
			utl.Die("%s creating the App object.\n", utl.Red("Error"))
		}
		appId := utl.Str(app["appId"])
		spObj := AzureObject{"appId": appId}
		sp := CreateDirObjectInAzure(ServicePrincipal, spObj, z)
		if sp == nil {
			utl.Die("%s creating the SP object.\n", utl.Red("Error"))
		}
	case OnlySPExists:
		idSp := utl.Str(sp["id"])
		appId := utl.Str(sp["appId"])
		utl.Die("SP (%s) named '%s' exists, and the associated AppID/ClientID is %s.\n",
			idSp, utl.Yel(displayName), appId)
	case OnlyAppExists:
		idApp := utl.Str(app["id"])
		appId := utl.Str(app["appId"])
		fmt.Printf("App (%s) named '%s' exists, its AppID/ClientID is %s.\n", idApp,
			utl.Yel(displayName), appId)
		if !force {
			msg := utl.Yel("Create corresponding SP? y/n ")
			if utl.PromptMsg(msg) != 'y' {
				utl.Die("Operation aborted by user.\n")
			}
		} else {
			fmt.Println("Creating corresponding SP...")
		}
		spObj := AzureObject{"appId": appId}
		sp := CreateDirObjectInAzure(ServicePrincipal, spObj, z)
		if sp == nil {
			utl.Die("Error creating the SP\n")
		}
	case BothExist:
		idApp := utl.Str(app["id"])
		appId := utl.Str(app["appId"])
		idSp := utl.Str(sp["id"])
		utl.Die("Both App (%s) and SP (%s) named '%s' exist. They share appId '%s'.\n",
			idApp, idSp, utl.Yel(displayName), appId)
	default:
		utl.Die("Unexpected app/sp existence state")
	}
}

// Deletes Azure AppSP pair from given indentifier
func DeleteAppSp(force bool, identifier string, z *Config) {
	app, sp, state := CheckAppSpExistence(identifier, z)
	switch state {
	case NeitherExists:
		utl.Die("No App or SP found with identifier '%s'\n", identifier)
	case OnlySPExists:
		// Delete SP only
		// Confirmation prompt
		utl.PrintYamlColor(sp.TrimForCache(ServicePrincipal))
		if !force {
			msg := utl.Yel("Delete above SP? y/n ")
			if utl.PromptMsg(msg) != 'y' {
				utl.Die("Operation aborted by user.\n")
			}
		} else {
			fmt.Println("Deleting above SP...")
		}
		idSp := utl.Str(sp["id"])
		err := DeleteDirObjectInAzure(ServicePrincipal, idSp, z)
		if err != nil {
			fmt.Println(err)
		}
	case OnlyAppExists:
		// Delete App only
		// Confirmation prompt
		utl.PrintYamlColor(app.TrimForCache(Application))
		if !force {
			msg := utl.Yel("Delete above App? y/n ")
			if utl.PromptMsg(msg) != 'y' {
				utl.Die("Operation aborted by user.\n")
			}
		} else {
			fmt.Println("Deleting above App...")
		}
		idApp := utl.Str(app["id"])
		err := DeleteDirObjectInAzure(Application, idApp, z)
		if err != nil {
			fmt.Println(err)
		}
	case BothExist:
		// Delete both
		utl.PrintYamlColor(app.TrimForCache(Application))
		fmt.Println(utl.Gra("and corresponding SP..."))
		utl.PrintYamlColor(sp.TrimForCache(ServicePrincipal))
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
		err := DeleteDirObjectInAzure(ServicePrincipal, idSp, z)
		if err != nil {
			fmt.Println(err)
		}
		err = DeleteDirObjectInAzure(Application, idApp, z)
		if err != nil {
			fmt.Println(err)
		}
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
		err := UpdateDirObjectInAzure(ServicePrincipal, idSp, obj, z)
		if err != nil {
			fmt.Println(err)
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
		err := UpdateDirObjectInAzure(Application, idApp, obj, z)
		if err != nil {
			fmt.Println(err)
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
		err := UpdateDirObjectInAzure(Application, idApp, obj, z)
		if err != nil {
			fmt.Println(err)
		}
		err = UpdateDirObjectInAzure(ServicePrincipal, idSp, obj, z)
		if err != nil {
			fmt.Println(err)
		}
	default:
		utl.Die("Unexpected App/SP existence state.\n")
	}
}

// Creates or updates an Azure App/SP pair from given object
func UpsertAppSp(force bool, obj AzureObject, z *Config) {
	// For the moment, all attributes in this object apply to *both* the App and the SP,
	// as long as they are consistent with the MS Graph API.

	// Cannot continue without at least the displayName and signInAudience
	displayName := utl.Str(obj["displayName"])
	signInAudience := utl.Str(obj["displayName"])
	if displayName == "" {
		utl.Die("Object is missing %s\n", utl.Red("displayName"))
	}
	if signInAudience == "" {
		utl.Die("Object is missing %s\n", utl.Red("signInAudience"))
	}

	// Check if either the App or the SP exist and process accordingly
	app, sp, state := CheckAppSpExistence(displayName, z)
	switch state {
	case NeitherExists:
		// So let's create them both
		utl.PrintYamlColor(obj)
		if !force {
			msg := fmt.Sprintf("%s App/SP pair with above parameters? y/n", utl.Yel("Create"))
			if utl.PromptMsg(msg) != 'y' {
				utl.Die("Operation aborted by user.\n")
			}
		} else {
			fmt.Printf("Creating App/SP pair with above parameters...\n")
		}

		appObj := CreateDirObjectInAzure(Application, obj, z)
		if appObj == nil {
			utl.Die("Error creating App object\n")
		}
		appId := utl.Str(app["appId"])
		spObj := AzureObject{"appId": appId}
		spObj = CreateDirObjectInAzure(ServicePrincipal, spObj, z)
		if spObj == nil {
			utl.Die("Error creating SP object\n")
		}
	case OnlySPExists:
		// So let's update the SP and create the App?
		idSp := utl.Str(sp["id"])
		fmt.Printf("There's an existing SP (%s) named '%s'.\n", idSp, utl.Yel(displayName))
		utl.Die("This condition is not supported. Aborting.\n")
	case OnlyAppExists:
		// So let's update the App and create the SP
		idApp := utl.Str(app["id"])
		fmt.Printf("There's an existing App (%s) named '%s'.\n", idApp, utl.Yel(displayName))
		utl.Die("This condition is not supported. Aborting.\n")
	case BothExist:
		// So let's update them both
		idApp := utl.Str(app["id"])
		idSp := utl.Str(sp["id"])
		UpdateDirObject(force, idApp, obj, Application, z)
		UpdateDirObject(force, idSp, obj, ServicePrincipal, z)
	}
}

// Helper function to check if the object is an App / Service Principal
func IsDirAppSp(obj AzureObject) bool {
	// Check if 'displayName' exists and is a non-empty string
	if utl.Str(obj["displayName"]) == "" {
		return false
	}

	// Check if 'signInAudience' exists and is a non-empty string
	if utl.Str(obj["signInAudience"]) == "" {
		return false
	}

	// If all checks pass, it's a valid App Service Principal
	return true
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
	if statCode != 200 {
		Logf("%s\n", utl.Red2(fmt.Sprintf("HTTP %d: %s", statCode, ApiErrorMsg(resp))))
	}
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
	if statCode != 200 {
		Logf("%s\n", utl.Red2(fmt.Sprintf("HTTP %d: %s", statCode, ApiErrorMsg(resp))))
	}
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
		if statCode != 204 {
			Logf("%s\n", utl.Red2(fmt.Sprintf("HTTP %d: %s", statCode, ApiErrorMsg(resp))))
		}
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
