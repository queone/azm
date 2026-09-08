# azm Architecture

## Purpose

`azm` gives an operator a fast, scriptable way to look up and manage Azure IAM objects, such as role definitions, role assignments, subscriptions, management groups, users, groups, applications, service principals, and directory roles, without the Azure SDK or the Azure CLI. The `maz` library provides the same capability to other Go programs.

## System Summary

A user runs `azm` with a short option code that names an object type and an optional filter, UUID, or YAML specfile. The `maz` library authenticates the caller with MSAL, obtains tokens for the Azure Resource Manager (ARM) and Microsoft Graph APIs, and issues direct HTTPS REST calls. Directory objects are fetched with Graph delta queries and kept in a local cache so repeated searches stay fast. Results print as text, JSON, or YAML, and write options create, update, or delete objects after confirmation.

## Current Platform

- Go

## Major Components

- `cmd/azm`: the command-line entrypoint; parses one to four positional arguments and dispatches to `maz`
- `pkg/maz`: the library; authentication and tokens (`token.go`, `token_decode.go`), REST calls (`api_calls.go`), directory objects (`dir_*.go`), resource objects (`res_*.go`), the local object cache (`maz_cache.go`), specfile comparison and skeletons (`specfile.go`, `skeleton.go`), and printing
- `cmd/raf`: a standalone filename generator for role-assignment specfiles; legacy, since `azm -sfn` now covers it
- `misc/`: scratch benchmarks and scripts that are not shipped
- external integrations: Microsoft Entra ID for MSAL logon (user device code, service principal secret, or OIDC), ARM for resource objects, and Microsoft Graph for directory objects
- local state: credentials, configuration, and cached object snapshots under `~/.maz`

## Core Files

- `AGENTS.md`: base governance contract
- `plan.md`: prioritized roadmap and approved direction
- `build.sh`: self-contained build / release-prep / release script (Bash 3.2+, no external tools)
- `govna/development-cycle.md`: workflow from roadmap through release
- `govna/ac-template.md`: acceptance-criteria template for new work
- `govna/build-release.md`: build, test, and release rules
- `cmd/azm/main.go`: usage text, argument dispatch, and the `programVersion` declaration
- `pkg/maz/maz_core.go`: configuration, credentials, and the `Config` object every call carries

## Data And Control Flow

`azm` validates the argument count, builds a `maz.Config`, and handles the few options that need no API access, such as printing login values or generating a UUID. For every other option it sets up API tokens, which loads or refreshes credentials from `~/.maz`, then calls the matching `maz` function. Read options resolve the object type from the option code, consult or refresh the local cache through delta queries, and print matching objects. Write options read a YAML specfile or arguments, compare them with the live tenant, prompt for confirmation unless forced, and call ARM or Graph to apply the change.

## AC Lifecycle Control Flow

The governed change path is `Draft → Audit → Refine → Implement → Ratify → Package`. Draft creates the AC; Audit, Refine, Implement, and Ratify are the four AC phases; Package is post-Ratify release preparation and is not a fifth phase.

Integrated audit adoption is the only command-mediated phase exception. It can advance one emitted adoption AC through immediate Audit and no-edit Refine, but it cannot enter Implement. Every unpackaged AC with implementation in the unreleased state enters the pending release batch, including work awaiting Ratify. A private pre-Implement calculation prevents that complete batch from growing beyond one 80-byte prefix-plus-summary message. Package requires every member to be Ratified, rejects excluded implemented work, and rechecks the complete batch before prep. A named request such as `Package AC70+AC71` establishes a fitting multi-AC batch; a standalone Package alias reuses the complete batch already established in the active session.

## Architecture Notes

- call ARM and Graph directly over HTTPS rather than through the Azure SDK; follow the official REST documentation
- cache directory objects locally and refresh them with Graph delta queries, resuming an interrupted delta fetch on the next run
- keep the Go module path `azm` with a `replace` to `github.com/queone/azm` so the library imports resolve both locally and from the published module
- the `azm` utility owns the repository release version; `raf` carries an independent version

## Conventions

- update this document when architecture or major workflow changes materially
- keep implementation detail in code and stable architecture here
