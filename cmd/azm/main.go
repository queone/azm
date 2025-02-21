package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/queone/azm/pkg/maz"
	"github.com/queone/utl"
)

const (
	program_name    = "azm"
	program_version = "0.1.0"
)

func printUsage() {
	n := utl.Whi2(program_name)
	v := program_version
	X := utl.Red("X")
	usage := fmt.Sprintf("%s v%s\n"+
		"Azure IAM CLI utility - github.com/queone/azm\n"+
		"%s\n"+
		"  %s [options] [arguments]\n"+
		"\n"+
		"  This utility simplifies the management of Azure App and Service Principal (SP) objects.\n"+
		"  Options starting with '-ap' affect App objects, while those starting with '-sp' affect SP\n"+
		"  objects. Other options may impact both.\n"+
		"\n"+
		"%s\n"+
		"  The following options allow you to read and query App and SP objects:\n"+
		"  -%s[j] [FILTER]                   List all %s objects (App or SP) tersely; optional JSON\n"+
		"                                   output; optional match on FILTER string for Id or\n"+
		"                                   DisplayName. If result is a single object, it is printed\n"+
		"                                   in more details.\n"+
		"  -st                              Show count of all App and SP objects in local cache and\n"+
		"                                   Azure tenant\n"+
		"%s\n"+
		"  The following options enable you to create, update, and manage App and SP objects:\n"+
		"  -k                               Generate a YAML skeleton file (appsp.yaml) for App/SP\n"+
		"                                   pair configuration\n"+
		"  -up[f] SPECFILE|NAME             Create or update an App/SP pair from a given configuration\n"+
		"                                   file or with a specified name; use the 'f' option to\n"+
		"                                   suppress the confirmation prompt. Specifile support currently\n"+
		"                                   has limited functionality.\n"+
		"  -rm[f] NAME|ID                   Delete an existing App/SP pair by displayName or App ID\n"+
		"  -rn[f] NAME|ID NEWNAME           Rename an App/SP pair with the given NAME/ID to NEWNAME\n"+
		"  -apas ID SECRET_NAME [EXPIRY]    Add a secret to an App with the given ID; optional expiry\n"+
		"                                   date (YYYY-MM-DD) or in X number of days\n"+
		"  -aprs[f] ID SECRET_ID            Remove a secret from an App with the given ID\n"+
		"  -spas ID SECRET_NAME [EXPIRY]    Add a secret to an SP with the given ID; optional expiry\n"+
		"                                   date (YYYY-MM-DD) or in X number of days\n"+
		"  -sprs[f] ID SECRET_ID            Remove a secret from an SP with the given ID\n"+
		"\n"+
		"%s\n"+
		"  The following options manage your login configuration and cache:\n"+
		"  -id                              Display the currently configured login values\n"+
		"  -id TenantId Username            Set up user credentials for interactive login\n"+
		"  -id TenantId ClientId Secret     Configure ID for automated login\n"+
		"  -tx                              Delete the current configured login values and token\n"+
		"  -apx                             Clear the local App cache\n"+
		"  -spx                             Clear the local SP cache\n"+
		"  -?, -h, --help                   Display this usage page\n"+
		"\n"+
		"%s\n"+
		"  To get started, try experimenting with different options and arguments.\n",
		n, v, utl.Whi2("Usage"), n, utl.Whi2("READ Options"), X, X,
		utl.Whi2("WRITE Options"), utl.Whi2("CONFIG Options"), utl.Whi2("Examples"))
	fmt.Print(usage)
	os.Exit(0)
}

func printUnknownCommandError() {
	helpCmd := utl.Yel(program_name + " -h")
	args := strings.Join(os.Args[1:], " ")
	unknownArgs := fmt.Sprintf("Unknown command or arguments: %s\n"+
		"Please use %s to see available options and usage.\n", utl.Yel(args), helpCmd)
	fmt.Print(unknownArgs)
	os.Exit(1)
}

func main() {
	numberOfArguments := len(os.Args[1:]) // Not including the program itself
	if numberOfArguments < 1 || numberOfArguments > 4 {
		printUsage() // Don't accept less than 1 or more than 4 arguments
	}

	// Set up global config z pointer variable. See Config type in github.com/queone/maz/blob/main/maz.go
	z := maz.NewConfig() // This includes z.ConfDir = "~/.maz", etc

	switch numberOfArguments {
	case 1: // 1 argument
		arg1 := os.Args[1]
		// First, parse those requests that do not need API tokens
		switch arg1 {
		case "-id":
			maz.DumpLoginValues(z)
		case "-h":
			printUsage()
		}
		maz.SetupApiTokens(z) // Next, parse requests that do need API tokens to be ready
		switch arg1 {
		case "-tx":
			maz.RemoveCacheFile("t", z)
			maz.RemoveCacheFile("id", z)
		case "-apx", "-spx":
			t := arg1[1:3]
			maz.RemoveCacheFiles(t, z)
		case "-ap", "-apj", "-sp", "-spj":
			t := arg1[1:3]
			jsonOption := len(arg1) > 3 && arg1[3] == 'j'  // Boolean check for 'j'
			all := maz.GetMatchingObjects(t, "", false, z) // false = don't go to Azure
			if all != nil {                                // This nil check avoids printing 'null'
				if jsonOption {
					utl.PrintJsonColor(all) // Print entire JSON list
				} else {
					for _, i := range all { // Print list tersely
						maz.PrintTersely(t, i)
					}
				}
			}
		case "-st":
			maz.PrintCountStatusAppsAndSps(z)
		case "-k":
			maz.CreateSkeletonFile("appsp")
		default:
			printUnknownCommandError()
		}
	case 2: // 2 arguments
		arg1 := os.Args[1]
		arg2 := os.Args[2]
		maz.SetupApiTokens(z)
		switch arg1 {
		case "-ap", "-apj", "-sp", "-spj":
			t := arg1[1:3]
			jsonOption := len(arg1) > 3 && arg1[3] == 'j' // Boolean check for 'j'
			if utl.ValidUuid(arg2) {
				x := maz.GetObjectFromAzureById(t, arg2, z) // Search by id
				if x != nil {                               // This nil check avoids printing 'null'
					if jsonOption {
						utl.PrintJsonColor(x) // Prints JSON object
					} else {
						maz.PrintObject(t, x, z)
					}
				}
			} else {
				matchingObjects := maz.GetMatchingObjects(t, arg2, false, z)
				if len(matchingObjects) > 1 {
					if jsonOption {
						utl.PrintJsonColor(matchingObjects)
					} else {
						for _, i := range matchingObjects { // Print list tersely
							maz.PrintTersely(t, i)
						}
					}
				} else if len(matchingObjects) == 1 {
					// Single object, let's get the latest from Azure
					obj := matchingObjects[0]
					id := obj["id"].(string)
					x := maz.GetObjectFromAzureById(t, id, z)
					if jsonOption {
						utl.PrintJsonColor(x)
					} else {
						maz.PrintObject(t, x, z)
					}
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
