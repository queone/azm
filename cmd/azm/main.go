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
	program_version = "1.0.0"

	clrPrevLine = "\x1B[1A\x1B[2K\r" // Move up one line, clear it, and return cursor to start
)

func printUsage(extended bool) {
	n := utl.Whi2(program_name)
	v := program_version
	X := utl.Red("X")
	usageHeader := fmt.Sprintf("%s v%s\n"+
		"Azure IAM CLI utility - github.com/queone/azm\n"+
		"%s\n"+
		"  %s [options] [arguments]\n", n, v, utl.Whi2("Usage"), n)
	usageHeader += fmt.Sprintf("\n"+
		"  This tool helps with querying and managing Azure IAM-related objects. Many options use %s\n"+
		"  as a placeholder for a 1–2 letter code indicating the object type. Supported types:\n\n"+
		"    %s = Resource Role Definitions     %s = Resource Role Assignments\n"+
		"    %s = Resource Subscriptions        %s = Resource Management Groups\n"+
		"    %s = Directory Users               %s = Directory Groups\n"+
		"    %s = Directory Applications        %s = Directory Service Principals\n"+
		"    %s = Directory Role Definitions    %s = Directory Role Assignments\n\n"+
		"  Replace %s with the relevant code in supported options.\n"+
		"\n", X,
		utl.Red(fmt.Sprintf("%2s", maz.ResRoleDefinition)), utl.Red(fmt.Sprintf("%2s", maz.ResRoleAssignment)),
		utl.Red(fmt.Sprintf("%2s", maz.Subscription)), utl.Red(fmt.Sprintf("%2s", maz.ManagementGroup)),
		utl.Red(fmt.Sprintf("%2s", maz.DirectoryUser)), utl.Red(fmt.Sprintf("%2s", maz.DirectoryGroup)),
		utl.Red(fmt.Sprintf("%2s", maz.Application)), utl.Red(fmt.Sprintf("%2s", maz.ServicePrincipal)),
		utl.Red(fmt.Sprintf("%2s", maz.DirRoleDefinition)), utl.Red(fmt.Sprintf("%2s", maz.DirRoleAssignment)), X)
	usageHeader += fmt.Sprintf("%s\n"+
		"  Try experimenting with different options and arguments, such as:\n"+
		"\n"+
		"  %s -id                                      To display the currently configured login values\n"+
		"  %s -ap                                      To list all Apps in current tenant\n"+
		"  %s -d 3819d436-726a-4e40-933e-b0ffeee1d4b9  Show resource role definition with this UUID\n"+
		"  %s -d Reader                                Show all roles with 'Reader' in the name\n"+
		"  %s -g MyGroup                               Show any directory group matching 'MyGroup'\n"+
		"  %s -s                                       To list all subscriptions in current tenant\n"+
		"  %s -?                                       To display the full list of options\n",
		utl.Whi2("Quick Examples"), n, n, n, n, n, n, n)

	set1 := fmt.Sprintf("%s, %s, %s, %s, and %s",
		utl.Red(maz.ResRoleDefinition), utl.Red(maz.ResRoleAssignment), utl.Red(maz.DirectoryGroup),
		utl.Red(maz.Application), utl.Red(maz.ServicePrincipal))

	usageExtended := fmt.Sprintf("\n%s\n"+
		"  Use optional [j] for JSON output\n"+
		"\n"+
		"  UUID                             Show all Azure objects linked to the given UUID\n"+
		"  -lc UUID                         Show all cached objects linked to the given UUID\n"+
		"  -%s[j] [FILTER]                   List all %s objects tersely (ID, name, etc.); optional match\n"+
		"                                   on FILTER string for Id, DisplayName, and other attributes. If\n"+
		"                                   the result is a single object, it is fetched directly from Azure\n"+
		"                                   and printed in more detail.\n"+
		"  -vs SPECFILE                     Compare specfile to Azure (%s only)\n"+
		"  -ar                              Resource role assignment report with resolved attribute names\n"+
		"  -apr[c] [DAYS]                   Password expiry report for Apps/SPs; CSV optional; limit by DAYS\n"+
		"  -mt                              List Management Group and subscriptions tree\n"+
		"  -pags                            List all Entra ID Privileged Access Groups\n"+
		"  -st                              Show count of all objects in local cache and Azure tenant\n",
		utl.Whi2("Read Options"), X, X, set1)

	usageExtended += fmt.Sprintf("\n%s\n"+
		"  Use optional [f] to bypass confirmation prompts (e.g., -rmf to skip confirm)\n"+
		"\n"+
		"  -%sk [NAME]                       Generate YAML skeleton (%s only); NAME optional\n",
		utl.Whi2("Write Options"), X, set1)

	usageExtended += fmt.Sprintf("\n%s"+
		"  -up[f] SPECFILE                  Create/update object defined in specfile (%s only)\n",
		clrPrevLine, set1)

	G := utl.Red(maz.DirectoryGroup)
	usageExtended += fmt.Sprintf("\n%s"+
		"  -up%s NAME [DESC] [ASSIGN]        Create a group (ASSIGN sets the isAssignableToRole flag):\n"+
		"                                   DESC: Optional description (required if ASSIGN is used)\n"+
		"                                   ASSIGN: Optional; set to 'true' to make group role-assignable\n"+
		"                                           (requires Privileged Role Administrator role)\n"+
		"                                   Examples: azgrp -up%s my_group1\n"+
		"                                             azgrp -up%s my_role_group \"\" true\n",
		clrPrevLine, G, G, G)

	usageExtended += fmt.Sprintf("\n%s"+
		"  -up%s NAME, -up%s NAME           Create App/SP pair with the given name (defaults for all else)\n",
		clrPrevLine, utl.Red(maz.Application), utl.Red(maz.ServicePrincipal))

	usageExtended += fmt.Sprintf("\n%s"+
		"  -rm[f] SPECFILE                  Delete object defined in specfile (%s only)\n"+
		"  -rm[f] ID|NAME                   Delete object (assignments don't support NAME)\n",
		clrPrevLine, set1)

	usageExtended += fmt.Sprintf("\n%s"+
		"  -rn%s[f] NAME|ID NEWNAME          Rename object (%s only)\n", clrPrevLine, X, set1)

	usageExtended += fmt.Sprintf("\n%s"+
		"  -apas ID NAME [EXPIRY]           Add secret to App ID; optional expiry (YYYY-MM-DD or in X days)\n"+
		"  -aprs[f] ID SECRET_ID            Remove secret from App ID\n"+
		"  -spas ID NAME [EXPIRY]           Add secret to SP ID; optional expiry (YYYY-MM-DD or in X days)\n"+
		"  -sprs[f] ID SECRET_ID            Remove secret from SP ID\n"+
		"\n", clrPrevLine)

	usageExtended += fmt.Sprintf("%s\n"+
		"  -id                              Display the currently configured login values\n"+
		"  -id TenantId Username            Set up user credentials for interactive login\n"+
		"  -id TenantId ClientId Secret     Configure ID for automated login\n"+
		"  -tx                              Delete the current token and other configured login values\n"+
		"  -xx                              Delete ALL local file cache\n"+
		"  -%sx                              Delete %s object local file cache\n"+
		"  -tmg                             Display current Microsoft Graph API access token\n"+
		"  -taz                             Display current Azure Resource API access token\n"+
		"  -td \"TokenString\"                Decode given JWT token string\n"+
		"  -uuid                            Generate a random UUID\n"+
		"  -sfn SPECFILE|ID                 Generate specfile from another specfile or object ID\n"+
		"  -?, -h, --help                   Display the full list of options\n"+
		"  LOGGING NOTE                     Use MAZLOG=1 to see extended logging\n",
		utl.Whi2("Other Options"), X, X)

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
	maz.Logf("Login is enabled\n")
	numberOfArguments := len(os.Args[1:]) // Exclude the program itself
	if numberOfArguments < 1 || numberOfArguments > 4 {
		// Don't accept less than 1, or more than 4 arguments
		printUsage(false) // false = display short usage
	}

	// Set up required global configuration pointer variable
	// For more info see https://github.com/queone/azm/blob/main/pkg/maz/maz_core.go
	z := maz.NewConfig()

	switch numberOfArguments {
	case 1: // 1 argument
		arg1 := os.Args[1]
		// Below cases don't need API access
		switch arg1 {
		case "-id":
			maz.DumpLoginValues(z)
		case "-?", "-h", "--help":
			printUsage(true) // true = display long usage
		case "-uuid":
			utl.Die("%s\n", uuid.New().String())
		case "-tx":
			maz.DeleteCurrentCredentials()
		}
		maz.SetupApiTokens(z) // Remaining cases need API access
		switch arg1 {
		case "-ax", "-dx", "-sx", "-mx", "-ux", "-gx", "-apx", "-spx", "-drx", "-dax", "-xx":
			mazType := arg1[1 : len(arg1)-1]
			maz.PurgeMazObjectCacheFiles(mazType, z)
		case "-d", "-a", "-s", "-m", "-u", "-g", "-ap", "-sp", "-dr", "-da",
			"-dj", "-aj", "-sj", "-mj", "-uj", "-gj", "-apj", "-spj", "-drj", "-daj":
			specifier := arg1[1:] // Remove arg1 leading hyphen
			maz.PrintMatchingObjects(specifier, "", z)
		case "-dk", "-ak", "-gk", "-apk":
			mazType := arg1[1 : len(arg1)-1]
			maz.CreateSkeletonFile(mazType, "")
		case "-ar":
			maz.PrintResRoleAssignmentReport(z)
		case "-apr", "-aprc":
			csvMode := arg1 == "-aprc" // flag ending in 'c' triggers CSV mode
			maz.PrintPasswordExpiryReport(csvMode, "", z)
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
		switch arg1 {
		case "-td":
			maz.DecodeAndValidateToken(arg2)
		}
		maz.SetupApiTokens(z) // Remaining cases need API access
		switch arg1 {
		case "-lc":
			maz.PrintCachedObjectsWithId(arg2, z)
		case "-kd", "-ka", "-kg", "-kap":
			mazType := arg1[2:]
			maz.CreateSkeletonFile(mazType, arg2)
		case "-d", "-a", "-s", "-m", "-u", "-g", "-ap", "-sp", "-dr", "-da",
			"-dj", "-aj", "-sj", "-mj", "-uj", "-gj", "-apj", "-spj", "-drj", "-daj":
			specifier := arg1[1:] // Remove the leading '-'
			maz.PrintMatchingObjects(specifier, arg2, z)
		case "-sfn":
			maz.GenerateAndPrintSpecfileName(arg2, z)
		case "-rm", "-rmf":
			force := arg1 == "-rmf" // flag ending in 'f' triggers force mode
			if utl.FileUsable(arg2) {
				maz.DeleteObjectBySpecfile(force, arg2, z)
			} else if utl.ValidUuid(arg2) {
				maz.DeleteObjectById(force, arg2, z)
			} else {
				maz.DeleteObjectByName(force, arg2, z)
			}
		case "-up", "-upf":
			force := arg1 == "-upf"
			maz.ApplyObjectBySpecfile(force, arg2, z)
		case "-upap", "-upsp":
			// Create AppSp pair with given name (no prompt, safe to force)
			maz.CreateAppSpByName(true, arg2, z)
		case "-upg":
			// Create group with given name (no prompt), not assignable to role; description = name
			maz.CreateDirGroupFromArgs(true, false, arg2, arg2, z)
		case "-vs":
			maz.CompareSpecfileToAzure(arg2, z)
		case "-apr", "-aprc":
			csvMode := arg1 == "-aprc" // flag ending in 'c' triggers CSV mode
			maz.PrintPasswordExpiryReport(csvMode, arg2, z)
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
			maz.ConfigureCredsFileForInterativeLogin(z)
		}
		maz.SetupApiTokens(z) // Remaining cases need API access
		switch arg1 {
		case "-rnd", "-rng", "-rnap", "-rnsp", "-rndr",
			"-rndf", "-rngf", "-rnapf", "-rnspf", "-rndrf":
			flagBody := arg1[3:] // e.g. "gf"
			force := strings.HasSuffix(flagBody, "f")
			mazType := strings.TrimSuffix(flagBody, "f")
			maz.RenameAzureObject(force, mazType, arg2, arg3, z)
		case "-upg":
			force := true // safe, no prompt needed
			isAssignableToRole := false
			name := arg2
			description := arg3
			maz.CreateDirGroupFromArgs(force, isAssignableToRole, name, description, z)
		case "-apas":
			maz.AddAppSpSecret(maz.Application, arg2, arg3, "", z)
		case "-aprs", "-aprsf":
			force := arg1 == "-aprsf" // flag ending in 'f' triggers force mode
			maz.RemoveAppSpSecret(maz.Application, arg2, arg3, force, z)
		case "-spas":
			maz.AddAppSpSecret(maz.ServicePrincipal, arg2, arg3, "", z)
		case "-sprs", "-sprsf":
			force := arg1 == "-sprsf"
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
			maz.ConfigureCredsFileForAutomatedLogin(z)
		}
		maz.SetupApiTokens(z) // Remaining cases need API access
		switch arg1 {
		case "-upg":
			force := true // safe, no prompt needed
			isAssignableToRole := utl.Bool(arg4)
			name := arg2
			description := arg3
			maz.CreateDirGroupFromArgs(force, isAssignableToRole, name, description, z)
		case "-apas":
			maz.AddAppSpSecret(maz.Application, arg2, arg3, arg4, z)
		case "-spas":
			maz.AddAppSpSecret(maz.ServicePrincipal, arg2, arg3, arg4, z)
		default:
			printUnknownCommandError()
		}
	}
}
