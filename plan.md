# azm Plan

## Product Direction

`azm` is a small Go command-line utility, and `maz` is the Go library behind it, for querying and managing a limited set of Azure IAM objects through direct REST calls to the Azure Resource Manager and Microsoft Graph APIs. Together they are a lightweight, customizable alternative to the official Azure SDK for Go and the Azure CLI for specialized use cases. The repository is active.

## Ideas To Explore

Ideas captured for future reference. A bullet list — each line starts with `- IE<N>: ` (sequential N) for stable references. Two kinds: (a) **pre-rubric IE** — `IE<N>: <one-liner>`, awaiting director discussion and the objective-fit rubric (see `AGENTS.md` Approval Boundaries); (b) **AC-pointer** — `IE<N>: <one-liner> → AC<N>`, pointing at a drafted AC stub not yet through critique. A pre-rubric entry that clears the rubric converts to an AC-pointer at AC-draft time, keeping its `IE<N>` number. Remove entries when the idea is rejected, retired, or (for AC-pointers) the AC has shipped and its file deleted. Not a historical record.

- IE1: Evaluate borrowing ARM and Graph bearer tokens from `az account get-access-token` (possibly as a third login mode beside interactive and client-secret) to retire `pkg/maz/token.go`, the MSAL dependency, and `token_cache.json`; weigh against the no-Azure-CLI design goal in `arch.md`, per-invocation `az` startup latency, and `maz` library consumers that would inherit the CLI dependency. → AC4
- IE2: Cache Azure CLI-borrowed tokens on the `maz` side (keyed by resource, reused until near the JWT `exp` claim) to avoid two `az` process starts per run; decide after the AC4 timing measurement.
