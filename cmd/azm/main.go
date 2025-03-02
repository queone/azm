package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/google/uuid"
	"github.com/queone/azm/pkg/maz"
	"github.com/queone/utl"
)

const (
	program_name    = "azm"
	program_version = "0.1.4"
)

func printUsage(extended bool) {
	n := utl.Whi2(program_name)
	v := program_version
	X := utl.Red("X")
	usageHeader := fmt.Sprintf("%s v%s\n"+
		"Azure IAM CLI manager - github.com/queone/azm\n"+
		"%s\n"+
		"  %s [options] [arguments]\n"+
		"\n"+
		"  This utility simplifies the querying and management of various Azure IAM-related objects.\n"+
		"  In many options %s is a placeholder for a 1-2 character code that specifies the type of\n"+
		"  Azure object to act on. The available codes are:\n\n"+
		"    %s = Resource Role Definitions     %s = Resource Role Assignments\n"+
		"    %s = Resource Subscriptions        %s = Resource Management Groups\n"+
		"    %s = Directory Users               %s = Directory Groups\n"+
		"    %s = Directory Applications        %s = Directory Service Principals\n"+
		"    %s = Directory Role Definitions    %s = Directory Role Assignments\n\n"+
		"  In those options, replace %s with the corresponding code to specify the object type.\n"+
		"\n"+
		"%s\n"+
		"  Try experimenting with different options and arguments, such as:\n"+
		"  %s -id                                      To display the currently configured login values\n"+
		"  %s -ap                                      To list all directory applications registered in\n"+
		"                                               current tenant\n"+
		"  %s -d 3819d436-726a-4e40-933e-b0ffeee1d4b9  To show resource RBAC role definition with this\n"+
		"                                               given UUID\n"+
		"  %s -d Reader                                To show all resource RBAC role definitions with\n"+
		"                                               'Reader' in their names\n"+
		"  %s -g MyGroup                               To show any directory group with the filter\n"+
		"                                               'MyGroup' in its attributes\n"+
		"  %s -s                                       To list all subscriptions in current tenant\n"+
		"  %s -h                                       To display the full list of options\n",
		n, v, utl.Whi2("Usage"), n, X,
		utl.Red("d "), utl.Red("a "), utl.Red("s "), utl.Red("m "), utl.Red("u "),
		utl.Red("g "), utl.Red("ap"), utl.Red("sp"), utl.Red("dr"), utl.Red("da"), X,
		utl.Whi2("Quick Examples"), n, n, n, n, n, n, n)
	usageExtended := fmt.Sprintf("\n%s (allow reading and querying Azure objects)\n"+
		"  UUID                             Show all Azure objects associated with the given UUID\n"+
		"  -%s[j] [FILTER]                   List all %s objects tersely; optional JSON output; optional\n"+
		"                                   match on FILTER string for Id, DisplayName, and other attributes.\n"+
		"                                   If the result is a single object, it is printed in more detail.\n"+
		"  -vs SPECFILE                     Compare YAML specfile to what's in Azure. Only for certain objects.\n"+
		"  -ar                              List all RBAC role assignments with resolved names\n"+
		"  -mt                              List Management Group and subscriptions tree\n"+
		"  -pags                            List all Azure AD Privileged Access Groups\n"+
		"  -st                              Show count of all objects in local cache and Azure tenant\n"+
		"  -tmg                             Display current Microsoft Graph API access token\n"+
		"  -taz                             Display current Azure Resource API access token\n"+
		"  -tc \"TokenString\"                Parse and display the claims contained in the given token\n"+
		"\n"+
		"%s (allow creating and managing Azure objects)\n"+
		"  -k%s                              Generate a YAML skeleton file for object type %s. Only\n"+
		"                                   certain objects are currently supported.\n"+
		"  -up[f] SPECFILE|NAME             Create or update object from given SPECFILE (only for certain\n"+
		"                                   objects); create with given name (again, only some objects); use\n"+
		"                                   the 'f' option to suppress the confirmation prompt\n"+
		"  -rn[f] NAME|ID NEWNAME           Rename object with given NAME or ID to NEWNAME (not all objects\n"+
		"                                   are supported); use 'f' to suppress confirmation\n"+
		"  -rm[f] NAME|ID                   Delete object with given NAME or ID (by name is only supported\n"+
		"                                   on some objects); use 'f' to suppress confirmation\n"+
		"  -apas ID SECRET_NAME [EXPIRY]    Add a secret to an App with the given ID; optional expiry\n"+
		"                                   date (YYYY-MM-DD) or in X number of days\n"+
		"  -aprs[f] ID SECRET_ID            Remove a secret from an App with the given ID\n"+
		"  -spas ID SECRET_NAME [EXPIRY]    Add a secret to an SP with the given ID; optional expiry\n"+
		"                                   date (YYYY-MM-DD) or in X number of days\n"+
		"  -sprs[f] ID SECRET_ID            Remove a secret from an SP with the given ID\n"+
		"\n"+
		"%s\n"+
		"  -id                              Display the currently configured login values\n"+
		"  -id TenantId Username            Set up user credentials for interactive login\n"+
		"  -id TenantId ClientId Secret     Configure ID for automated login\n"+
		"  -tx                              Delete the current configured login values and token\n"+
		"  -xx                              Delete ALL local file cache\n"+
		"  -%sx                              Delete %s object local file cache\n"+
		"  -uuid                            Generate a random UUID\n"+
		"  -?, -h, --help                   Display the full list of options\n",
		utl.Whi2("Read Options"), X, X, utl.Whi2("Write Options"), X, X, utl.Whi2("Other Options"), X, X)
	fmt.Print(usageHeader)
	if extended {
		fmt.Print(usageExtended)
	}
	os.Exit(0)
}

