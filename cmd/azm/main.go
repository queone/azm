package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
	"github.com/queone/azm/pkg/maz"
	"github.com/queone/utl"
)

const (
	program_name    = "azm"
	program_version = "0.6.4"
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
		"  %s -d 3819d436-726a-4e40-933e-b0ffeee1d4b9  To show resource Role definition with this\n"+
		"                                               given UUID\n"+
		"  %s -d Reader                                To show all resource Role definitions with\n"+
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
		"  -ar                              Resource role assignment report with resolved attribute names\n"+
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
		"  -up[f] SPECFILE|NAME             Create or update object by given SPECFILE (only for certain\n"+
		"                                   objects); create with given name (again, only some objects); use\n"+
		"                                   the 'f' option to suppress the confirmation prompt\n"+
		"  -rm[f] SPECFILE|ID|NAME          Delete object by given SPECFILE (only for certain objects);\n"+
		"                                   delete by given NAME or ID (by name is only supported on some\n"+
		"                                   objects); use 'f' to suppress confirmation\n"+
		"  -rn[f] NAME|ID NEWNAME           Rename object with given NAME or ID to NEWNAME (not all objects\n"+
		"                                   are supported); use 'f' to suppress confirmation\n"+
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
	args := utl.Yel(program_name + " " + strings.Join(os.Args[1:], " "))
	help := utl.Yel(program_name + " -h")
	utl.Die("Unsupported command: %s. Run %s for more info.\n", args, help)
}

func main() {
	numberOfArguments := len(os.Args[1:]) // Not including the program itself
	if numberOfArguments < 1 || numberOfArguments > 4 {
		printUsage(false) // Don't accept less than 1 or more than 4 arguments
	}

	// Set up global config z pointer variable
	// See Config type in https://github.com/queone/azm/blob/main/pkg/maz/maz.go
	z := maz.NewConfig() // This includes z.ConfDir = "~/.maz", and so on

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
			utl.RemoveFile(filepath.Join(z.ConfDir, z.TokenFile)) // Remove token file
			utl.RemoveFile(filepath.Join(z.ConfDir, z.CredsFile)) // Remove credentials file
		case "-xx":
			// Loop through each mazType in CacheSuffix
			for mazType := range maz.CacheSuffix {
				if err := maz.RemoveCacheFiles(mazType, z); err != nil {
					fmt.Printf("Error removing %s cache files: %v\n", utl.Red(mazType), err)
				}
			}
		case "-ax", "-dx", "-sx", "-mx", "-ux", "-gx", "-apx", "-spx", "-drx", "-dax":
			mazType := arg1[1 : len(arg1)-1]
			maz.RemoveCacheFiles(mazType, z)
		case "-d", "-a", "-s", "-m", "-u", "-g", "-ap", "-sp", "-dr", "-da",
			"-dj", "-aj", "-sj", "-mj", "-uj", "-gj", "-apj", "-spj", "-drj", "-daj":
			specifier := arg1[1:] // Remove arg1 leading '-'
			maz.PrintMatchingObjects(specifier, "", z)
		case "-kd", "-ka", "-kg", "-kap":
			mazType := arg1[2:]
			maz.CreateSkeletonFile(mazType)
		case "-ar":
			maz.PrintResRoleAssignmentReport(z)
		case "-mt":
			maz.PrintAzureMgmtGroupTree(z)
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
			specifier := arg1[1:] // Remove the leading '-'
			maz.PrintMatchingObjects(specifier, arg2, z)
		case "-rm", "-rmf":
			force := false
			if arg1 == "-rmf" {
				force = true
			}
			if utl.FileUsable(arg2) {
				maz.DeleteObjectBySpecfile(force, arg2, z)
			} else if utl.ValidUuid(arg2) {
				maz.DeleteObjectById(force, arg2, z)
			} else {
				maz.DeleteObjectByName(force, arg2, z)
			}
		case "-up", "-upf":
			force := false
			if arg1 == "-upf" {
				force = true
			}
			if utl.FileUsable(arg2) {
				maz.ApplyObjectBySpecfile(force, arg2, z)
			} else {
				utl.Die("Specfile %s is missing or empty\n", utl.Yel(arg2))
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
			maz.AddAppSpSecret(maz.Application, arg2, arg3, "", z)
		case "-aprs", "-aprsf":
			force := false
			if arg1 == "-aprsf" {
				force = true
			}
			maz.RemoveAppSpSecret(maz.Application, arg2, arg3, force, z)
		case "-spas":
			maz.AddAppSpSecret(maz.ServicePrincipal, arg2, arg3, "", z)
		case "-sprs", "-sprsf":
			force := false
			if arg1 == "-sprsf" {
				force = true
			}
			maz.RemoveAppSpSecret(maz.ServicePrincipal, arg2, arg3, force, z)
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
			maz.AddAppSpSecret(maz.Application, arg2, arg3, arg4, z)
		case "-spas":
			maz.AddAppSpSecret(maz.ServicePrincipal, arg2, arg3, arg4, z)
		default:
			printUnknownCommandError()
		}
	}
	os.Exit(0)
}
