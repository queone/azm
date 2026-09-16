package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/google/uuid"
	"github.com/queone/azm/internal/utl"
	"github.com/queone/azm/pkg/maz"
)

const (
	programName        = "azm"
	programVersion     = "1.3.3"
	programDescription = "Azure IAM CLI utility"
	programURL         = "github.com/queone/azm"
)

// typeCodes lists every object type code and its meaning for the help page.
var typeCodes = []utl.Row{
	{Form: maz.ResRoleDefinition, Meaning: []string{"Resource role definitions"}},
	{Form: maz.ResRoleAssignment, Meaning: []string{"Resource role assignments"}},
	{Form: maz.Subscription, Meaning: []string{"Resource subscriptions"}},
	{Form: maz.ManagementGroup, Meaning: []string{"Resource management groups"}},
	{Form: maz.DirectoryUser, Meaning: []string{"Directory users"}},
	{Form: maz.DirectoryGroup, Meaning: []string{"Directory groups"}},
	{Form: maz.Application, Meaning: []string{"Directory applications"}},
	{Form: maz.ServicePrincipal, Meaning: []string{"Directory service principals"}},
	{Form: maz.DirRoleDefinition, Meaning: []string{"Directory role definitions"}},
	{Form: maz.DirRoleAssignment, Meaning: []string{"Directory role assignments"}},
}

// helpPage returns the help page data. The short page keeps only Usage and Examples
// and points at -h for every option.
func helpPage(short bool) utl.Help {
	row := func(form string, meaning ...string) utl.Row {
		return utl.Row{Form: form, Meaning: meaning}
	}
	const set = "d, a, g, ap, and sp only"
	h := utl.Help{
		Name:        programName,
		Version:     programVersion,
		Description: programDescription,
		URL:         programURL,
		Sections: []utl.Section{
			{Heading: "Usage", Rows: []utl.Row{
				row("azm OPTION [ARGUMENT ...]"),
				row("azm UUID"),
			}},
			{Heading: "Options", Rows: []utl.Row{
				row("UUID", "Show every Azure object linked to the given UUID"),
				row("-lc UUID", "Show every cached object linked to the given UUID"),
				row("-X[j] [FILTER]", "List X objects tersely; FILTER matches the id, name, and other",
					"attributes; a single match is fetched from Azure in full"),
				row("-vs SPECFILE", "Compare a specfile to Azure ("+set+")"),
				row("-ar", "Resource role assignment report with resolved attribute names"),
				row("-apr[c] [DAYS]", "Password expiry report for apps and SPs; c for CSV; limit to DAYS"),
				row("-mt", "Print the management group and subscription tree"),
				row("-pags", "List every Entra ID privileged access group"),
				row("-st", "Count the objects in the local cache and in the Azure tenant"),
				row("-Xk [NAME]", "Write a YAML skeleton specfile ("+set+")"),
				row("-up[f] SPECFILE", "Create or update the object in a specfile ("+set+")"),
				row("-upg NAME [DESC] [ASSIGN]", "Create a group; ASSIGN true makes it role-assignable, needs DESC,",
					"and requires the Privileged Role Administrator role"),
				row("-upap NAME, -upsp NAME", "Create an app and SP pair with the given name"),
				row("-rm[f] SPECFILE", "Delete the object in a specfile ("+set+")"),
				row("-rm[f] ID|NAME", "Delete an object by id or name; assignments by id only"),
				row("-rnX[f] NAME|ID NEWNAME", "Rename an object ("+set+")"),
				row("-apas ID NAME [EXPIRY]", "Add a secret to an app; EXPIRY is YYYY-MM-DD or a day count"),
				row("-aprs[f] ID SECRET_ID", "Remove a secret from an app"),
				row("-spas ID NAME [EXPIRY]", "Add a secret to an SP; EXPIRY is YYYY-MM-DD or a day count"),
				row("-sprs[f] ID SECRET_ID", "Remove a secret from an SP"),
				row("-id", "Print the configured login values"),
				row("-id TENANT_ID USERNAME", "Configure interactive user login"),
				row("-id TENANT_ID CLIENT_ID SECRET", "Configure automated client login"),
				row("-tx", "Delete the token cache and the configured login values"),
				row("-xx", "Delete the whole local object cache"),
				row("-Xx", "Delete the local cache of X objects"),
				row("-tmg", "Print the current Microsoft Graph access token"),
				row("-taz", "Print the current Azure Resource Manager access token"),
				row("-td TOKEN", "Decode a JWT token string"),
				row("-uuid", "Generate a random UUID"),
				row("-sfn SPECFILE|ID", "Suggest a specfile name from a specfile or an object id"),
			}, Note: []string{
				"Append j to a list option for JSON output. Append f to a write option to skip",
				"the confirmation prompt. Set MAZLOG=1 for extended logging.",
			}},
			{Heading: "Types", Rows: typeCodes, Note: []string{
				"Replace X in an option with one of these codes.",
			}},
			{Heading: "Examples", Rows: []utl.Row{
				row("azm -id", "Print the configured login values"),
				row("azm -ap", "List every app in the tenant"),
				row("azm -d 3819d436-726a-4e40-933e-b0ffeee1d4b9", "Show the role definition with this id"),
				row("azm -d Reader", "Show every role with Reader in its name"),
				row("azm -g MyGroup", "Show every group matching MyGroup"),
				row("azm -s", "List every subscription in the tenant"),
			}},
		},
	}
	if short {
		h = h.Select("Usage", "Examples")
		h.Sections[1].Note = []string{"Run azm -h for every option."}
	}
	return h
}

// printHelp prints the full or short help page on stdout.
func printHelp(short bool) {
	fmt.Print(utl.RenderHelp(helpPage(short), utl.ColorEnabled()))
}

func printUnknownCommandError() {
	args := utl.Yel(programName + " " + strings.Join(os.Args[1:], " "))
	help := utl.Yel(programName + " -h")
	utl.Die("Unsupported command: %s. Run %s for more info.\n", args, help)
}

// versionLine returns the exact text printed by -v and --version.
func versionLine() string {
	return programName + " v" + programVersion
}

func main() {
	if len(os.Args) == 2 {
		switch os.Args[1] {
		case "-v", "--version":
			fmt.Println(versionLine())
			return
		case "-h", "-?", "--help":
			printHelp(false)
			return
		}
	}
	maz.PrintRuntimeInfo()
	numberOfArguments := len(os.Args[1:]) // Exclude the program itself
	if numberOfArguments < 1 || numberOfArguments > 4 {
		// Don't accept less than 1, or more than 4 arguments
		printHelp(true)
		return
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