func printUnknownCommandError() {
	helpCmd := utl.Yel(program_name + " -h")
	args := strings.Join(os.Args[1:], " ")
	unknownArgs := fmt.Sprintf("Unknown command or arguments: %s\n"+
		"Run %s to see extended usage.\n", utl.Yel(args), helpCmd)
	fmt.Print(unknownArgs)
	os.Exit(1)
}

func main() {
	numberOfArguments := len(os.Args[1:]) // Not including the program itself
	if numberOfArguments < 1 || numberOfArguments > 4 {
		printUsage(false) // Don't accept less than 1 or more than 4 arguments
	}

	// Set up global config z pointer variable
	// See Config type in https://github.com/queone/azm/blob/main/pkg/maz/maz.go
	z := maz.NewConfig() // This includes z.ConfDir = "~/.maz", etc

	switch numberOfArguments {
	case 1: // 1 argument
		arg1 := os.Args[1]
		// First, parse those requests that do not need API tokens
		switch arg1 {
		case "-id":
			maz.DumpLoginValues(z)
		case "-?", "-h", "--help":
			printUsage(true)
		case "-uuid":
			utl.Die("%s\n", uuid.New().String())
		}
		maz.SetupApiTokens(z) // Next, parse requests that do need API tokens
		switch arg1 {
		case "-tx":
			maz.RemoveCacheFile("t", z)
			maz.RemoveCacheFile("id", z)
		case "-xx":
			// Loop through each type in CacheSuffix.
			for t := range maz.CacheSuffix {
				if err := maz.RemoveCacheFiles(t, z); err != nil {
					fmt.Printf("Error removing cache files for type '%s': %v\n", t, err)
				}
			}

		// Migrating from RemoveCacheFile() ==> to RemoveCacheFiles()
		case "-dx", "-ax", "-mx":
			t := arg1[1:] // Single out the object type
			maz.RemoveCacheFile(t, z)
		case "-sx", "-ux", "-gx", "-apx", "-spx", "-drx", "-dax":
			t := arg1[1 : len(arg1)-1]
			maz.RemoveCacheFiles(t, z)

		case "-d", "-a", "-s", "-m", "-u", "-g", "-ap", "-sp", "-dr", "-da",
			"-dj", "-aj", "-sj", "-mj", "-uj", "-gj", "-apj", "-spj", "-drj", "-daj":
			t := arg1[1:]                         // Remove the leading '-'
			printJson := arg1[len(arg1)-1] == 'j' // If last char is 'j' JSON output is required
			if printJson {
				t = t[:len(t)-1] // Remove the 'j' from t
			}
			allObjects := maz.GetMatchingObjects(t, "", false, z) // false = get from cache, not Azure
			if printJson {
				utl.PrintJsonColor(allObjects) // Print entire set in JSON
			} else {
				for _, i := range allObjects { // Print entire set tersely
					maz.PrintTersely(t, i)
				}
			}
		case "-kd", "-ka", "-kg", "-kap":
			t := arg1[2:]
			maz.CreateSkeletonFile(t)
		case "-ar":
			maz.PrintRoleAssignmentReport(z)
		case "-mt":
			maz.PrintMgTree(z)
		case "-pags":
			maz.PrintPags(z)
		case "-st":
			maz.PrintCountStatus(z)
		case "-tmg":
			fmt.Println(z.MgToken)
		case "-taz":
			fmt.Println(z.AzToken)
		default:
			if utl.ValidUuid(arg1) {
				maz.PrintObjectById(arg1, z)
			} else {
				printUnknownCommandError()
			}
		}
	case 2: // 2 arguments
		arg1 := os.Args[1]
		arg2 := os.Args[2]
		maz.SetupApiTokens(z)
		switch arg1 {
		case "-tc":
			maz.DecodeJwtToken(arg2)

		case "-d", "-a", "-s", "-m", "-u", "-g", "-ap", "-sp", "-dr", "-da",
			"-dj", "-aj", "-sj", "-mj", "-uj", "-gj", "-apj", "-spj", "-drj", "-daj":
			t := arg1[1:]                         // Remove the leading '-'
			printJson := arg1[len(arg1)-1] == 'j' // If last char is 'j' JSON output is required
			if printJson {
				t = t[:len(t)-1] // Remove the 'j' from t
			}
			matchingObjects := maz.GetMatchingObjects(t, arg2, false, z) // false = get from cache, not Azure
			if len(matchingObjects) > 1 {
				if printJson {
					utl.PrintJsonColor(matchingObjects) // Print macthing list in JSON format
				} else {
					for _, i := range matchingObjects {
						maz.PrintTersely(t, i) // Print list tersely
					}
				}
			} else if len(matchingObjects) == 1 {
				singleObj := matchingObjects[0]
				isFromCache := !utl.Bool(singleObj["maz_from_azure"])
				if isFromCache {
					// If object is from cache, then get the full version from Azure
					id := singleObj["id"].(string)
					if t == "s" {
						// For subscriptions, use 'subscriptionId' (UUID) instead of the fully-qualified 'id'
						id = singleObj["subscriptionId"].(string)
					}
					singleObj = maz.GetAzureObjectById(t, id, z)
				}
				if printJson {
					utl.PrintJsonColor(singleObj) // Print in JSON format
				} else {
					maz.PrintObject(t, singleObj, z) // Print in regular format
				}
			}
		case "-rm", "-rmf":
			force := false
			if arg1 == "-rmf" {
				force = true
			}
			maz.DeleteAppSpByIdentifier(force, arg2, z)
		case "-up", "-upf":
			force := false
			if arg1 == "-upf" {
				force = true
			}
			if utl.FileUsable(arg2) {
				maz.UpsertAppSpFromFile(force, arg2, z) // Create/update from arg2 as specfile
			} else {
				maz.CreateAppSpByName(force, arg2, z)
			}
		case "-vs":
			maz.CompareSpecfileToAzure(arg2, z)
		default:
			printUnknownCommandError()
		}
	case 3: // 3 arguments
		arg1 := os.Args[1]
		arg2 := os.Args[2]
		arg3 := os.Args[3]
		switch arg1 {
		case "-id":
			z.TenantId = arg2
			z.Username = arg3
			maz.SetupInterativeLogin(z)
		}
		maz.SetupApiTokens(z) // The remaining 3-arg requests do need API access
		switch arg1 {
		case "-rn", "-rnf":
			force := false
			if arg1 == "-rnf" {
				force = true
			}
			maz.RenameAppSp(force, arg2, arg3, z)
		case "-apas":
			maz.AddAppSpSecret("ap", arg2, arg3, "", z)
		case "-aprs", "-aprsf":
			force := false
			if arg1 == "-aprsf" {
				force = true
			}
			maz.RemoveAppSpSecret("ap", arg2, arg3, force, z)
		case "-spas":
			maz.AddAppSpSecret("sp", arg2, arg3, "", z)
		case "-sprs", "-sprsf":
			force := false
			if arg1 == "-sprsf" {
				force = true
			}
			maz.RemoveAppSpSecret("sp", arg2, arg3, force, z)
		default:
			printUnknownCommandError()
		}
	case 4: // 4 arguments
		arg1 := os.Args[1]
		arg2 := os.Args[2]
		arg3 := os.Args[3]
		arg4 := os.Args[4]
		switch arg1 {
		case "-id":
			z.TenantId = arg2
			z.ClientId = arg3
			z.ClientSecret = arg4
			maz.SetupAutomatedLogin(z)
		}
		maz.SetupApiTokens(z)
		switch arg1 {
		case "-apas":
			maz.AddAppSpSecret("ap", arg2, arg3, arg4, z)
		case "-spas":
			maz.AddAppSpSecret("sp", arg2, arg3, arg4, z)
		default:
			printUnknownCommandError()
		}
	}
	os.Exit(0)
}
