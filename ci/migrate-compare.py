#!/usr/bin/env python3
"""Compare a live TEST/STAGE reference with target before and after versioned migration."""
import argparse
import base64
import difflib
import hashlib
import json
from pathlib import Path
from dbctl import DB, normalize

p = argparse.ArgumentParser()
p.add_argument('environment', choices=['stage', 'prod'])
p.add_argument('--reference', required=True, help='Base64 normalized live reference from previous SSH step')
p.add_argument('--version', required=True, type=int)
p.add_argument('--build', required=True)
a = p.parse_args()
if not a.build.isdigit():
    p.error('numeric build ID required')
root = Path(__file__).resolve().parent.parent
reference = base64.b64decode(a.reference, validate=True)
if reference != normalize((root / 'database/schema/current.sql').read_bytes()):
    raise SystemExit('Live reference does not match committed schema; refuse migration')
files = sorted((root / 'migrations').glob('*.up.sql'))
if max(int(f.name.split('_')[0]) for f in files) != a.version:
    raise SystemExit('Reference version does not match this release')
db = DB(a.environment)
folder = Path.home() / 'userapi-lab8-evidence' / a.build
folder.mkdir(parents=True, exist_ok=True)
before = normalize(db.dump())
(folder / 'reference.sql').write_bytes(reference)
(folder / 'before.sql').write_bytes(before)
diff = ''.join(difflib.unified_diff(before.decode().splitlines(True), reference.decode().splitlines(True), fromfile=a.environment + '-before', tofile='reference'))
(folder / 'before.diff').write_text(diff)
ledger = db.sql("SELECT to_regclass('public.schema_migrations') IS NOT NULL").strip() == b't'
if ledger:
    old_version, dirty = db.sql('SELECT version,dirty FROM public.schema_migrations').decode().strip().split('|')
    if dirty != 'f': raise SystemExit('Dirty migration state; refuse migration')
    old_version = int(old_version)
else:
    old_version = 1
snapshot = 'current.sql' if old_version == a.version else ('baseline.sql' if old_version == 1 else 'v' + str(old_version) + '.sql')
if before != normalize((root / 'database/schema' / snapshot).read_bytes()):
    raise SystemExit('Unexpected schema drift at target recorded version; refuse migration before DDL')
print('PRE-MIGRATION SCHEMA COMPARISON', flush=True)
print(diff if diff else 'Schemas already identical', flush=True)
# Fingerprint all existing user columns, so new columns cannot hide changes to old data.
columns = db.sql("SELECT string_agg(quote_ident(column_name),',' ORDER BY ordinal_position) FROM information_schema.columns WHERE table_schema='public' AND table_name='users'").decode().strip()
query = 'SELECT count(*),md5(COALESCE(jsonb_agg(to_jsonb(t) ORDER BY id)::text,\'[]\')) FROM (SELECT ' + columns + ' FROM public.users) t'
old_rows = db.sql(query).decode().strip()
if a.environment == 'prod':
    # Backup is a separate preceding pipeline step; require its release-specific marker.
    marker = folder / 'backup.json'
    info = json.loads(marker.read_text())
    backup = Path(info['path'])
    if hashlib.sha256(backup.read_bytes()).hexdigest() != info['sha256']:
        raise SystemExit('Backup missing or checksum mismatch; refuse PROD migration')
    print('Release-specific backup verified before database changes', flush=True)
db.migrate(a.version)
after = normalize(db.dump())
(folder / 'after.sql').write_bytes(after)
if after != reference:
    print(''.join(difflib.unified_diff(reference.decode().splitlines(True), after.decode().splitlines(True))))
    raise SystemExit('Schemas still differ after migration; application deployment blocked')
new_rows = db.sql(query).decode().strip()
if old_rows != new_rows:
    raise SystemExit('Pre-existing user data fingerprint changed; application deployment blocked')
print('POST-MIGRATION: SCHEMAS IDENTICAL', flush=True)
print('Reference SHA256:', hashlib.sha256(reference).hexdigest(), flush=True)
print('Target SHA256:   ', hashlib.sha256(after).hexdigest(), flush=True)
print('Existing users count|fingerprint preserved:', new_rows, flush=True)
(folder / 'result.json').write_text(json.dumps({'environment':a.environment,'version':a.version,'schemas_identical':True,'sha256':hashlib.sha256(after).hexdigest(),'existing_users_preserved':True,'users_fingerprint':new_rows},indent=2))
