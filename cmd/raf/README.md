# Resource Assignment Filename Generator (raf)
A Go utility that generates standardized filenames for Azure role assignment YAML specifications.

## Features
- Generates consistent filenames for role assignments
- Handles different Azure scope types (management groups, subscriptions)
- Sanitizes names to lowercase with hyphens
- Prevents duplicate filenames
- Lightweight with no external dependencies

## Installation and Usage
1. Compile with:
```bash
git clone github.com/queone/azm
cd cmd/raf
go build -o raf raf.go
# copy result binary to somewhere in your PATH
```

Then run as:
```bash
export AZ_TOKEN="your_azure_access_token"
raf <input-yaml-file>
```

Example:
```bash
raf assignment.yaml
# Output: sub-prod-01_jane-doe_contributor.yaml
```

2. You can also just run as a Go script with:
```bash
git clone github.com/queone/azm
cd cmd/raf
go run raf.go <input-yaml-file>
```

3. Or run the Python script with:
```bash
python3 raf.py <input-yaml-file>
```

## Input Requirements
```bash
properties:
  roleDefinitionId: "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"  # Role ID
  principalId: "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"       # User/Group/SP ID
  scope: "/subscriptions/xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"  # Scope path
```

## Specfile Name Format
The utility generates standardized filenames following this strict format:

`<scope>__<principal>__<role>.yaml`

### Naming Rules
1. **File Extension**: Always ends with `.yaml`
2. **Case**: Entire filename is lowercase
3. **Structure**: Three components separated by underscores (`_`)
4. **Uniqueness**: Errors if filename exists

### Component Breakdown

| Component  | Source | Example | Notes |
|------------|--------|---------|-------|
| **Scope** | `properties.scope` | `mg-root` | Root management groups |
|            |        | `sub-prod-01` | Subscription scopes |
|            |        | `custom-scope` | Other scopes (sanitized) |
| **Principal** | `properties.principalId` | `jane-doe` | User/group/service principal name |
| **Role** | `properties.roleDefinitionId` | `contributor` | Role definition name |

### Scope Handling Logic
```text
if scope is root management group → "mg-root"
else if scope starts with "/subscriptions/" → subscription name
else → sanitized scope path
```

### Name Sanitization
All components are:
- Converted to lowercase
- Spaces replaced with hyphens
- Special characters removed
- Trimmed to valid filename characters
This format ensures consistent, predictable naming across all role assignment specifications.

Key improvements:
1. Structured the information in clear sections
2. Added a visual table for component breakdown
3. Included explicit scope handling logic
4. Specified sanitization rules
5. Maintained all your original requirements
6. Made it more scannable
7. Added examples for each component type

The formatting uses standard Markdown that will render nicely on GitHub/GitLab. Would you like me to adjust any part of this structure?
