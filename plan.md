# azm Plan

## Product Direction

`azm` is a small Go command-line utility, and `maz` is the Go library behind it, for querying and managing a limited set of Azure IAM objects through direct REST calls to the Azure Resource Manager and Microsoft Graph APIs. Together they are a lightweight, customizable alternative to the official Azure SDK for Go and the Azure CLI for specialized use cases. The repository is active.

## Ideas To Explore

Ideas captured for future reference. A bullet list — each line starts with `- IE<N>: ` (sequential N) for stable references. Two kinds: (a) **pre-rubric IE** — `IE<N>: <one-liner>`, awaiting director discussion and the objective-fit rubric (see `AGENTS.md` Approval Boundaries); (b) **AC-pointer** — `IE<N>: <one-liner> → govna/ac<N>-<slug>.md`, pointing at a drafted AC stub not yet through critique. A pre-rubric entry that clears the rubric converts to an AC-pointer at AC-draft time, keeping its `IE<N>` number. Remove entries when the idea is rejected, retired, or (for AC-pointers) the AC has shipped and its file deleted. Not a historical record.

- IE1: Migrate `azm` off `queone/utl`.
- IE2: Make the config directory XDG-friendly: resolve config and credentials to `$XDG_CONFIG_HOME/maz` (default `~/.config/maz`) and the cached directory snapshots to `$XDG_CACHE_HOME/maz` (default `~/.cache/maz`), falling back to an existing `~/.maz`; today `pkg/maz/maz_core.go` hardcodes `ConfigBaseDir = ".maz"` under the home directory.
- IE3: Retire `cmd/raf` now that `azm -sfn` generates specfile names directly; the old `build` script had already stopped installing it.
