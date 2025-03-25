import os
import re
import sys
import json
import yaml
import argparse
import requests
from pathlib import Path
from typing import Dict, Any

class RoleAssignment:
    def __init__(self, data: Dict[str, Any]):
        self.properties = data.get('properties', {})
        self.principal_id = self.properties.get('principalId', '')
        self.role_definition_id = self.properties.get('roleDefinitionId', '')
        self.scope = self.properties.get('scope', '')

def sanitize_name(name: str) -> str:
    name = name.strip().lower()
    return name.replace(' ', '-')

def get_from_graph(url: str, token: str) -> str:
    headers = {
        'Authorization': f'Bearer {token}',
        'ConsistencyLevel': 'eventual'
    }
    
    try:
        response = requests.get(url, headers=headers)
        response.raise_for_status()
        data = response.json()
        
        # Try displayName first, then userPrincipalName, then mail
        display_name = data.get('displayName', '')
        if display_name:
            return sanitize_name(display_name)
        
        user_principal_name = data.get('userPrincipalName', '')
        if user_principal_name:
            return sanitize_name(user_principal_name.split('@')[0])
        
        mail = data.get('mail', '')
        if mail:
            return sanitize_name(mail.split('@')[0])
            
    except (requests.RequestException, json.JSONDecodeError):
        pass
    
    return 'UNKNOWN'

def get_principal_name(target_id: str) -> str:
    token = os.getenv('MG_TOKEN')
    if not token:
        return 'UNKNOWN'
    
    # Try user endpoint first
    user_url = f'https://graph.microsoft.com/v1.0/users/{target_id}'
    if (name := get_from_graph(user_url, token)) != 'UNKNOWN':
        return name
    
    # Try group endpoint next
    group_url = f'https://graph.microsoft.com/v1.0/groups/{target_id}'
    if (name := get_from_graph(group_url, token)) != 'UNKNOWN':
        return name
    
    # Try service principal last
    sp_url = f'https://graph.microsoft.com/v1.0/servicePrincipals/{target_id}'
    if (name := get_from_graph(sp_url, token)) != 'UNKNOWN':
        return name
    
    return 'UNKNOWN'

def get_subscription_name(target_id: str) -> str:
    token = os.getenv('AZ_TOKEN')
    if not token:
        return 'UNKNOWN'
    
    url = f'https://management.azure.com/subscriptions/{target_id}?api-version=2016-06-01'
    headers = {'Authorization': f'Bearer {token}'}
    
    try:
        response = requests.get(url, headers=headers)
        response.raise_for_status()
        data = response.json()
        display_name = data.get('displayName', '')
        return sanitize_name(display_name) if display_name else 'UNKNOWN'
    except (requests.RequestException, json.JSONDecodeError):
        return 'UNKNOWN'

def get_role_definition_name(target_id: str) -> str:
    token = os.getenv('AZ_TOKEN')
    if not token:
        return 'UNKNOWN'
    
    url = f'https://management.azure.com/providers/Microsoft.Authorization/roleDefinitions/{target_id}?api-version=2022-04-01'
    headers = {'Authorization': f'Bearer {token}'}
    
    try:
        response = requests.get(url, headers=headers)
        response.raise_for_status()
        data = response.json()
        role_name = data.get('properties', {}).get('roleName', '')
        return sanitize_name(role_name) if role_name else 'UNKNOWN'
    except (requests.RequestException, json.JSONDecodeError):
        return 'UNKNOWN'

def is_root_management_group(scope: str) -> bool:
    pattern = r'^/providers/Microsoft\.Management/managementGroups/[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$'
    return bool(re.match(pattern, scope))

def sanitize_part(s: str) -> str:
    s = s.strip()
    result = []
    prev_hyphen = False
    
    for char in s:
        if char.isalnum():
            result.append(char.lower())
            prev_hyphen = False
        elif char.isspace() or not char.isprintable():
            if not prev_hyphen:
                result.append('-')
                prev_hyphen = True
        else:
            if not prev_hyphen:
                result.append('-')
                prev_hyphen = True
    
    sanitized = ''.join(result).strip('-')
    return sanitized if sanitized else 'unknown'

def get_scope_name(scope: str) -> str:
    if is_root_management_group(scope):
        return 'mg-root'
    elif scope.startswith('/subscriptions/'):
        sub_id = Path(scope).name
        return sanitize_part(get_subscription_name(sub_id))
    else:
        return sanitize_part(scope)

def generate_filename(ra: RoleAssignment) -> str:
    part1 = get_scope_name(ra.scope)
    if not part1:
        raise ValueError('invalid scope')
    
    part2 = sanitize_part(get_principal_name(ra.principal_id))
    part3 = sanitize_part(get_role_definition_name(ra.role_definition_id))
    
    if not part2 or not part3:
        raise ValueError('invalid principal or role name')
    
    filename = f'{part1}_{part2}_{part3}.yaml'
    
    if os.path.exists(filename):
        raise FileExistsError(f'file "{filename}" already exists')
    
    return filename

def main():
    parser = argparse.ArgumentParser()
    parser.add_argument('input_file')
    args = parser.parse_args()
    
    if not args.input_file:
        print('Error: missing input file argument', file=sys.stderr)
        sys.exit(1)
    
    try:
        with open(args.input_file, 'r') as f:
            yaml_data = yaml.safe_load(f)
        
        ra = RoleAssignment(yaml_data)
        filename = generate_filename(ra)
        print(filename)
    except Exception as e:
        print(f'Error: {e}', file=sys.stderr)
        sys.exit(1)

if __name__ == '__main__':
    main()
