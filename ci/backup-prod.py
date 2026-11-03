#!/usr/bin/env python3
"""Back up existing PROD data and prove a restore before allowing migration."""
import hashlib
import json
from pathlib import Path
import subprocess
import sys
from dbctl import DB, NATIVE, run

build = sys.argv[1]
if not build.isdigit():
    raise SystemExit('numeric build ID required')
db = DB('prod')
raw_before = db.sql("SELECT count(*),md5(COALESCE(jsonb_agg(to_jsonb(u) ORDER BY id)::text,'[]')) FROM public.users u").decode().strip()
path = db.backup(Path.home() / 'userapi-backups')
# Restore into a distinct scratch database, never into the live database.
name = 'userapi_lab8_restore_' + build
run(NATIVE + ['createdb', name])
try:
    run(NATIVE + ['pg_restore', '--exit-on-error', '--no-owner', '--no-privileges', '-d', name], path.read_bytes())
    restored = run(NATIVE + ['psql', '-X', '-At', '-d', name, '-c', "SELECT count(*),md5(COALESCE(jsonb_agg(to_jsonb(u) ORDER BY id)::text,'[]')) FROM public.users u"]).decode().strip()
    if restored != raw_before:
        raise RuntimeError('Restored users do not match pre-backup users fingerprint')
    print('RESTORE VERIFIED in separate database:', name, flush=True)
    print('Original and restored users count|fingerprint:', restored, flush=True)
finally:
    # Only the uniquely named scratch restore database created by this run is removed.
    run(NATIVE + ['dropdb', name])
folder = Path.home() / 'userapi-lab8-evidence' / build
folder.mkdir(parents=True,exist_ok=True)
(folder / 'backup.json').write_text(json.dumps({'path':str(path),'bytes':path.stat().st_size,'sha256':hashlib.sha256(path.read_bytes()).hexdigest(),'restore_verified':True,'users':restored},indent=2))
print('Backup marker recorded for build', build, flush=True)
