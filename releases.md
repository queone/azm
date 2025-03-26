## Releases

### v0.7.1
Release Date: 2025-mar-26
- Optimized Normalize() cache function, included new BatchDeleteByIds() method for AzureObjectList type
- Previous Cache code was open to lock contention, so we now have lock during in-memory operations
- Cache.Delete() and Cache.Upsert() methods no longer do a Save(); this is now forced on the caller

### v0.7.0
Release Date: 2025-mar-26
- Important FetchDirObjectsDelta() and RefreshLocalCacheWithAzure() function updates:
  1. Delta vs. Full Sync Optimization:
    - Use regular pagination (?$top=999) for initial syncs (faster)
    - Use delta queries (/delta) only for updates
  2. Throttling Retries:
    - Added exponential backoff for HTTP 429 responses
  3. Delta Token Resilience:
    - Fall back to full sync if delta token fails to load/save
  4. Consistent Pagination Handling:
    - Reused retry logic for @odata.nextLink requests
  5. Parallelization uses Worker Pool Pattern:
    - 5 concurrent workers process URLs
    - Buffered channels prevent blocking
  6. Non-Blocking Result Processing:
    - Uses select with default to interleave:
      - Result processing
      - Pagination control
  7. Graceful Shutdown:
    - Proper channel closing
    - Drains remaining results before return
  8. Progress Reporting:
    - Maintains verbose output
    - Updates every 100 items

### v0.6.7
Release Date: 2025-mar-25
- Moved raf to its own prper folder cmd/raf/
- Updated build script to also build and install raf
- Created a raf.py

### v0.6.6
Release Date: 2025-mar-25
- Added stand-alone script (not yet building to exec) cmd/raf.go utility to help create resource role definition specfile names
- Prepping main.go to support future options from azapp and azgrp

### v0.6.5
Release Date: 2025-mar-25
- When printing objects, now back to using attribute names, for example displayName instead of display_name
- Enhanced -k* option with skeleton specfile functions to now take a name for the object, which is also used for the filename
- Brought in misc/group-benchmark.go from azgrp for future benchmarking

### v0.6.4
Release Date: 2025-mar-24
- More bug fixes: Typo, message 'The object was still created' had wrong check
- Added Log() function to help debugging by setting MAZ_LOG environment variable
- Added die() and printf to maz pkg for more readable code
- Starting to remove error returns for many functions, because they have a definitive purpose and can just die() with a message

### v0.6.3
Release Date: 2025-mar-24
- Fixed bug with adding a new dir object

### v0.6.2
Release Date: 2025-mar-24
- Updated -vs option flow to now compare appsp specfiles also

### v0.6.1
Release Date: 2025-mar-24
- Fixed in previous version: Removed float assertion in SpsCountAzure() which caused a runtime panic.

### v0.6.0
Release Date: 2025-mar-24
- Renamed UpsertAppSpFromSpecfile() to UpsertAppSp() and simplified logic now that we have a generic GetObjectFromFile() function
- Same for UpsertGroupFromSpecfile() to UpsertGroup()
- Added IsResRoleDefinition() and IsResRoleAssignment() functions, and also standardized the code for AppSp and groups ones

### v0.5.0
Release Date: 2025-mar-23
- Renamed to GetAzureRbacDefinitionByScopeAndName(), and it doesn't really need to return error
- Renamed a number of functions with 'Rbac' to 'ResRole'
- Converted all previous mazType magic strings to their constant values
- Changed the 'RBAC' reference to 'resource' wherever possible

### v0.4.1
Release Date: 2025-mar-23
- UpsertAzureResRoleDefinition() now pretty prints what's to be added/remove using DiffRoleDefinitionSpecfileVsAzure()
- CreateAzureResRoleAssignment() and DeleteAzureResRoleAssignment() now work as expected

