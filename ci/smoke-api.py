#!/usr/bin/env python3
"""Explicitly approved lab smoke test; delete only the record created by this run."""
import argparse
import json
import re
import urllib.error
import urllib.request
import uuid
import time

p = argparse.ArgumentParser()
p.add_argument('--base-url', default='http://127.0.0.1:8080')
p.add_argument('--city', action='store_true')
a = p.parse_args()
base = a.base_url.rstrip('/') + '/users'

def request(method, path, payload=None, expected=200):
    req = urllib.request.Request(base + path, data=json.dumps(payload).encode() if payload else None,
        method=method, headers={'Content-Type':'application/json','Authorization':'lab8'})
    try:
        with urllib.request.urlopen(req, timeout=10) as response:
            code, text = response.status, response.read().decode()
    except urllib.error.HTTPError as response:
        code, text = response.code, response.read().decode()
    print(method, path, 'HTTP', code, text.strip(), flush=True)
    if code != expected:
        raise RuntimeError('Unexpected HTTP status')
    return text

# Wait for a DB-backed read before any POST, avoiding retries of a write request.
for attempt in range(30):
    try:
        with urllib.request.urlopen(a.base_url.rstrip('/') + '/users/0', timeout=2) as response:
            pass
    except urllib.error.HTTPError as response:
        if response.code == 404: break
    except (OSError, TimeoutError):
        pass
    time.sleep(1)
else:
    raise SystemExit('API did not become ready for smoke test')

uid = None
try:
    name = 'Lab8Check-' + uuid.uuid4().hex[:10]
    payload = {'name':name,'age':24,'email':'check@example.test'}
    if a.city: payload['city'] = 'Moscow'
    body = request('POST', '', payload)
    uid = re.search(r'ID - (\d+)', body).group(1)
    assert 'check@example.test' in body
    if a.city: assert 'Moscow' in body
    body = request('GET','/' + uid)
    assert 'check@example.test' in body
    update = {'email':'check+updated@example.test'}
    if a.city: update['city'] = 'Kazan'
    body = request('PATCH','/' + uid, update)
    assert 'check+updated@example.test' in body
    if a.city: assert 'Kazan' in body
    body = request('GET','/' + uid)
    assert 'check+updated@example.test' in body
    if a.city: assert 'Kazan' in body
    print('PASS: new fields persisted through create/read/update', flush=True)
finally:
    if uid is not None:
        request('DELETE','/' + uid)
        request('GET','/' + uid, expected=404)
        print('CLEANUP VERIFIED: only temporary user created by this run was deleted', flush=True)
