package maz

import (
	"fmt"
	"strings"

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
	if x["keyCredentials"] != nil {
		PrintCertificateList(x["keyCredentials"].([]interface{}))
	}

	// Print secret list & expiry details, not actual secretText (which cannot be retrieve anyway)
	if x["passwordCredentials"] != nil {
		PrintSecretList(x["passwordCredentials"].([]interface{}))
	}

	// Print federated credentials
	apiUrl := ConstMgUrl + "/v1.0/applications/" + id + "/federatedIdentityCredentials"
	r, statusCode, _ := ApiGet(apiUrl, z, nil)
	if statusCode == 200 && r != nil && r["value"] != nil {
		fedCreds := r["value"].([]interface{})
		if len(fedCreds) > 0 {
			fmt.Println(utl.Blu("federated_credentials") + ":")
			for _, item := range fedCreds {
				cred, ok := item.(map[string]interface{})
				if !ok {
					fmt.Printf("  %s\n", utl.Gre("(unable to read cred)"))
					continue
				}
				iId := utl.Gre(utl.Str(cred["id"]))
				name := utl.Gre(utl.Str(cred["name"]))
				sub := utl.Gre(utl.Str(cred["subject"]))
				iss := utl.Gre(utl.Str(cred["issuer"]))
				var audiences []string
				if audList, ok := cred["audiences"].([]interface{}); ok {
					for _, audience := range audList {
						audiences = append(audiences, utl.Str(audience)) // Convert and append to the slice
					}
				}
				aud := utl.Gre(strings.Join(audiences, ", "))
				// TODO: Fix the coloring padding
				//fmt.Printf("  %-36s  %-40s  %-40s  %-40s  %s\n", iId, name, sub, iss, aud)
				fmt.Printf("  %-36s  %-20s  %s  %s  %s\n", iId, name, sub, iss, aud)
			}
		}
	}

	// Print any owners
	apiUrl = ConstMgUrl + "/beta/applications/" + id + "/owners"
	r, statusCode, _ = ApiGet(apiUrl, z, nil)
	if statusCode == 200 && r != nil && r["value"] != nil {
		PrintOwners(r["value"].([]interface{}))
	}

	// Print any oAuth2 permission scopes
	if x["api"] != nil {
		api := x["api"].(map[string]interface{})
		oauth2PermissionScopes := api["oauth2PermissionScopes"].([]interface{})
		scopeValueMap := make(map[string]string)
		if len(oauth2PermissionScopes) > 0 {
			fmt.Printf("%s:\n", utl.Blu("oauth2_permission_scopes"))
			for _, i := range oauth2PermissionScopes {
				a := i.(map[string]interface{})
				scopeId := utl.Str(a["id"])
				enabledStat := "Disabled"
				if utl.Str(a["isEnabled"]) == "true" {
					enabledStat = "Enabled"
				}
				apiName := appDisplayName
				scopeType := "Delegated"
				scopeValue := utl.Str(a["value"])
				scopeValueMap[scopeId] = scopeValue // Keep building scopeValueMap (to be used for preAuthApp below)
				fmt.Printf("  %s%s  %s%s  %s%s  %s%s  %s\n",
					utl.Gre(scopeId), utl.PadSpaces(38, len(scopeId)),
					utl.Gre(enabledStat), utl.PadSpaces(10, len(enabledStat)),
					utl.Gre(apiName), utl.PadSpaces(50, len(apiName)),
					utl.Gre(scopeType), utl.PadSpaces(12, len(scopeType)),
					utl.Gre(scopeValue))
			}
		}
		// Also print any pre authorized applications
		preAuthorizedApplications := api["preAuthorizedApplications"].([]interface{})
		if len(preAuthorizedApplications) > 0 {
			fmt.Printf("%s:\n", utl.Blu("  pre_authorized_applications"))
			for _, i := range preAuthorizedApplications {
				a := i.(map[string]interface{})
				clientId := utl.Str(a["appId"])
				permissionIds := a["permissionIds"].([]interface{})
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
	if x["requiredResourceAccess"] != nil && len(x["requiredResourceAccess"].([]interface{})) > 0 {
		fmt.Printf("%s:\n", utl.Blu("api_permissions_assigned"))
		APIs := x["requiredResourceAccess"].([]interface{}) // Assert to JSON array
		for _, a := range APIs {
			api := a.(map[string]interface{})
			// Getting this API's name and permission value such as Directory.Read.All is a 2-step process:
			// 1) Get all the roles for given API and put their id/value pairs in a map, then
			// 2) Use that map to enumerate and print them

			// Let's drill down into the permissions for this API
			if api["resourceAppId"] == nil {
				fmt.Printf("  %-50s %s\n", "Unknown API", "Missing resourceAppId")
				continue // Skip this API, move on to next one
			}
			resAppId := utl.Str(api["resourceAppId"])

			// Get this API's SP object with all relevant attributes
			params := map[string]string{"$filter": "appId eq '" + resAppId + "'"}
			apiUrl := ConstMgUrl + "/beta/servicePrincipals"
			r, _, _ := ApiGet(apiUrl, z, params)

			// Result is a list because this could be a multi-tenant app, having multiple SPs
			if r["value"] == nil {
				fmt.Printf("  %-50s %s\n", resAppId, "Unable to get Resource App object. Skipping this API.")
				continue
			}

			// TODO: Handle multiple SPs

			SPs := r["value"].([]interface{})
			if len(SPs) > 1 {
				utl.Die("  %-50s %s\n", resAppId, "Error. Multiple SPs for this AppId. Aborting.")
			} else if len(SPs) < 1 {
				fmt.Printf("  %-50s %s\n", resAppId, "Unable to get Resource App object. Skipping this API.")
				continue
			}

			// Currently only handling the expected single-tenant entry
			sp := SPs[0].(map[string]interface{})

			// 1. Put all API role id:name pairs into roleMap list
			roleMap := make(map[string]string)
			if sp["appRoles"] != nil { // These are for Application types
				for _, i := range sp["appRoles"].([]interface{}) { // Iterate through all roles
					role := i.(map[string]interface{})
					//utl.PrintJsonColor(role) // DEBUG
					if role["id"] != nil && role["value"] != nil {
						roleMap[utl.Str(role["id"])] = utl.Str(role["value"]) // Add entry to map
					}
				}
			}
			if sp["publishedPermissionScopes"] != nil { // These are for Delegated types
				for _, i := range sp["publishedPermissionScopes"].([]interface{}) {
					role := i.(map[string]interface{})
					//utl.PrintJsonColor(role) // DEBUG
					if role["id"] != nil && role["value"] != nil {
						roleMap[utl.Str(role["id"])] = utl.Str(role["value"])
					}
				}
			}
			if len(roleMap) < 1 {
				fmt.Printf("  %-50s %s\n", resAppId, "Error getting list of appRoles.")
				continue
			}

			// 2. Parse this app permissions, and use roleMap to display permission value
			if api["resourceAccess"] != nil && len(api["resourceAccess"].([]interface{})) > 0 {
				Perms := api["resourceAccess"].([]interface{})
				//utl.PrintJsonColor(Perms)             // DEBUG
				apiName := utl.Str(sp["displayName"]) // This API's name
				for _, i := range Perms {             // Iterate through perms
					perm := i.(map[string]interface{})
					pid := utl.Str(perm["id"]) // JSON string
					var pType string = "?"
					if utl.Str(perm["type"]) == "Role" {
						pType = "Application"
					} else {
						pType = "Delegated"
					}
					fmt.Printf("  %s%s  %s%s  %s\n", utl.Gre(apiName), utl.PadSpaces(40, len(apiName)),
						utl.Gre(pType), utl.PadSpaces(14, len(pType)), utl.Gre(roleMap[pid]))
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
