package main

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/queone/maz"
	"github.com/queone/utl"
	"golang.org/x/exp/rand"
)

// Generates dummy JSON groups and save to file for testing.
func GenerateDummyGroupsJson(count int, filePath string) ([]interface{}, string) {
	rand.Seed(uint64(time.Now().UnixNano())) // Seed random number generator
	groups := make([]interface{}, count)
	for i := 0; i < count; i++ {
		displayName := fmt.Sprintf("group-%05d", i+1)
		description := fmt.Sprintf("group %s", utl.GenerateRandomString(20))
		isAssignableToRole := rand.Intn(2) == 1

		// Create a new JSON object as a map
		group := map[string]interface{}{
			"id":                 uuid.New().String(), // Unique ID
			"displayName":        displayName,         // Unique display name
			"description":        description,         // Description with random text
			"isAssignableToRole": isAssignableToRole,  // Randomly assign IsAssignableToRole
			"mailEnabled":        false,
			"mailNickname":       "NotSet",
			"securityEnabled":    true,
		}

		groups[i] = group
	}

	commatized := utl.Int2StrWithCommas(count)
	if len(groups) < 1 {
		utl.Die("==> Failed to generate" + commatized + " groups.\n")
	}

	utl.SaveFileJson(groups, filePath, false) // false = without compression
	fmt.Printf("==> Generated %s interface{} JSON groups and saved them to cache file %s\n",
		utl.Blu(commatized), utl.Blu(filePath))

	// Return the generated groups and one random group's UUID
	randomIndex := rand.Intn(count)
	randomGroup := groups[randomIndex].(map[string]interface{})
	return groups, randomGroup["id"].(string)
}

// Generates dummy typed groups for testing.
func GenerateDummyGroupsTyped(count int, filePath string) (maz.DirGroupList, string) {
	rand.Seed(uint64(time.Now().UnixNano())) // Seed random number generator
	groups := make(maz.DirGroupList, count)
	for i := 0; i < count; i++ {
		id := uuid.New().String()
		displayName := fmt.Sprintf("group-%05d", i+1)
		description := fmt.Sprintf("group %s", utl.GenerateRandomString(20))
		isAssignableToRole := rand.Intn(2) == 1
		group := maz.DirGroup{
			Id:                 id,
			DisplayName:        displayName,
			Description:        description,
			IsAssignableToRole: isAssignableToRole,
		}
		groups[i] = &group
	}

	commatized := utl.Int2StrWithCommas(count)
	if len(groups) < 1 {
		utl.Die("==> Failed to generate" + commatized + " groups.\n")
	}

	if err := maz.SaveDirGroupsToCache(filePath, groups); err != nil {
		utl.Die("==> Failed to save groups to cache.\n")
	}

	fmt.Printf("==> Generated %s maz.DirGroup groups and saved them to cache file %s\n",
		utl.Blu(commatized), utl.Blu(filePath))

	// Return the generated groups and one random group's UUID
	randomIndex := rand.Intn(count)
	return groups, groups[randomIndex].Id
}

// Run directory group benchmarks
// func runBenchmark() {
// 	z := maz.NewConfig() // This includes z.ConfDir = "~/.maz", etc
// 	maz.SetupApiTokens(z)

// 	count := 200000
// 	var start time.Time
// 	randomUuid := ""
// 	// Generate JSON groups then FIND random one
// 	start = time.Now()
// 	cacheFileJson := "groups-cache.json"
// 	_, randomUuid = GenerateDummyGroupsJson(count, cacheFileJson)
// 	x, found := maz.GetGroupFromCache(randomUuid, cacheFileJson)
// 	if found {
// 		fmt.Println("==> Selected below random JSON group:")
// 		maz.PrintGroupJsonTersely(x)

// 		// RENAME
// 		fmt.Println("==> Renaming it ...")
// 		displayName := utl.Str(x["displayName"])
// 		opts := maz.NewOptions().
// 			Set("force", true).
// 			Set("uuid", randomUuid).
// 			Set("newName", displayName+" RENAMED").
// 			Set("groupType", "JSON").
// 			Set("cachePath", cacheFileJson).
// 			Set("object", x)
// 		maz.RenameAzGroup(opts, z)
// 		x, _ := maz.GetGroupFromCache(randomUuid, cacheFileJson)
// 		maz.PrintGroupJsonTersely(x)

// 		// DELETE
// 		fmt.Println("==> Deleting it ...")
// 		maz.DeleteGroupFromCache(randomUuid, cacheFileJson)
// 		groups := maz.GetCachedObjects(cacheFileJson)
// 		fmt.Println("==> New Total count =", len(groups))

// 	} else {
// 		fmt.Println("==> Could not find random group", randomUuid)
// 	}
// 	fmt.Printf("==> JSON groups took %s\n", utl.Cya2(time.Since(start)))

// 	// Generate typed groups then FIND random one
// 	start = time.Now()
// 	cacheFileTyped := "groups-cache.typed"
// 	_, randomUuid = GenerateDummyGroupsTyped(count, cacheFileTyped)
// 	y, err := maz.GetGroupFromCacheNew(randomUuid, cacheFileTyped)
// 	if err == nil {
// 		fmt.Println("==> Selected below random Typed group:")
// 		maz.PrintDirGroupTersely(&y)

// 		// RENAME
// 		fmt.Println("==> Renaming it ...")
// 		displayName := utl.Str(y.DisplayName)
// 		opts := maz.NewOptions().
// 			Set("force", true).
// 			Set("uuid", randomUuid).
// 			Set("newName", displayName+" RENAMED").
// 			Set("groupType", "Typed").
// 			Set("cachePath", cacheFileTyped).
// 			Set("object", y)
// 		maz.RenameAzGroup(opts, z)
// 		y, _ := maz.GetGroupFromCacheNew(randomUuid, cacheFileTyped)
// 		maz.PrintGroupTypedTersely(&y)

// 		// DELETE
// 		fmt.Println("==> Deleting it ...")
// 		err := maz.DeleteGroupFromCacheNew(y.Id, cacheFileTyped)
// 		if err != nil {
// 			utl.Die("Error: " + err.Error() + "\n")
// 		}
// 		groups, _ := maz.GetAllGroupsFromCache(cacheFileTyped)
// 		fmt.Println("==> New Total count =", len(groups))

// 	} else {
// 		fmt.Println("==> Could not find random group", randomUuid)
// 	}
// 	fmt.Printf("==> Typed groups took %s\n", utl.Cya2(time.Since(start)))
// }
