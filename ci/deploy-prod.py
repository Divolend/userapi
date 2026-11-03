#!/usr/bin/env python3
"""Deploy the immutable app image against the existing native PROD database."""
import os
from pathlib import Path
import secrets
import sys
import time
import urllib.request
import urllib.error
from dbctl import DB, run, NATIVE

tag = sys.argv[1]
if not tag.startswith('prod-') or not tag[5:].isdigit():
    raise SystemExit('Expected immutable prod-N image tag')
secret_file = Path.home() / 'userapi-production.env'
if not secret_file.exists():
    # A local-only runtime role with table DML rights, no DDL or administrative privileges.
    password = secrets.token_hex(32)
    sql = "CREATE ROLE userapi_runtime LOGIN PASSWORD '" + password + "' NOSUPERUSER NOCREATEDB NOCREATEROLE NOREPLICATION;\n"
    sql += 'GRANT CONNECT ON DATABASE rest_api_db TO userapi_runtime;\nGRANT USAGE ON SCHEMA public TO userapi_runtime;\nGRANT SELECT,INSERT,UPDATE,DELETE ON public.users TO userapi_runtime;\nGRANT USAGE,SELECT ON SEQUENCE public.users_id_seq TO userapi_runtime;\n'
    run(NATIVE + ['psql','-X','-v','ON_ERROR_STOP=1','-d','rest_api_db'],sql.encode())
    with secret_file.open('x') as f:
        os.chmod(secret_file,0o600)
        f.write('DB_HOST=127.0.0.1\nDB_PORT=5432\nDB_NAME=rest_api_db\nDB_USER=userapi_runtime\nDB_PASSWORD=' + password + '\n')
    print('Created restricted local runtime role; credentials saved only in mode 0600 environment file')
root = Path(__file__).resolve().parent.parent
os.environ['APP_TAG'] = tag
run(['sudo','-n','docker','pull','derbara/userapi:' + tag])
# Pass tag explicitly through sudo; do not pass the secret environment file contents.
run(['sudo','-n','env','APP_TAG=' + tag,'docker','compose','-f',str(root / 'docker-compose.prod.yaml'),'up','-d'])
for i in range(30):
    try:
        # ID 0 should return 404 while proving DB-backed requests can complete.
        urllib.request.urlopen('http://127.0.0.1:8080/users/0',timeout=2)
    except urllib.error.HTTPError as e:
        if e.code == 404:
            print('PROD deployed:',tag,'; DB-backed API responds HTTP 404 for missing ID 0')
            break
    except (OSError,TimeoutError):
        pass
    time.sleep(1)
else:
    raise SystemExit('PROD API health check failed')
print(run(['sudo','-n','docker','ps','--filter','name=userapi_prod_app','--format','{{.Names}} {{.Image}} {{.Status}}']).decode())
