package maz

import (
	"fmt"

	"github.com/queone/utl"
)

// Prints user object in YAML-like format
func PrintUser(obj map[string]interface{}, z *Config) {
	id := utl.Str(obj["id"])
	if id == "" {
		Logf("PrintUser(): Directory user object has no %s attribute\n", utl.Yel("id"))
		return
	}
	Logf("Printing directory user object with ID %s\n", id)

	// Print the most important attributes first
	fmt.Printf("%s\n", utl.Gra("# Directory user"))
	fmt.Printf("%s: %s\n", utl.Blu("id"), utl.Gre(id))
	fmt.Printf("%s: %s\n", utl.Blu("displayName"), utl.Gre(utl.Str(obj["displayName"])))
	fmt.Printf("%s: %s\n", utl.Blu("userPrincipalName"), utl.Gre(utl.Str(obj["userPrincipalName"])))
	fmt.Printf("%s: %s\n", utl.Blu("onPremisesSamAccountName"), utl.Gre(utl.Str(obj["onPremisesSamAccountName"])))
	fmt.Printf("%s: %s\n", utl.Blu("onPremisesDomainName"), utl.Gre(utl.Str(obj["onPremisesDomainName"])))

	// Print app role assignment members and the specific role assigned
	//apiUrl := ConstMgUrl + "/v1.0/users/" + id + "/appRoleAssignments"
	apiUrl := ConstMgUrl + "/beta/users/" + id + "/appRoleAssignments"
	appRoleAssignments := GetAzureAllPages(apiUrl, z)
	PrintAppRoleAssignmentsOthers(appRoleAssignments, z)

	// Print all groups and roles it is a member of
	apiUrl = ConstMgUrl + "/v1.0/users/" + id + "/transitiveMemberOf"
	resp, statCode, _ := ApiGet(apiUrl, z, nil)
	if statCode != 200 {
		Logf("%s\n", utl.Red2(fmt.Sprintf("HTTP %d: %s", statCode, ApiErrorMsg(resp))))
	}
	transitiveMemberOf := utl.Slice(resp["value"])
	PrintMemberOfs(transitiveMemberOf)
}
