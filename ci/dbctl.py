#!/usr/bin/env python3
"""Local host DB operations. Run on the selected VM; secrets never enter reports."""
import argparse
import hashlib
import json
import os
from pathlib import Path
import subprocess
import sys
import urllib.parse

ROOT = Path(__file__).resolve().parent.parent
NATIVE = ['sudo', '-n', '-u', 'postgres']
MIGRATE = '/usr/local/bin/userapi-migrate'


def run(args, data=None):
    return subprocess.check_output(args, input=data, stderr=subprocess.PIPE)


def normalize(data):
    lines = []
    for line in data.decode().splitlines():
        s = line.strip()
        if not s or s.startswith(('--', '\\restrict ', '\\unrestrict ', 'SET ', 'SELECT pg_catalog.set_config(')):
            continue
        lines.append(line.rstrip())
    return ('\n'.join(lines) + '\n').encode()


class DB:
    def __init__(self, env):
        self.env = env
        self.container = {'test': 'userapi_lab8_db', 'stage': 'userapi_stage_db'}.get(env)
        if self.container:
            info = json.loads(run(['sudo', '-n', 'docker', 'inspect', self.container]))[0]
            self.vars = dict(x.split('=', 1) for x in info['Config']['Env'] if '=' in x)
            self.name = self.vars['POSTGRES_DB']
            self.user = self.vars['POSTGRES_USER']
        else:
            self.name, self.user = 'rest_api_db', 'postgres'

    def command(self, tool, args, data=None):
        if self.container:
            cmd = ['sudo', '-n', 'docker', 'exec', '-i', self.container, tool]
        else:
            cmd = NATIVE + [tool]
        return run(cmd + ['-U', self.user, '-d', self.name] + args, data)

    def sql(self, sql):
        return self.command('psql', ['-X', '-v', 'ON_ERROR_STOP=1', '-Atc', sql])

    def dump(self):
        return self.command('pg_dump', ['--schema-only', '--no-owner', '--no-privileges', '-n', 'public', '--exclude-table=public.schema_migrations'])

    def migrate(self, version, adopt=False):
        if self.container:
            run(['sudo', '-n', 'docker', 'cp', MIGRATE, self.container + ':/tmp/userapi-migrate'])
            # Copy into /tmp only, never into the persistent data directory.
            run(['sudo', '-n', 'docker', 'exec', self.container, 'mkdir', '-p', '/tmp/userapi-migrations'])
            run(['sudo', '-n', 'docker', 'cp', str(ROOT / 'migrations') + '/.', self.container + ':/tmp/userapi-migrations'])
            q = urllib.parse.quote
            url = 'postgres://' + q(self.user, safe='') + ':' + q(self.vars['POSTGRES_PASSWORD'], safe='') + '@127.0.0.1:5432/' + q(self.name, safe='') + '?sslmode=disable'
            cmd = ['sudo', '-n', 'docker', 'exec', self.container, '/tmp/userapi-migrate', '-path', '/tmp/userapi-migrations', '-database', url]
        else:
            # Peer authentication as postgres through the local Unix socket.
            files = sorted((ROOT / 'migrations').glob('*.sql'))
            signature = hashlib.sha256(b''.join(f.read_bytes() for f in files)).hexdigest()[:16]
            folder = '/var/lib/userapi-migrations/' + signature
            run(['sudo', '-n', 'install', '-d', '-m', '755', folder])
            for source in files:
                run(['sudo', '-n', 'install', '-m', '644', str(source), folder + '/' + source.name])
            cmd = NATIVE + [MIGRATE, '-path', folder, '-database', 'postgres:///rest_api_db?host=/var/run/postgresql&sslmode=disable']
        if self.sql("SELECT to_regclass('public.schema_migrations') IS NOT NULL").strip() == b'f':
            if adopt:
                if normalize(self.dump()) != normalize((ROOT / 'database/schema/current.sql').read_bytes()):
                    raise RuntimeError('Refusing manual adoption: real schema differs from exported current.sql')
                run(cmd + ['force', str(version)])
                print('Manually applied schema verified against current.sql; recorded version', version, flush=True)
                return
            baseline = normalize((ROOT / 'database/schema/baseline.sql').read_bytes())
            if normalize(self.dump()) != baseline:
                raise RuntimeError('Refusing baseline: real database differs from verified version 1 schema')
            run(cmd + ['force', '1'])
            print('Verified existing baseline; recorded migrate version 1', flush=True)
        current = self.sql('SELECT version,dirty FROM public.schema_migrations').decode().strip()
        print('Before migration version|dirty:', current, flush=True)
        major, dirty = current.split('|')
        if dirty != 'f' or int(major) > version:
            raise RuntimeError('Refusing dirty state or automatic downgrade')
        if int(major) != version:
            run(cmd + ['goto', str(version)])
        print('After migration version|dirty:', self.sql('SELECT version,dirty FROM public.schema_migrations').decode().strip(), flush=True)

    def backup(self, folder):
        folder.mkdir(parents=True, exist_ok=True, mode=0o700)
        os.chmod(folder, 0o700)
        from datetime import datetime, timezone
        stamp = datetime.now(timezone.utc).strftime('%Y%m%dT%H%M%SZ')
        path = folder / (self.env + '-' + stamp + '.dump')
        raw = self.command('pg_dump', ['-Fc', '--no-owner', '--no-privileges'])
        with path.open('xb') as f:
            os.chmod(path, 0o600)
            f.write(raw)
        listing = run(NATIVE + ['pg_restore', '--list'], raw) if not self.container else run(['sudo', '-n', 'docker', 'exec', '-i', self.container, 'pg_restore', '--list'], raw)
        if b'TABLE DATA public users' not in listing:
            raise RuntimeError('Backup has no users data entry')
        digest = hashlib.sha256(raw).hexdigest()
        path.with_suffix('.dump.sha256').write_text(digest + '  ' + path.name + '\n')
        path.with_suffix('.dump.list').write_bytes(listing)
        print('Backup saved:', path, 'bytes=', len(raw), 'sha256=', digest, flush=True)
        print('Backup archive verified: TABLE DATA public users present', flush=True)
        return path


def main():
    p = argparse.ArgumentParser()
    p.add_argument('environment', choices=['test', 'stage', 'prod'])
    p.add_argument('action', choices=['dump', 'normalized', 'migrate', 'backup', 'status'])
    p.add_argument('--version', type=int)
    p.add_argument('--adopt-exported', action='store_true', help='Manual setup only: adopt schema identical to current.sql')
    args = p.parse_args()
    db = DB(args.environment)
    if args.action == 'dump': sys.stdout.buffer.write(db.dump())
    elif args.action == 'normalized': sys.stdout.buffer.write(normalize(db.dump()))
    elif args.action == 'migrate':
        if not args.version: p.error('--version required')
        db.migrate(args.version, args.adopt_exported)
    elif args.action == 'backup': db.backup(Path.home() / 'userapi-backups')
    else:
        print(db.sql('SELECT version()').decode().strip())
        print(db.sql("SELECT column_name,data_type,is_nullable,column_default FROM information_schema.columns WHERE table_schema='public' AND table_name='users' ORDER BY ordinal_position").decode().strip())
        print('users count:', db.sql('SELECT count(*) FROM users').decode().strip())


if __name__ == '__main__':
    try:
        main()
    except subprocess.CalledProcessError as error:
        # The command may contain a local connection password; do not print argv.
        sys.stderr.write(error.stderr.decode() if error.stderr else 'Database command failed\n')
        sys.exit(1)
