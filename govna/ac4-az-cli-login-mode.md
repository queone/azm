# AC4 Borrow ARM and Graph tokens from the Azure CLI as a third login mode

## Summary

`azm` gains an opt-in `az_cli` login mode. When it is configured, `maz` asks the locally installed Azure CLI for its Azure Resource Manager (ARM) and Microsoft Graph bearer tokens by running `az account get-access-token`, instead of running its own MSAL flows. A user who already runs `az login` then needs no browser popup, device code, or client secret for `azm`.

The two existing modes stay the default and unchanged: interactive browser login through the MSAL public client, and automated client id plus secret login through the MSAL confidential client. The Azure CLI is required only when a user opts in, so the no-CLI design goal in `arch.md` holds for the default path. `maz` library callers that never set the new mode see no change.

The Azure CLI owns token caching and refresh in this mode. `maz` runs `az` once per API when a run starts, accepts the returned token after the same structural check the `MAZ_AZ_TOKEN` and `MAZ_MG_TOKEN` path already applies, and does not write `token_cache.json`. No new module dependency is added; the process call uses the standard library.

Code impact: one new file in `pkg/maz`, changes in `pkg/maz/maz_core.go` and `cmd/azm/main.go`, new tests, and documentation updates in `README.md` and `arch.md`. The change is user-visible, so the release takes a MINOR bump at Package time.

### Settled decisions

- Selection key: `az_cli: true` in `credentials.yaml`, or `MAZ_AZ_CLI=true` in the environment.
- Precedence: supplied `MAZ_AZ_TOKEN` plus `MAZ_MG_TOKEN` first, then `az_cli`, then `interactive` plus `username`, then `client_id` plus `client_secret`. The same order applies to the credentials file. Environment variables still beat the file.
- `tenant_id` stays required and must be a UUID in this mode, as in the other two. It is passed to `az` as `--tenant`, so the token comes from the configured tenant even when the Azure CLI's current default is a different one.
- Command: `az account get-access-token --resource <url> --tenant <tenant_id> --output json`. The resource URL is the requested scope with its `/.default` suffix removed, so ARM and Graph share one code path.
- Token acceptance: parse the `accessToken` field and require it to pass `SplitJWT`. No JWKS signature check, no expiry bookkeeping, and no `maz`-side cache in this AC.
- Errors: when `az` is not on `PATH`, the error names the Azure CLI, says it is not found, and points to `-id` to pick another mode. When `az` exits non-zero, the error includes the `az` standard error text and tells the user to run `az login`.
- CLI setup form: `azm -id az TenantId`. The literal `az` comes first so the form cannot collide with the existing `-id TenantId Username` form when a username happens to be `az`.
- `-tx` keeps its current behavior. It removes `maz` files only and never runs `az logout`.
- Testability: the `az` process call sits behind one package-level runner value that tests replace, so no test executes `az`.

## In Scope

### Files to create

- `pkg/maz/token_az_cli.go` — `GetTokenFromAzCli(scopes []string, z *Config) (string, error)`; a helper that derives the resource URL from a scope; a helper that builds the exact `az` argument list; a helper that parses the `az` JSON output and validates the token structurally; the package-level runner value that wraps the standard-library process call; error wrapping for the missing-binary and non-zero-exit cases.
- `pkg/maz/token_az_cli_test.go` — tests for the helpers above and for `GetApiToken` dispatch using a fake runner.
- `pkg/maz/maz_core_test.go` — tests for credentials selection from environment variables and from a credentials file under a temporary config directory, and for the credentials-file text the new `-id az` form writes.

### Files to modify

- `pkg/maz/maz_core.go` — add `AzCli bool` to `Config`; add `MAZ_AZ_CLI` to the environment-variable map; read `az_cli` in `SetupMazCredentialsFromEnvVars` and `SetupMazCredentialsFromFile` with the settled precedence; route `GetApiToken` to `GetTokenFromAzCli` when `AzCli` is set; add `ConfigureCredsFileForAzCliLogin` and build its file text through a helper tests can call; print `MAZ_AZ_CLI` and `az_cli` in `DumpLoginValues` and extend its precedence comment.
- `cmd/azm/main.go` — handle `-id az TenantId` in the three-argument `-id` case before the interactive form; add the usage line for the new form.
- `README.md` — add a short login paragraph under Getting Started that names the three modes, their `-id` forms, and the `az login` prerequisite for the new mode.
- `arch.md` — update System Summary, the external-integrations bullet, and the Core Files list to mention the Azure CLI mode and `token_az_cli.go`.