### v0.4.0
Release Date: 2025-mar-22
- Completed role assignments; now all objects follow new model
- Delete old RemoveCacheFile() and cleaned that up a bit
- Deleted old GetCachedObjects() now that we are following a new caching model
- Replaced all AzureObjectList.Add() with normal append() because it's faster and more idiomatic
- As misc/go-slice-benchmarks.go shows, for-loop pointer memory-walk optimization wasn't really optimizing, and clarity actually suffers. We willl still switch from value-based to index-based loop for code simplicity.
- In maz.go, switched from strconv.ParseBool() to utl.Bool() as it suddenly began failing. Bizarre!
- Fixed bug whereby cacheNeedsRefreshing boolean also needed to check if cache.Count() < 1
- Bump github.com/golang-jwt/jwt/v5 from 5.2.1 to 5.2.2 to correct CVE-2025-30204
- Fixed DiffRoleDefinitionSpecfileVsAzure() so that -vs specfile shows proper coloring of what would be updated

### v0.3.2
Release Date: 2025-mar-20
- Optimized all functions that build id:name maps with pointer memory-walk, and also renamed them for consistency
- Where appropriate switched to the standard path.Base() instead of utl.LastElem()
- Also switched from generic utl.LastElem() to utl.LastElemByDot() for a bit more efficiency
- In printing.go PrintTersely() is now fully migrated and using AzureObject for all mazType

### v0.3.1
Release Date: 2025-mar-19
- Moving away from syntactic sugar types like JsonType in api-calls.go because it causes too many issues.
- Function DeleteObjectByName() allows deleting objects by name, but only some objects are supported
- Utility -rm "a name" option works for resource RABC role definitions

### v0.3.0
Release Date: 2025-mar-17
- Major architectural renaming of objects, files, functions and so on. Shifting most references from Azure resource 'role' to 'rbac', e. g. 'resource role definition' is now 'resource RBAC definition', and so on. Another example: Instead of GetResRoleDefinitionById(), it is now GetAzureRbacDefinitionById()
- Major rework of `res-rbac-defs.go`, renaming of functions, hopefully making code more clear
- In res-subs.go, pdated all subscription API calls to use api-version=2024-11-01
- In helper.go, renamed FindAzObjectsById() to FindAzureObjectsById()
- In cmd/azm/main.go, simplified utility by moving code to and calling PrintMatchingObjects()
- In dir-apps.go, in PrintApp(), rename federated_ids to federated_credentials, and also print 'aud' list in last column
- Revamped api-calls.go functions to report errors instead of panicking, also added new GetApiErrorMessage() to prettify API error printing
- Replacing all Object type codes magic strings such as 'd' for RBAC role definitions with constants like RbacDefinitionCode
- Optimized many loops of large lists by memory-walking items with pointers which is more efficient

### v0.2.0
Release Date: 2025-mar-02
- **res-mgmt-groups.go**:
  - Migrated resource management groups -m option and object handling to new Cache type model
  - Updated all mgmt group API calls to use api-version=2023-04-01
  - Renamed PrintMgTree() to PrintAzureMgmtGroupTree()
- **res-subs.go**:
  - Updated all subscription API calls to use api-version=2024-11-01
- **helper.go**:
  - Renamed FindAzObjectsById() to FindAzureObjectsById()
- **cmd/azm/main.go**:
  - Simplified utility by moving code to and calling PrintMatchingObjects()
                      
### v0.1.4
Release Date: 2025-mar-02
- **res-subs.go**: Migrated resource subscriptions -s option and object handling to new Cache type model
- Renamed GetMatchingObjects() to GetMatchingDirObjects() to indicate it's only for Directory, MS Graph objects
- Renamed GetObjects() to GetMatchingObjects() to be the generic object matching and querying function, to operate on **any** MS Graph and Azure ARM object supported by this library

### v0.1.3
Release Date: 2025-mar-01
- Improved `build` script
- Renamed all Api*Debug() functions to Api*Verbose()
- Rewrote CheckApiError(utl.Trace2(1), obj, statusCode, err) function for debugging
- Fixed -xx option by improving RemoveCacheFiles(), which now also does not load cache first
- Fixed issues with directory objects and cache, especially role asisgnments which do not use UUIDs
  
