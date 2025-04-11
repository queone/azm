#!/usr/bin/env python3
# jah.py
# This script was developed as a proof-of-concept to then develop the code in token_decode.go.
# This is incomplete but could be developed further as a Python alternative.

import sys
import json
import base64
import requests
from jose import jwt
from jose.utils import base64url_decode
from cryptography.hazmat.primitives.asymmetric import rsa
from cryptography.hazmat.backends import default_backend
from cryptography.x509 import load_der_x509_certificate
from urllib.parse import urlparse

BLUE       = Blu  = "\033[1;34m"
CYAN       = Cya  = "\033[1;36m"
GREEN      = Grn  = "\033[0;32m"
GRAY       = Gra  = "\033[1;30m"
RED        = Red  = "\033[1;31m"
YELLOW     = Yel  = "\033[1;33m"
RESET      = Rst  = "\033[0m"

def base64url_decode_str(data):
    padding = '=' * ((4 - len(data) % 4) % 4)
    return base64.urlsafe_b64decode(data + padding)

def extract_token_parts(token):
    header_b64, payload_b64, signature_b64 = token.split('.')
    header = json.loads(base64url_decode_str(header_b64).decode('utf-8'))
    payload = json.loads(base64url_decode_str(payload_b64).decode('utf-8'))
    return header, payload, signature_b64

def get_jwks_uri(issuer):
    if 'sts.windows.net' in issuer:
        return f"{issuer.rstrip('/')}/discovery/v2.0/keys"
    raise ValueError("Unsupported issuer")

def load_public_key(key_entry):
    try:
        if "x5c" in key_entry:
            cert_der = base64.b64decode(key_entry["x5c"][0])
            cert = load_der_x509_certificate(cert_der, default_backend())
            return cert.public_key()
        else:
            n = int.from_bytes(base64url_decode(key_entry['n'].encode()), byteorder='big')
            e = int.from_bytes(base64url_decode(key_entry['e'].encode()), byteorder='big')
            return rsa.RSAPublicNumbers(e, n).public_key(default_backend())
    except Exception as e:
        print(f"{Red}Error loading public key: {e}{Rst}")
        return None

def validate_issuer_claim(payload):
    tid = payload.get("tid")
    iss = payload.get("iss")
    if not tid or not iss:
        print("Missing 'tid' or 'iss' in token payload")
        return False
    if "{tenantid}" in iss.lower():
        expected_iss = iss.lower().replace("{tenantid}", tid.lower())
        if expected_iss != iss.lower():
            print("Issuer contains templated {tenantid} but does not match token tid")
            return False
    if tid.lower() not in iss.lower():
        print("Issuer does not contain tid")
        return False
    parsed = urlparse(iss)
    segments = parsed.path.strip("/").split("/")
    if len(segments) < 1 or segments[0].lower() != tid.lower():
        print("Issuer URI structure invalid — tid mismatch")
        return False
    return True

def print_token_components(header, claims, signature_b64):
    print(f"{Blu}header{Rst}:")
    for k, v in header.items():
        print(f"  {Blu}{k}{Rst}: {Grn}{v}{Rst}")
    print(f"{Blu}claims{Rst}:")
    for k, v in claims.items():
        print(f"  {Blu}{k}{Rst}: {Grn}{v}{Rst}")
    print(f"{Blu}signature{Rst}: {Grn}{signature_b64}{Rst}  {Gra}# In base64 format{Rst}")

def main(token):
    header, payload, signature_b64 = extract_token_parts(token)

    kid = header.get("kid")
    aud = payload.get("aud")
    iss = payload.get("iss")

    print("Validating token...")
    jwks_uri = get_jwks_uri(iss)
    print(f"JWKS URI: {jwks_uri}")

    jwks = requests.get(jwks_uri).json()
    keys = jwks.get("keys", [])
    print(f"Available KIDs: {[k['kid'] for k in keys]}")

    key_entry = next((k for k in keys if k["kid"] == kid), None)
    if not key_entry:
        print(f"{Red}Public key not found for KID{Rst}")
        return

    public_key = load_public_key(key_entry)
    if not public_key:
        return

    if aud == "https://graph.microsoft.com":
        print(f"Detected MS Graph token - skipping formal signature validation.\n"
              "Performing MS Graph issuer/tid structural validation.")
        print_token_components(header, payload, signature_b64)
        print(f"{Blu}status{Rst}:")
        print(f"{Gra}  # See https://learn.microsoft.com/en-us/entra/identity-platform/access-tokens{Rst}")
        if validate_issuer_claim(payload):
            print(f"  {Blu}valid{Rst}: {Grn}true{Rst}  {Gra}# PASSED: Issuer structure valid and tid match{Rst}")
        else:
            print(f"  {Blu}valid{Rst}: {Red}false{Rst}  {Gra}# FAILED: Issuer URI structure invalid — tid mismatch{Rst}")
    elif aud == "https://management.azure.com":
        try:
            decoded = jwt.decode(token, public_key, algorithms=["RS256"], audience=aud, issuer=iss)
            print_token_components(header, decoded, signature_b64)
            print(f"{Blu}status{Rst}:")
            print(f"  {Blu}valid{Rst}: {Grn}true{Rst}  {Gra}# PASSED formal validation{Rst}")
        except Exception:
            print_token_components(header, payload, signature_b64)
            print(f"{Blu}status{Rst}:")
            print(f"  {Blu}valid{Rst}: {Red}false{Rst}  {Gra}# FAILED formal validation{Rst}")
    else:
        print(f"Unrecognized 'aud' audience field value: {aud}")

if __name__ == "__main__":
    if len(sys.argv) != 2:
        print(f"{Yel}Usage: jah2.py <JWT_token>{Rst}")
        sys.exit(1)
    main(sys.argv[1])
