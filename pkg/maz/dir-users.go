package maz

import (
	"fmt"

	"github.com/queone/utl"
)

// Prints user object in YAML-like format
func PrintUser(x map[string]interface{}, z *Config) {
	id := utl.Str(x["id"])
	if id == "" {
		return
	}

	// Print the most important attributes first
	fmt.Printf("%s\n", utl.Gra("# Directory User"))
	fmt.Printf("%s: %s\n", utl.Blu("object_id"), utl.Gre(id))
	fmt.Printf("%s: %s\n", utl.Blu("display_name"), utl.Gre(utl.Str(x["displayName"])))
	fmt.Printf("%s: %s\n", utl.Blu("user_principal_name"), utl.Gre(utl.Str(x["userPrincipalName"])))
	fmt.Printf("%s: %s\n", utl.Blu("on_premises_sam_account_name"), utl.Gre(utl.Str(x["onPremisesSamAccountName"])))
	fmt.Printf("%s: %s\n", utl.Blu("on_premises_domain_ame"), utl.Gre(utl.Str(x["onPremisesDomainName"])))

	// Print app role assignment members and the specific role assigned
	//apiUrl := ConstMgUrl + "/v1.0/users/" + id + "/appRoleAssignments"
	apiUrl := ConstMgUrl + "/beta/users/" + id + "/appRoleAssignments"
	appRoleAssignments := GetAzAllPages(apiUrl, z)
	PrintAppRoleAssignmentsOthers(appRoleAssignments, z)

	// Print all groups and roles it is a member of
	apiUrl = ConstMgUrl + "/v1.0/users/" + id + "/transitiveMemberOf"
	r, statusCode, _ := ApiGet(apiUrl, z, nil)
	if statusCode == 200 && r != nil && r["value"] != nil {
		memberOf := r["value"].([]interface{})
		PrintMemberOfs("g", memberOf)
	}
}
