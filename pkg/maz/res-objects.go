package maz

// Consolidate -a/-d/-s/-m into single generic functions:
//     One GetMatchingResourceObjects(mazType...) in favor of below 4:
//       GetMatchingAzureMgmtGroups()
//       GetMatchingAzureSubscriptions()
//       GetMatchingResRoleAssignments()
//       GetMatchingResRoleDefinitions()
//     One CacheResourceObjects(mazType...) in favor of below 4:
//       CacheAzureMgmtGroups()
//       CacheAzureSubscriptions()
//       CacheAzureResRoleAssignments()
//       CacheAzureResRoleDefinitions()

func CacheResourceObjects(mazType string) {

}
