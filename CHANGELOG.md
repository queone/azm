# Changelog

| Version | Summary |
|---------|---------|
| Unreleased | |
| 1.2.0 | AC2 XDG config dirs, drop utl dep, module path, retire raf, backfill changelog |
| 1.1.0 | AC1 adopt govna canon v0.53.0, build.sh, -v/--version; go fix modernizations |
| 1.0.6 | Work around Graph v1.0 not returning group members that are service principals by using the beta endpoint |
| 1.0.5 | Fix the -id option and normalize the credentials file path it prints |
| 1.0.4 | Resume a previously interrupted partial delta fetch on every directory object lookup |
| 1.0.3 | Faster cache Normalize without the locking bottleneck; save in-progress delta sets and resume interrupted delta fetches |
| 1.0.2 | Rename the binary save and load helpers to their map variants; clearer cache Normalize logging; delta fetch logic moves to its own file with its sequential constraints documented |
| 1.0.1 | Race-detector test builds; deduplicate API results by id and stop delta workers from fetching the same URLs twice |
| 1.0.0 | Finish the maz and azm consolidation and refactoring; fetch functions log through MAZLOG instead of verbose output |
| 0.9.2 | Password expiry report for apps and service principals through the new -apr and -aprc options; explain parallel batch fetching |
| 0.9.1 | Parallel cache and Azure ID lookups across object types; MAZLOG timings print with thousands separators |
| 0.9.0 | Stop caching objects with empty or malformed IDs and handle non-GUID Entra role assignment IDs; new -upg, -upap, and -upsp options create groups and App/SP pairs by name; new -lc option searches the local cache |
| 0.8.13 | Integrate the former azgrp group functions |
| 0.8.12 | Timing summary for building resource ID maps in the role assignment report; cleaner usage alignment and colors |
| 0.8.11 | Rename MAZ_LOG to MAZLOG; one generic rename option for supported objects; shared parallel fetch for role assignments and definitions |
| 0.8.10 | Log access permission errors instead of masking them; consistent logging across the library |
| 0.8.9 | Credentials file, token cache, and config directory become globals set at init; streamlined token acquisition, decoding, and verification; simpler Logf |
| 0.8.8 | -td verifies Azure token signatures; token decoding moves to token_decode.go; Log renamed Logf; source files renamed with underscores |
| 0.8.7 | Better debug output from interactive token acquisition |
| 0.8.6 | Interactive token acquisition prompts less often, tolerates temporary network issues, and re-authenticates only when needed; Log prints file and line |
| 0.8.5 | API calls report errors to stderr when MAZ_LOG is set; raf no longer compiled by default because azm -sfn covers it |
| 0.8.4 | Token debugging release 2 |
| 0.8.3 | Token debugging release |
| 0.8.2 | Fix token acquisition issues |
| 0.8.1 | Correct the previous Normalize fix |
| 0.8.0 | Fix the Normalize bottleneck on an empty cache |
| 0.7.4 | Credentials and App/SP values print in a YAML-like layout |
| 0.7.3 | Resource lookups start moving to the faster Azure Resource Graph API; -a finds assignments hidden under resource group scopes; generic GetIdNameMap replaces four ID map helpers; -Xi and -Xn options removed |
| 0.7.2 | New -iX and -nX options print an ID or name given the other; new -sfn option suggests a specfile name; fix the -dr delta fetch bug tied to the $top filter |
| 0.7.1 | Optimized cache Normalize with batch deletes and locking during in-memory operations; Delete and Upsert no longer save implicitly |
| 0.7.0 | Delta and full sync optimization: regular pagination for initial syncs, exponential backoff on HTTP 429, fallback to full sync when delta tokens fail, a five-worker pool with non-blocking result processing, graceful shutdown, and progress every 100 items |
| 0.6.7 | raf moves to cmd/raf and the build installs it; add raf.py |
| 0.6.6 | Add the stand-alone raf script for resource role specfile names; prepare main.go for the azapp and azgrp options |
| 0.6.5 | Objects print with attribute names such as displayName; -k skeleton options take an object name that also names the file; bring in the group benchmark |
| 0.6.4 | Fix the 'object was still created' check; MAZ_LOG debug logging; die and printf helpers; many functions now die with a message instead of returning errors |
| 0.6.3 | Fix adding a new directory object |
| 0.6.2 | -vs now compares App/SP specfiles too |
| 0.6.1 | Remove a float assertion in SpsCountAzure that caused a runtime panic |
| 0.6.0 | Simplify UpsertAppSp and UpsertGroup on the generic GetObjectFromFile; add IsResRoleDefinition and IsResRoleAssignment |
| 0.5.0 | Rename Rbac functions to ResRole, use constants for object type codes, and say resource rather than RBAC throughout |
| 0.4.1 | Role definition upserts pretty-print the diff against Azure; role assignment create and delete work as expected |
| 0.4.0 | Role assignments complete the new object model; old cache helpers removed; append replaces AzureObjectList.Add; cache refresh also triggers on an empty cache; jwt bumped to 5.2.2 for CVE-2025-30204; -vs colors updated fields correctly |
| 0.3.2 | Consistent, faster id-to-name map builders; path.Base and LastElemByDot replace generic helpers; PrintTersely uses AzureObject for every type |
| 0.3.1 | Drop the JsonType sugar in API calls; DeleteObjectByName for supported objects; -rm works for resource role definitions |
| 0.3.0 | Major renaming from resource role to RBAC across files and functions; subscription API calls use api-version 2024-11-01; API errors are reported with prettier messages instead of panicking; object type codes become constants; PrintApp shows federated_credentials and the aud list |
| 0.2.0 | Management groups migrate to the Cache model with api-version 2023-04-01 and PrintAzureMgmtGroupTree; subscriptions use api-version 2024-11-01; FindAzureObjectsById; main.go delegates to PrintMatchingObjects |
| 0.1.4 | Subscriptions migrate to the Cache model; GetMatchingDirObjects for Graph objects and GetMatchingObjects as the generic matcher for any supported object |
| 0.1.3 | Improved build script; Api*Verbose naming; rewritten CheckApiError; -xx no longer loads the cache first; fixes for directory objects whose IDs are not UUIDs |
| 0.1.2 | Work in progress: directory objects mostly working with cache bugs, resource objects not yet migrated, options still being ported; error checking moves into ApiCall |
| 0.1.1 | Options -uuid, -tmg, -taz, -tc, and -st ported; basic and extended usage messages; IsValidTokenFormat; DecodeJwtToken shows the base64 signature |
| 0.1.0 | Initial combined pkg/maz library and cmd/azm utility replacing the separate azapp and azgrp tools; device code login for VMs; AzureObject, AzureObjectList, and Cache types; generic per-type fetch, match, count, create, update, delete, and rename functions; YAML-only skeleton specfiles; new Config type; group create, update, and delete; introduce a release notes file |
