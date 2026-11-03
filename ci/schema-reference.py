#!/usr/bin/env python3
"""Capture the live reference schema for later SSH steps, without DB credentials."""
import base64
import hashlib
from pathlib import Path
import sys
from dbctl import DB, normalize

environment = sys.argv[1]
if environment not in ('test', 'stage'):
    raise SystemExit('Reference must be test for DEV or stage for PROD')
db = DB(environment)
actual = normalize(db.dump())
expected = normalize((Path(__file__).resolve().parent.parent / 'database/schema/current.sql').read_bytes())
if actual != expected:
    raise SystemExit('Live reference differs from schema exported in this commit; stop delivery')
version, dirty = db.sql('SELECT version,dirty FROM public.schema_migrations').decode().strip().split('|')
if dirty != 'f':
    raise SystemExit('Reference database has dirty migration state')
print('Reference environment:', environment)
print('Reference schema SHA256:', hashlib.sha256(actual).hexdigest())
print('Reference migration version:', version)
print("##teamcity[setParameter name='lab8.reference' value='" + base64.b64encode(actual).decode() + "']")
print("##teamcity[setParameter name='lab8.version' value='" + version + "']")
