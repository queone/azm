## Releases

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

---

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

---

### TODO
- Ensure all pkg/maz functions are properly commented and reflected at https://pkg.go.dev/github.com/queone/azm/pkg/maz
- Move away from using `interface{}` and `map[string]interface{}` JSON objects in favor of dedicated object types
