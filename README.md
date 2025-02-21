# azm
This repository hosts two components:

- The [`pkg/maz`](pkg/maz/README.md) library: a small Go library for managing a limited set of Azure IAM objects and functions
- The [`cmd/azm`](cmd/azm/README.md) utility: a small utility that implements the `maz` library to manage Azure IAM objects

For more information, click on each component link above to open their respective READMEs.

## Introduction
The azm repository provides a lightweight alternative to the official Azure SDK for Go and Azure CLI tool. It's designed for specialized use cases and provides a simple, customizable solution for managing Azure IAM objects.

## Why?
There are already actively maintained [Azure SDKs](https://github.com/Azure/azure-sdk-for-go) and [CLI tools](https://learn.microsoft.com/en-us/cli/azure/), so why create another library and utility? Here are a few reasons:
- **Learning and experimentation**: Building a custom SDK can be a great way to learn about Go and REST API development.
- **Specialized use cases**: If your application only interacts with a smaller subset of Microsoft Graph APIs, a lightweight custom SDK might be simpler and faster.
- **Control over dependencies**: The official SDK might introduce dependencies or abstractions you want to avoid.
- **Custom abstractions**: If you need custom abstractions or behavior not easily achieved with the official SDK, a custom library can provide more flexibility.
- **Direct API access**: This library performs direct HTTPS REST API calls, following the official Microsoft Azure API documentation, providing a straightforward and efficient way to interact with Azure services.
- **Quick search capabilities**: For instance, the `azm` utility allows for quick-and-dirty searches of App/SP pairs by App name, App ID, or object ID, making it a convenient tool for rapid exploration and discovery.
- **Cross-platform compatibility**: The `azm` utility can be quickly compiled to run on Windows, macOS, or Linux, providing a flexible and portable solution for managing Azure IAM objects.

## Getting Started
To get started with azm, follow these steps:
- Clone the repository: `git clone https://github.com/queone/azm`
- Change into the repository directory: `cd azm`
- Build the azm utility: `./build`
- Run the azm utility without arguments to print the usage page.
- Try experimenting with different options and arguments.

## Releases

See [releases](releases.md) for a list of available releases and changelogs.