### Schema changes

None. `credentials.yaml` gains one optional boolean key; existing files keep working unchanged.

## Out Of Scope

- Replacing or removing the MSAL modes, `pkg/maz/token.go`, the MSAL module dependency, or `token_cache.json`.
- Caching borrowed tokens on the `maz` side to avoid two `az` process starts per run. Deferred to `plan.md` IE2 until the timing recorded by AT10 shows it matters.
- Making `tenant_id` optional or deriving it from the token's `tid` claim.
- Running `az logout`, reading or editing the Azure CLI's own token cache, or any other change to Azure CLI state.
- Adopting `azidentity` or any other Azure SDK for Go module.
- Managed identity, Cloud Shell, and other Azure CLI account types that reject `--tenant`.
- Changing the precedence or behavior of the two existing login modes.
- CHANGELOG and version edits, which happen at Package time.

## Migration findings

None. This AC is not audit-emitted.

## Acceptance Tests

**AT1** [Automated] [Pre-release gate] — The `az` argument list is exactly `account get-access-token --resource https://management.azure.com --tenant <tenant_id> --output json` for the ARM scope and the Graph equivalent for the Graph scope. A scope with no `/.default` suffix passes through as its own resource URL.

**AT2** [Automated] [Pre-release gate] — Parsing `{"accessToken":"<jwt>", ...}` returns the token. Output with no `accessToken` field, or with a value that fails `SplitJWT`, returns an error that names the field.

**AT3** [Automated] [Pre-release gate] — A fake runner returning the standard-library not-found error yields an error whose text names the Azure CLI, says it was not found on `PATH`, and mentions `-id`. A fake runner returning a non-zero exit with standard error text yields an error containing that text and the phrase `az login`.

**AT4** [Automated] [Pre-release gate] — With `MAZ_TENANT_ID` set to a UUID and `MAZ_AZ_CLI=true`, credentials setup sets `AzCli` true and leaves `Interactive` false, even when `MAZ_INTERACTIVE=true` and `MAZ_USERNAME` are also set. With `MAZ_AZ_TOKEN` and `MAZ_MG_TOKEN` both structurally valid, setup returns early with `AzCli` false.

**AT5** [Automated] [Pre-release gate] — A credentials file in a temporary config directory containing `tenant_id` and `az_cli: true` sets `AzCli` true. A file containing both `az_cli: true` and `interactive: true` sets `AzCli` true and `Interactive` false.

**AT6** [Automated] [Pre-release gate] — With `AzCli` true, `GetApiToken` calls the injected runner once with the AT1 argument list and returns the runner's token. The MSAL paths are not reached.

**AT7** [Automated] [Pre-release gate] — The credentials-file text for `-id az <tenant>` is exactly two lines, `tenant_id: <tenant>` and `az_cli: true`, in the same column layout the existing two forms use.

**AT8** [Automated] [Pre-release gate] — `rg -n '"  -id az TenantId' cmd/azm/main.go` matches once; `rg -n 'az_cli' README.md` and `rg -n 'az_cli' arch.md` each match at least once; `rg -n 'token_az_cli.go' arch.md` matches once.

**AT9** [Manual] [Pre-release gate] — After `az login`, run `azm -id az <tenant>`, then `azm -taz` and `azm -tmg`. Each prints a JWT, and `azm -td` on each shows an `aud` of the ARM and Graph resource respectively. `azm -ap` then lists applications with no browser or device-code prompt.

**AT10** [Manual] [Pre-release gate] — With `-id az` configured, record `time azm -st` for three runs and the same for the interactive mode with a warm MSAL cache. Report the numbers in the Implement completion report to inform IE2.

**AT11** [Manual] [Pre-release gate] — After `az logout`, `azm -ap` exits non-zero with an error that names `az login`. With `az` removed from `PATH`, `azm -ap` exits non-zero with an error that names the Azure CLI as not found. `azm -id` shows `az_cli: true`, and `azm -tx` removes the credentials file while `az account show` still reports a logged-in account.

## Status

`PENDING` — drafted and held at Draft by Director instruction; awaiting a Director request to Audit.
