## azm

( WORK IN PROGRESS )

`azm` is a [CLI](https://en.wikipedia.org/wiki/Command-line_interface) utility for limited management of [Indentity and Access Management (IAM)](https://www.nist.gov/identity-access-management) related Azure objects.

### Capabilities

- managing [Azure App Registrations and Service Principal pairs](https://learn.microsoft.com/en-us/entra/identity-platform/app-objects-and-service-principals?tabs=browser)


`azm` can perform the following functions.

NOTE: If the logon account does not have the required privileges, some of these functions will simply **not work**.

1. READ Options
    - List all Azure directory group in a tersely manner (group Id and displayName). There's also an option to display them all in JSON format
    - Compare RBAC role definitions and assignments that are defined in a YAML specfile to what that object currently looks like in the Azure tenant
    - List all Privileged Access Groups, which are typically used for PIM.
    - List status count on the number of groups in the Azure tenant, and in the local cache.

2. WRITE Options
    - Quickly create, delete, rename, or update a group, with or without a prompt confirmation.
    - Output a sample, minimalist YAML or JSON specfile representing a group, which can then be used to create/update that group.
    - Does __not__ manage group membership.

3. CONFIG Options
    - Option to logon using 3 different methods: User, SP or ODIC logon.
    - Delete the local cache


### Getting Started

To compile `azm`, first make sure you have installed and set up the Go language on your system. You can do that by following [these instructions](https://que.one/golang/#install-go) or by following other similar recommendations.

- Also ensure that `$GOPATH/bin/` is in your `$PATH`, since that's where the executable binary will be placed.
- Open a `bash` shell, clone this repo, then switch to the `azm` working directory
- Type `./buildgo` to build and install the binary executable
- Note that `buildgo` is only a runner that downloads and runs the actual `buildgo` script sitting remotely at <https://raw.githubusercontent.com/git719/tips/refs/heads/main/scripts/buildgo>
- To build from a *regular* Windows Command Prompt, you can run the corresponding line in the **remote** `buildgo` script (`go build ...`)
- If no errors, you should then be able to type `azm` and see the usage screen for this utility.

This utility has been successfully tested on Windows, multiple versions of Linux, and on macOS. In Windows it works from a regular CMD.EXE or PowerShell prompt, as well as from a GitBASH prompt. Remember that Go allows cross-compilation targets to other OSes, so for example, from a Windows machine you could compile a Linux executable binary.

Below other sections in this README explain how to set up access, and use the utility in your own Azure tenant. 


### Access Requirements

First and foremost you need to know the special **Tenant ID** for your tenant. This is the unique UUID that identifies your Microsoft Azure tenant.

Then, you need a User ID or a Service Principal with the appropriate access rights. Either one will need the necessary privileges to perform the functions described above. For Read-Only functions that typically means getting Entra ID __Directory Reader__ role or MS Graph `Directory.Read.All` permission. 

When you run `azm` without any arguments you will see the **usage** screen listed below in this README. As you can quickly surmise, the `-id` argument will allow you to set up a connection to your tenant in two possible ways. The first way is interactively, with a User ID (aka, a [User Principal Name (UPN)](https://learn.microsoft.com/en-us/entra/identity/hybrid/connect/plan-connect-userprincipalname)), and that's described below in **User Logon**. The the method is using a Service Principal (SP) with a secret, and that's described under **SP Logon**.

Another way of connecting is to leverage an access token that has been acquired in another manner, perhaps via an OIDC login. That is a special version of the SP Logon and a bit more advanced.


#### User Logon

For example, if your Tenant ID was **c44154ad-6b37-4972-8067-0ef1068079b2**, and your User ID UPN was __bob@contoso.com__, you would type: 

```bash
$ azm -id c44154ad-6b37-4972-8067-0ef1068079b2 bob@contoso.com
Updated /Users/myuser/.maz/credentials.yaml file
```

`azm` responds that the special `credentials.yaml` file has been updated accordingly.

To view, dump all configured logon values type the following: 

```bash
$ azm -id
config_dir: /Users/MYUSER/.maz  # Config and cache directory
config_env_variables:
  # 1. MS Graph and Azure ARM tokens can be supplied directly via MAZ_MG_TOKEN and
  #    MAZ_AZ_TOKEN environment variables, and they have the highest precedence.
  #    Note, MAZ_TENANT_ID is still required when using these 2.
  # 2. Credentials supplied via environment variables have precedence over those
  #    provided via credentials file.
  # 3. The MAZ_USERNAME + MAZ_INTERACTIVE combo have priority over the MAZ_CLIENT_ID
  #    + MAZ_CLIENT_SECRET combination.
  MAZ_TENANT_ID:
  MAZ_USERNAME:
  MAZ_INTERACTIVE:
  MAZ_CLIENT_ID:
  MAZ_CLIENT_SECRET:
  MAZ_MG_TOKEN:
  MAZ_AZ_TOKEN:
config_creds_file:
  file_path: /Users/MYUSER/.maz/credentials.yaml
  tenant_id: c44154ad-6b37-4972-8067-0ef1068079b2
  username: bob@contoso.com
  interactive: true
```

Above tells you that the utility has been configured to use Bob's UPN for access via the special credentials file. Note that above is only a configuration setup, it actually hasn't logged Bob onto the tenant yet. To logon as Bob you have have to run any command that actually makes an MS Graph call, and the logon will happen automatically, in this case it will be an interactive browser popup logon.

Note also, that instead of setting up Bob's login with the `-id` argument, you could have setup the special 3 operating system environment variables to achieve the same. Had you done that, running `azapp -id` would have displayed below instead:

```bash
$ azm -id
config_dir: /Users/myuser/.maz  # Config and cache directory
config_env_variables:
  # 1. Credentials supplied via environment variables override values provided via credentials file
  # 2. MAZ_USERNAME+MAZ_INTERACTIVE login have priority over MAZ_CLIENT_ID+MAZ_CLIENT_SECRET login
  MAZ_TENANT_ID: c44154ad-6b37-4972-8067-0ef1068079b2
  MAZ_USERNAME: bob@contoso.com
  MAZ_INTERACTIVE: true
  MAZ_CLIENT_ID:
  MAZ_CLIENT_SECRET:
  MAZ_MG_TOKEN:
  MAZ_AZ_TOKEN:
config_creds_file:
  file_path: /Users/myuser/.maz/credentials.yaml
  tenant_id: 
  username: 
  interactive:
```

#### SP Logon

To use an SP logon it means you first have to set up a dedicated App Registrations, grant it the same role and MS Graph security access roles mentioned above. For how to do an Azure App Registration please reference some other sources on the Web. By the way, this method is NOT RECOMMENDED, as you would be exposing the secret as an environment variables, which is not a very good security practice.

Once above is setup, you then follow the same logic as for User ID logon above, but specifying 3 instead of 2 values; or use the other environment variables (MAZ_CLIENT_ID and MAZ_CLIENT_SECRET). 

The utility ensures that the permissions for configuration directory where the `credentials.yaml` file is only accessible by the owning user. However, storing a secrets in a clear-text file is a very poor security practice and should __never__ be use other than for quick tests, and so on. The environment variable options was developed pricisely for this SP logon pattern, where the utility could be setup to run from say a [Docker container](https://en.wikipedia.org/wiki/Docker_(software)) and the secret injected as an environment variable, and that would be a much more secure way to run the utility.

#### OIDC Logon

An even better security practive when using the SP logon method is to leverage any process that can acquire OIDC tokens and make them available to this utility via the `MAZ_MG_TOKEN` and `MAZ_AZ_TOKEN` environment variable. If using OIDC logon, say for instance within a Github Workflow Action, you need to specify **both** these tokens and also the `MAZ_TENANT_ID` one.

(TODO: Need OIDC setup example, and how configure the SP on the Azure side with federated login.)

These login methods and the environment variables are described in more length in the [maz package README](https://github.com/queone/azm/blob/main/pkg/maz/README.md).


### Quick Examples

1. List any Azure directory group by name: 

```bash
$ azm "My Group"
# Directory Group
id: 6844e17c-b584-40ba-a90c-e9e83675830b
displayName: my-group-8088
description: My Group 6 description
isAssignableToRole: false
$
```

- Another way of listing the same group is to call it by its Object Id: `azapp 6844e17c-b584-40ba-a90c-e9e83675830b`
- The YAML listing format is more human-friendly and easier to read, and only displays the attributes that are the most relevant to most engineers
- You can also display it in `JSON` format by calling: `azapp -j 6844e17c-b584-40ba-a90c-e9e83675830b`
- One advantage of the `JSON` call is that actually goes directly to Azure and grabs the actual object directly from MS Graph, and displays the extended set of attributes.

2. List all groups that have a particular string in their `displayName` or `description` field: 

```bash
$ azm group
4b7fc6ce-1a7e-42b5-b6a5-41b81b5863f8  my-group-1
c16ed929-bb18-4da5-a406-9c71f724917e  my-group-2
6f47e741-485e-4cb8-adc5-260eb175ca03  Special entry
6844e17c-b584-40ba-a90c-e9e83675830b  my-group-8088
c7cf6fdc-c541-456d-b7b2-84728d2789c2  Some other group
$
```

- This is a terse format with only the group's `id` and it's `displayName`
- Note that the search is case-insensitive
- Also note that some matching entries may seem correct, but that's because the match happened on the `description` which is not shown in this format.

3. Create a group named `My Group`:

```bash
$ azm -up "My Group"
Create NEW group with below attributes:
displayName: My Group
mailEnabled: false
mailNickname: NotSet
securityEnabled: true
Create group? y/n y
id: 9992c5f9-9b2d-4a29-8142-17baa8136074  # Created
$
```

4. ( TODO: Provide other examples ... )

### Known Issues

The program is stable enough to be relied on as a quick, useful utility. There are a number of little things that can of course be improved. Please see [Planned Features in releases](releases.md) page. In general, is also worth remembering [Tony Hoare](https://en.wikipedia.org/wiki/Tony_Hoare)'s famous quote: "_Inside every large program is a small program struggling to get out_", so I'm sure there's a small bug here and there.

### Feedback
The primary goal of this utility is to serve as a study aid for coding Azure utilities in the Go language, so the code is deliberately kept simple and clear for this reason.

This utility also serves as a quick little _Swiss Army knife_ for managing Azure App/SP pairs. Note that the bulk of the code is actually in the [maz](https://github.com/queone/azm/blob/main/pkg/maz/README.md) library, and other supporting packages. Please visit those repos for more info.

This is published as an open source project, so feel free to clone and use on your own, with proper attributions. Feel free to reach out if you have any questions or comments.



## ========================================================================

### Capabilities

- Only focuses on the smaller set of Azure objects that are related to IAM 
- Do quick and dirty searches of any IAM related object types in the azure tenant
- Supports leveraging OIDC Github Action workflows with no passwords for a configured Azure Service Principal
- Developed as part of a framework library for acquiring Azure [JWT](https://jwt.io/) token using the [MSAL library for Go](https://github.com/AzureAD/microsoft-authentication-library-for-go) (leverages [maz](https://github.com/queone/azm/blob/main/pkg/maz/README.md) library)
- Quickly get a token to access an Azure tenant's **Resources** Services API via endpoint <https://management.azure.com> ([REST API](https://learn.microsoft.com/en-us/rest/api/azure/))
- Quickly get a token to access an Azure tenant's **Security** Services API via endpoint <https://graph.microsoft.com> ([MS Graph](https://learn.microsoft.com/en-us/graph/overview))

This is a little _Swiss Army knife_ that can very quickly perform the following functions:
1. Read-Only Functions
    > **Note**<br>
    If logon account does not have the required *Read-Only* privileges, below functions will fail
    - List the following [Azure Resources Services](https://que.one/azure/#azure-resource-services) objects in your tenant:
        - RBAC Role Definitions
        - RBAC Role Assignments
        - Azure Subscriptions
        - Azure Management Groups
    - List the following [Azure Security Services](https://que.one/azure/#azure-security-services) objects:
        - Azure AD Users
        - Azure AD Groups
        - Applications
        - Service Principals
        - Azure AD Roles that have been **activated**
        - Azure AD Roles standard definitions
    - Compare RBAC role definitions and assignments that are defined in a YAML __specification file__ to what that object currently looks like in the Azure tenant
    - Dump the current Resources or Security JWT tokens (see **pman** below)
    - Perform *many* other related listing functions
2. Read-Write Functions
    > **Note**<br>
    Again, if logon account does not have the required *Read-Write* privileges, below functions will fail
    - Delete/Create/Update the following [Azure Resources Services](https://que.one/azure/#azure-resource-services) objects in your tenant:
        - RBAC Role Definitions
        - RBAC Role Assignments
    - Can output a sample RBAC Role definition or assignment YAML __specification file__, that can then be used to create a new role or assignment
    - Update the following [Azure Security Services](https://que.one/azure/#azure-security-services) objects:
        - Service Principals: Can only create or delete SP secrets (Cannot yet create SPs)
        - Applications: Can only create or delete App secrets (Cannot yet create Apps)
    - Create a UUID
    - Other functions may be added


### Getting Started

To compile `azm`, first make sure you have installed and set up the Go language on your system. You can do that by following [these instructions here](https://que.one/golang/#install-go-on-macos) or by following other similar recommendations found across the web.

- Also ensure that `$GOPATH/bin/` is in your `$PATH`, since that's where the executable binary will be placed.
- Open a `bash` shell, clone this repo, then switch to the `azm` working directory
- Type `./build` to build the binary executable
- To build from a *regular* Windows Command Prompt, just run the corresponding line in the `build` file (`go build ...`)
- If there are no errors, you should now be able to type `azm` and see the usage screen for this utility.

This utility has been successfully tested on macOS, Ubuntu Linux, and Windows. In Windows it works from a regular CMD.EXE, or PowerShell prompts, as well as from a GitBASH prompt. Remember that Go allows cross-compilation targets to be other OSes, so for examplce, from a Windows machine you can compile a Linux executable binary.

Below other sections in this README explain how to set up access and use the utility in your own Azure tenant. 


### Access Requirements

First and foremost you need to know the special **Tenant ID** for your tenant. This is the unique UUID that identifies your Microsoft Azure tenant.

Then, you need a User ID or a Service Principal with the appropriate access rights. Either one will need the necessary privileges to perform the functions described above. For Read-Only functions that typically means getting _Reader_ role access to read **resource** objects, and _Global Reader_ role access to read **security** objects. The higher the scope of these access assignments, the more you will be able to see with the utility. 

When you run `azm` without any arguments you will see the **usage** screen listed below in this README. As you can quickly surmise, the `-id` argument will allow you to set up these 2 optional ways to connect to your tenant; either interactively with a User ID, also known as a [User Principal Name (UPN)](https://learn.microsoft.com/en-us/entra/identity/hybrid/connect/plan-connect-userprincipalname), or using a Service Principal or SP with a secret.

Another way of connecting is to use access tokens that have been acquired in another manner, perhaps via an OIDC login. 


#### User Logon

For example, if your Tenant ID was **c44154ad-6b37-4972-8067-0ef1068079b2**, and your User ID UPN was __bob@contoso.com__, you would type:

```
$ azm -id c44154ad-6b37-4972-8067-0ef1068079b2 bob@contoso.com
Updated /Users/myuser/.maz/credentials.yaml file
```
`azm` responds that the special `credentials.yaml` file has been updated accordingly.

To view, dump all configured logon values type the following:

```bash
$ azm -id
config_dir: /Users/myuser/.maz  # Config and cache directory
config_env_variables:
  # 1. MS Graph and Azure ARM tokens can be supplied directly via MAZ_MG_TOKEN and
  #    MAZ_AZ_TOKEN environment variables, and they have the highest precedence.
  #    Note, MAZ_TENANT_ID is still required when using these 2.
  # 2. Credentials supplied via environment variables have precedence over those
  #    provided via credentials file.
  # 3. The MAZ_USERNAME + MAZ_INTERACTIVE combo have priority over the MAZ_CLIENT_ID
  #    + MAZ_CLIENT_SECRET combination.
  MAZ_TENANT_ID:
  MAZ_USERNAME:
  MAZ_INTERACTIVE:
  MAZ_CLIENT_ID:
  MAZ_CLIENT_SECRET:
  MAZ_MG_TOKEN:
  MAZ_AZ_TOKEN:
config_creds_file:
  file_path: /Users/myuser/.maz/credentials.yaml
  tenant_id: c44154ad-6b37-4972-8067-0ef1068079b2
  username: bob@contoso.com
  interactive: true
```

Above tells you that the utility has been configured to use Bob's UPN for access via the special credentials file. Note that above is only a configuration setup, it actually hasn't logged Bob onto the tenant yet. To logon as Bob you have have to run any command, and the logon will happen automatically, in this case it will be an interactive browser popup logon.

Note also, that instead of setting up Bob's login with the `-id` argument, you could have setup the special 3 operating system environment variables to achieve the same. Had you done that, running `azm -id` would have displayed below instead:

```bash
$ azm -id
config_dir: /Users/myuser/.maz  # Config and cache directory
config_env_variables:
  # 1. Credentials supplied via environment variables override values provided via credentials file
  # 2. MAZ_USERNAME+MAZ_INTERACTIVE login have priority over MAZ_CLIENT_ID+MAZ_CLIENT_SECRET login
  MAZ_TENANT_ID: c44154ad-6b37-4972-8067-0ef1068079b2
  MAZ_USERNAME: bob@contoso.com
  MAZ_INTERACTIVE: true
  MAZ_CLIENT_ID:
  MAZ_CLIENT_SECRET:
  MAZ_MG_TOKEN:
  MAZ_AZ_TOKEN:
config_creds_file:
  file_path: /Users/myuser/.maz/credentials.yaml
  tenant_id: 
  username: 
  interactive:
```

#### SP Logon

To use an SP logon it means you first have to set up a dedicated App Registrations, grant it the same Reader resource and Global Reader security access roles mentioned above. For how to do an Azure App Registration please reference some other sources on the Web. By the way, this method is NOT RECOMMENDED, as you would be exposing the secret as an environment variables, which is not very good security practice.

Once above is setup, you then follow the same logic as for User ID logon above, but specifying 3 instead of 2 values; or use the other environment variables (MAZ_CLIENT_ID and MAZ_CLIENT_SECRET). 

The utility ensures that the permissions for configuration directory where the `credentials.yaml` file is only accessible by the owning user. However, storing a secrets in a clear-text file is a very poor security practice and should __never__ be use other than for quick tests, etc. The environment variable options was developed pricisely for this SP logon pattern, where the utility could be setup to run from say a [Docker container](https://en.wikipedia.org/wiki/Docker_(software)) and the secret injected as an environment variable, and that would be a much more secure way to run the utility.

An even better security practive when using the SP logon method is to leverage any process that can acquire OIDC tokens and make them available to this utility via the `MAZ_MG_TOKEN` and `MAZ_AZ_TOKEN` environment variable. If using OIDC logon, say for instance within a Github Workflow Action, you need to specify **both** these tokens and also the `MAZ_TENANT_ID` one.

(TODO: Need OIDC setup example, and how configure the SP on the Azure side with federated login.)

These login methods and the environment variables are described in more length in the [maz README](https://github.com/queone/azm/blob/main/pkg/maz/README.md).


### Quick Examples

1. List any Azure RBAC role, like the Built-in "Billing Reader" role for example:

```bash
$ azm -d "Billing Reader"
id: fa23ad8b-c56e-40d8-ac0c-ce449e1d2c64
properties:
  roleName: Billing Reader
  description: Allows read access to billing data
  assignableScopes:
    - /
  permissions:
    - actions:
        - Microsoft.Authorization/*/read
        - Microsoft.Billing/*/read
        - Microsoft.Commerce/*/read
        - Microsoft.Consumption/*/read
        - Microsoft.Management/managementGroups/read
        - Microsoft.CostManagement/*/read
        - Microsoft.Support/*
      notActions:
      dataActions:
      notDataActions:
```

- Another way of listing the same role is to call it by its UUID: `azm -d fa23ad8b-c56e-40d8-ac0c-ce449e1d2c64`
- The YAML listing format is more human-friendly and easier to read, and only displays the attributes that are most relevant to Azure systems engineers
- You can also display it in JSON format by calling: `azm -dj fa23ad8b-c56e-40d8-ac0c-ce449e1d2c64`
- One advantage of the JSON format is that it displays every single attribute in the Azure object

2. Add a secret to an Application object: 

```bash
$ azm -apas 51afab9e-0225-4c36-81f0-f42289c1a57a "My Secret"
App_Object_Id: 51afab9e-0225-4c36-81f0-f42289c1a57a
New_Secret_Id: 7c140771-c547-43f9-8525-d08bd234e267
New_Secret_Name: My Secret
New_Secret_Expiry: 2025-01-06
New_Secret_Text: <deliberately obsfucated>
```

As the **usage** section shows, the secret Expiry defaults to 366 days if none is given. 

- Note that you have to use the **Objectd ID**, not the App ID (Client ID) of the application
- The name could have been nulled with `""`
- To remove above secret, you can simply do: `azm -aprs 51afab9e-0225-4c36-81f0-f42289c1a57a 7c140771-c547-43f9-8525-d08bd234e267`

3. Generate a random [UUID](https://en.wikipedia.org/wiki/Universally_unique_identifier). This can be very handy sometimes. Simply do: `azm -uuid`

### Usage
Run the `azm` utility without arguments to get the default usage message: 

```bash
$ azm
azm v0.1.2
Azure IAM CLI manager - github.com/queone/azm
Usage
  azm [options] [arguments]

  This utility simplifies the querying and management of various Azure IAM-related objects.
  In many options X is a placeholder for a 1-2 character code that specifies the type of
  Azure object to act on. The available codes are:

    d  = Resource Role Definitions     a  = Resource Role Assignments
    s  = Resource Subscriptions        m  = Resource Management Groups
    u  = Directory Users               g  = Directory Groups
    ap = Directory Applications        sp = Directory Service Principals
    dr = Directory Role Definitions    da = Directory Role Assignments

  In those options, replace X with the corresponding code to specify the object type.

Quick Examples
  Try experimenting with different options and arguments, such as:
  azm -id                                      To display the currently configured login values
  azm -ap                                      To list all directory applications registered in
                                               current tenant
  azm -d 3819d436-726a-4e40-933e-b0ffeee1d4b9  To show resource RBAC role definition with this
                                               given UUID
  azm -d Reader                                To show all resource RBAC role definitions with
                                               'Reader' in their names
  azm -g MyGroup                               To show any directory group with the filter
                                               'MyGroup' in its attributes
  azm -s                                       To list all subscriptions in current tenant
  azm -h                                       To display the full list of options
```

Run the utility with the `-h` argument to get the full list of options: 

```bash
$ azm -h
azm v0.1.2
Azure IAM CLI manager - github.com/queone/azm
Usage
  azm [options] [arguments]

  This utility simplifies the querying and management of various Azure IAM-related objects.
  In many options X is a placeholder for a 1-2 character code that specifies the type of
  Azure object to act on. The available codes are:

    d  = Resource Role Definitions     a  = Resource Role Assignments
    s  = Resource Subscriptions        m  = Resource Management Groups
    u  = Directory Users               g  = Directory Groups
    ap = Directory Applications        sp = Directory Service Principals
    dr = Directory Role Definitions    da = Directory Role Assignments

  In those options, replace X with the corresponding code to specify the object type.

Quick Examples
  Try experimenting with different options and arguments, such as:
  azm -id                                      To display the currently configured login values
  azm -ap                                      To list all directory applications registered in
                                               current tenant
  azm -d 3819d436-726a-4e40-933e-b0ffeee1d4b9  To show resource RBAC role definition with this
                                               given UUID
  azm -d Reader                                To show all resource RBAC role definitions with
                                               'Reader' in their names
  azm -g MyGroup                               To show any directory group with the filter
                                               'MyGroup' in its attributes
  azm -s                                       To list all subscriptions in current tenant
  azm -h                                       To display the full list of options

Read Options (allow reading and querying Azure objects)
  UUID                             Show all Azure objects associated with the given UUID
  -X[j] [FILTER]                   List all X objects tersely; optional JSON output; optional
                                   match on FILTER string for Id, DisplayName, and other attributes.
                                   If the result is a single object, it is printed in more detail.
  -vs SPECFILE                     Compare YAML specfile to what's in Azure. Only for certain objects.
  -ar                              List all RBAC role assignments with resolved names
  -mt                              List Management Group and subscriptions tree
  -pags                            List all Azure AD Privileged Access Groups
  -st                              Show count of all objects in local cache and Azure tenant
  -tmg                             Display current Microsoft Graph API access token
  -taz                             Display current Azure Resource API access token
  -tc "TokenString"                Parse and display the claims contained in the given token

Write Options (allow creating and managing Azure objects)
  -kX                              Generate a YAML skeleton file for object type X. Only
                                   certain objects are currently supported.
  -up[f] SPECFILE|NAME             Create or update an App/SP pair from a given configuration
                                   file or with a specified name; use the 'f' option to
                                   suppress the confirmation prompt. Specfile support
                                   currently has limited functionality.
  -rm[f] NAME|ID                   Delete an existing App/SP pair by displayName or App ID
  -rn[f] NAME|ID NEWNAME           Rename an App/SP pair with the given NAME/ID to NEWNAME
  -apas ID SECRET_NAME [EXPIRY]    Add a secret to an App with the given ID; optional expiry
                                   date (YYYY-MM-DD) or in X number of days
  -aprs[f] ID SECRET_ID            Remove a secret from an App with the given ID
  -spas ID SECRET_NAME [EXPIRY]    Add a secret to an SP with the given ID; optional expiry
                                   date (YYYY-MM-DD) or in X number of days
  -sprs[f] ID SECRET_ID            Remove a secret from an SP with the given ID

Other Options
  -id                              Display the currently configured login values
  -id TenantId Username            Set up user credentials for interactive login
  -id TenantId ClientId Secret     Configure ID for automated login
  -tx                              Delete the current configured login values and token
  -xx                              Delete ALL cache local files
  -Xx                              Delete X object local file cache
  -uuid                            Generate a random UUID
  -?, -h, --help                   Display the full list of options
```

### pman

Utility `pman` (see <https://que.one/scripts>) is a poor man's REST API Postman BASH script, which leverages `azm`'s `-tmg` and `-taz` arguments to get the current user's token to make other generic REST API calls against those 2 Microsoft APIs.


### Token Sample Code

Included in this repo are examples of how to get a Microsoft token with 3 different languages. The token can be for any API. The examples use Docker Compose.
1. [Python Example](https://github.com/git719/azm/tree/main/token-python)
2. [PowerShell Example](https://github.com/git719/azm/tree/main/token-powershell)
3. [Node Example](https://github.com/git719/azm/tree/main/token-node)


### Container

There is also a Docker Compose file and a Dockerfile for an example of how to use this program within a container.


### To-Do and Known Issues

The program is stable enough to be relied on as a small utility, but there are a number of little niggly things that could be improved. Will put a list together at some point.

At any rate, no matter how stable any code is, it is always worth remembering computer scientist [Tony Hoare](https://en.wikipedia.org/wiki/Tony_Hoare)'s famous quote:
> "Inside every large program is a small program struggling to get out."


### Coding Philosophy and Feedback
The primary goal of this utility is to serve as a study aid for coding Azure utilities in Go, as well as to serve as a quick _Swiss Army knife* utility for managin tenant IAM objects. If you look through the code I think you will find that is relatively straightforward. There is a deliberate effor to keep the code as clear as possible, and simple to understand and maintain.

Note that the bulk of the code is actually in the [maz](https://github.com/queone/azm/blob/main/pkg/maz/README.md) library, and other packages. Please visit that repo for more info.

This utility along with the required libraries are obviously very useful to me. I don't think I'm going to formalize the way to contribute to this project, but if you find it useful and have some improvement suggestion please let me know. Anyway, this is published as an open source project, so feel free to clone and use on your own, with proper attributions.



# Password Expiry Reporting
The `azm -apr` option allows reporting of password expiries for both Apps and SPs combined.

```bash
  -apr[c] [DAYS]                   Password expiry report for Apps/SPs; CSV optional; limit by DAYS$ pwrep -ap
```

The reporting is printed in 2 different formats: in regular table text as shown below, or in a [Comma-separated_values (CSV)](https://en.wikipedia.org/wiki/Comma-separated_values) optional format, which can be redirected to a file for further processing: 

```bash
azm -apr
TYPE   NAME                  CLIENT_ID                              SECRET_ID                              SECRET_NAME      EXPIRY
ap     localtest-sp          f706dd63-a9d5-4ba2-b57a-476225d5f23b   42f0558c-bf40-4bc3-bc60-03003932d07d   MyName2          2026-01-01 00:00
ap     sp-validator          20726181-d443-426e-a07d-6e13f592cc57   a9a775c7-aaa1-47cc-ac0f-edad9749a4d9   Initial          2024-11-24 16:27
ap     tf-az-sp00            43e6a637-587f-49bf-b4f1-ae5473d2b9b4   bb5b4b41-4cea-4949-bbf6-086cf8e1605b   today            2023-01-23 04:59
ap     tf-az-sp00            43e6a637-587f-49bf-b4f1-ae5473d2b9b4   5e77300b-0d36-4f81-889d-30ec4818423c   Joe's Test       2023-04-23 00:30
ap     tf-az-sp00            43e6a637-587f-49bf-b4f1-ae5473d2b9b4   673cb88c-4845-4798-b980-5dc3480b7feb   Initial          2024-10-27 22:43
sp     sp_site_extension     5c6daa9d-27c6-4b5b-9f76-c1ef09af406e   a7775243-61e7-452b-9f74-3a236d2f2625                    2025-09-25 02:06
sp     sp_site_reader        ce882285-1954-4b07-a38e-615bd0e931f1   07db8e95-0375-4e1b-8a99-f5aec348fc8e   2nd_secret       2024-05-19 00:00
sp     sp_site_reader        ce882285-1954-4b07-a38e-615bd0e931f1   bfc7f9fa-57ba-4785-b08f-1bec4f9aef98   new-secret       2024-10-01 15:45
```
- Already expired secrets are highligted in <span style="color:red">red color text</span> .