### v0.1.2
Release Date: 2025-feb-28
- Still incomplete and not fully working
- Directory objects are mostly working but still many bugs with cache
- Resource objects are still to be migrated
- cmd/azm:
  - Continuing to migrate old azm options to this new version 
- pkg/maz recent changes:
  - Simplified GetMatchingObjects()
  - Upgraded package dependencies:
    - github.com/queone/utl v1.3.1
  - Dropped ApiErrorCheck() and embeded the error checking directly into ApiCall()
  
### v0.1.1
Release Date: 2025-feb-23
- cmd/azm:
  - Options migrated to shi new azm version: -uuid, -tmg, -taz, -tc, -st
  - Now with a basic default usage message, and an extended more detailed one
- pkg/maz recent changes:
  - Cosmetic adjustment of PrintCountStatus()
  - Renamed ValidToken() to IsValidTokenFormat() to emphasize this is only checking string formating
  - Also update maz.go calls to above
  - DecodeJwtToken() now displays base64 encoded signature instead of a byte array

### v0.1.0
Release Date: 2025-feb-21
- Initial commit for this new combined pkg/maz library and cmd/azm repository for easier maintenance
- Updated `build` script to always 1st compile the pkg/maz, then build the cmd/azm binary
- cmd/azm recent changes:
  - Abandoning the idea of multiple individual utilities like `azapp`, `azgrp`, and so forth
  - Having this single `azm` utility makes for easier maintenance
  - Leveraging AcquireTokenByDeviceCode() support to allow login from within a VM
  - Upgraded package dependencies:
    - <github.com/queone/utl> v1.3.0
  - Updated printUsage() style and details
- pkg/maz recent changes:
  - Upgraded package dependencies:
    - <github.com/AzureAD/microsoft-authentication-library-for-go> v1.4.0
    - <github.com/queone/utl> v1.3.0
  - **api-calls.go**:
    - Fixed PrintApiErrMsg() to handle condition when there's only a single line
  - **files.go**:
    - Fixed SaveFileBinaryList() atomic file replacement part to do retries when it fails
  - **token.go**:
    - Added support for AcquireTokenByDeviceCode() for when calling from within a VM
  - Dropped support for JSON skeleton specfiles. YAML files are much more flexible, and allow comments
  - Now using `AzureObject` and `AzureObjectList` as basic types, replacing old `map[string]interface{}` and `[]interface{}`
  - Now using `Cache` type for more intuitively managing cached objects. It optimizes lookups and deletions of objects in cache.
  - Now using more generic functions that leverage below **t** 2-letter code for processing the respective Azure object type: 

