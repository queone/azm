package maz

// Consolidate -a/-d/-s/-m into single generic functions:
//     One GetMatchingResourceObjects(mazType...) in favor of below 4:
//       GetMatchingAzureMgmtGroups()
//       GetMatchingAzureSubscriptions()
//       GetMatchingRbacAssignments()
//       GetMatchingRbacDefinitions()
//     One CacheResourceObjects(mazType...) in favor of below 4:
//       CacheAzureMgmtGroups()
//       CacheAzureSubscriptions()
//       CacheAzureRbacAssignments()
//       CacheAzureRbacDefinitions()

func CacheResourceObjects(mazType string) {

}
