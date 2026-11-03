#!/usr/bin/env bash
set -uo pipefail
cd "$(dirname "$0")/.."
source ci/sast-lib.sh
sast_run semgrep/semgrep:1.183.0 /tmp/sql-security.json:sql-security.json semgrep scan --config ci/sql-security.yml --metrics=off --disable-version-check --no-git-ignore --include '*.sql' --exclude reports --error --json-output=/tmp/sql-security.json .
rc=$?
if [ "$rc" -ne 0 ]; then fail_build lab8-sql "SQL security scan failed (exit $rc)"; fi
echo 'SQL security scan passed'