| 2-Letter Code | Cache file suffix    | Code file            | Notes                                 |
|---------------|----------------------|----------------------|---------------------------------------|
| `d`           | `_res-role-defs`     | `res-role-defs.go`   | Resource RBAC role definition objects |
| `a`           | `_res-role-assgns`   | `res-role-assgns.go` | Resource RBAC role assignment objects |
| `s`           | `_res-subs`          | `res-subs.go`        | Resource subscriptions objects        |
| `mg`          | `_res-mgmt-groups`   | `res-mgmt-groups.go` | Resource management groups objects    |
| `u`           | `_dir-users`         | `dir-users.go`       | Directory users objects               |
| `g`           | `_dir-groups`        | `dir-groups.go`      | Directory group objects               |
| `sp`          | `_dir-sps`           | `dir-sps.go`         | Directory service principal objects   |
| `ap`          | `_dir-apps`          | `dir-apps.go`        | Directory application objects         |
| `ad`          | `_dir-roles`         | `dir-roles.go`       | Directory role definition objects     |

  - Switched to generic `FetchDirObjects(t, z)` and away from individual types like `GetAzureGroups(z)`
  - Switched to generic `GetMatchingObjects(t, filter, z)` and away from individual ones like `GetMatchingGroups(filter, z)`
  - Switched to generic `RemoveCacheFiles(t, z)` and away from individual ones like `RemoveAppCacheFile(z)`
  - Switched to generic `ObjectCountAzure(t, z)` and away from individual ones like `GroupCountAzure(z)`
  - Switched to generic `ObjectCountLocal(t, z)` and away from individual ones like `GroupCountLocal(z)`
  - Switched to generic `GetObjectIdMap(t, z)` and away from individual ones like `GetIdMapGroups(z)`
  - Switched to generic `GetObjectFromAzureById(t, id, z)` and away from individual ones like `GetGroupFromAzureById(id, z)`
  - Switched to generic `GetObjectFromAzureByName(t, name, z)` and away from individual ones like `GetGroupFromAzureByName(name, z)`
  - New generic function `DeleteDirObject(t, id, z)` to delete any MS Graph object
  - New generic function `CreateDirObject(t, obj, z)` to create any MS Graph object
  - New generic function `UpdateDirObject(t, id, z)` to update any MS Graph object
  - New generic function `RenameDirObject(t, id, z)` to rename any MS Graph object
  - New `ApiEndpoint` and `MazObjName` map variables to help genericized other functions
  - Fix many a *non-constant format string in call to fmt.Printf* errors
  - Updated formatting in many `utl.Die()` function calls.
  - **helper.go**:
    - Cosmetic changes to functions `GetObjectFromFile()` and `CompareSpecfileToAzure()` to make it easier to read.
  - **mg-groups.go**:  
    - Cosmetic updates to make code more readable.
  - Introduced this new `releases.md` file
  - Incorporated `token_accessor.go` file into `token.go`, for simplicity.
  - Major refactoring to remove usage of  `SaveFileJsonGzip()` and `LoadFileJsonGzip()` functions, in favor of updated `SaveFileJson()` and `LoadFileJson()` functions which now support a `compress` option instead. This involved updates of multiples files.
  - Renamed all functions with 'AzGroup' in their names to now have 'DirGroup'
  - Renamed all files with underscore, to now use a hyphen.
  - **options.go**:
    - New file for new `Options` type with constructors and relevant methods for easier creation and updating of Azure objects.
    - Actually, this now seems a bit cumbersome and may go away soon.
  - **maz.go**:
    - New `Config` type with constructors and relevant methods for easier tracking of global configurations.
    - `Config` struct no longer includes the `AuthorityUrl` field. It now relies on the global `ConstAuthUrl` instead.
    - Created new `Config` type to replace the current `Bundle` type.
    - Updated `SetupCredentials()` to align with the new `Config` structure.
  - **token.go**:
    - Refactored `GetTokenInteractively()` and `GetTokenByCredentials()` to now leverage the global `Config` variables.
  - **api-calls.go**:
    - Added `ApiPatch()` and `ApiPatchDebug()` to support creating and updating MS Graph objects.
  - **helper.go**:
    - Updated `GetObjFromFile()` to use the new `utl.ValidateJson()` and `ValidateYaml()` functions for more effective validations.
    - Renamed `GetObjFromFile()` to `GetObjectFromFile()` for better readability and consistency.
    - Function `GetObjectFromFile()` also nows supports checking for directory group objects as well.
  - **skeleton.go**:
    - Added the ability to create `directory-group.yaml/json` skeletons in `CreateSkeletonFile()`.
  - **mg-groups.go**:
    - Major refactoring for handling directory groups:
    - Now supports creating, updating, and deleting groups.
    - Introduced the new `maz.DirGroup` type for stronger typing, replacing the use of `interface{}` and `map[string]interface{}` JSON objects.
    - Added `LoadCacheDirGroups()` and `SaveCacheDirGroups()` for file caching of `maz.DirGroup` lists.
    - Introduced `UpsertGroupInCache()` to replace the older `AddGroupToCache()` function, providing improved functionality for updating or adding groups to the cache.
  - Updated go.mod to indicate new major version: `module github.com/queone/maz/v2`. (Botched tag v2.0.0-rc1)

### TODO
- Ensure all pkg/maz functions are properly commented so they appear correctly at https://pkg.go.dev/github.com/queone/azm/pkg/maz
- Move away from using `interface{}` and `map[string]interface{}` in function arguments for JSON objects, in favor of dedicated object types
