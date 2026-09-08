# azm
This repository hosts two components:

- The [`pkg/maz`](pkg/maz/README.md) library: a small Go library for managing a limited set of Azure IAM objects and functions
- The [`cmd/azm`](cmd/azm/README.md) utility: a small utility that implements the `maz` library to manage Azure IAM objects

They provide a lightweight alternative to the official Azure SDK for Go and the Azure CLI tool. They are also designed for limited, specialized use cases as a simple, customizable solution for managing Azure IAM objects.

## Why?
The [Azure SDKs](https://github.com/Azure/azure-sdk-for-go) and [CLI tools](https://learn.microsoft.com/en-us/cli/azure/) are well-maintained, so why another library and utility?:
- **Learning and experimentation**: Building a custom SDK can be a great way to learn about Go and REST API development.
- **Specialized use cases**: If your application only interacts with a smaller subset of Microsoft Graph APIs, a lightweight custom SDK might be simpler and faster.
- **Direct API access**: This library performs direct HTTPS REST API calls, following the official Microsoft Azure API documentation, providing a straightforward and efficient way to interact with Azure services.
- **Cross-platform compatibility**: The `azm` utility can be quickly compiled to run on Windows, macOS, or Linux, providing a flexible and portable solution for managing Azure IAM objects.
- **Quick search capabilities**: For instance, the `azm` utility allows for quick-and-dirty searches of App/SP pairs by App name, App ID, or object ID, making it a convenient tool for rapid exploration and discovery.

## Getting Started
To get started with azm, follow these steps:
- Clone the repository: `git clone https://github.com/queone/azm`
- Change into the repository directory: `cd azm`
- Build, validate, and install the azm utility: `./build.sh`
- Run the azm utility without arguments to print the usage page
- For extended usage info do `azm -?`
- Try experimenting with different options and arguments

`./build.sh` is the canonical build command. It tidies dependencies, formats, vets, tests, runs staticcheck, compiles every utility under `cmd/`, and installs the binaries into `$(go env GOPATH)/bin`. Pass utility names to scope the build, for example `./build.sh azm`.

## Governance
This repo is governed by an explicit session-entry contract for AI coding agents. See [`AGENTS.md`](AGENTS.md) for the operational rules and [`govna/operator-contract-rationale.md`](govna/operator-contract-rationale.md) for the design reasoning. Governance files are maintained with [govna](https://github.com/queone/govna).

## Releases
Every release, including the history that predates the govna adoption, is recorded in [CHANGELOG.md](CHANGELOG.md).
