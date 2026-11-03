#!/usr/bin/env bash
# Run with bash ci/test.sh unit|integration. All containers are disposable.
set -euo pipefail
cd "$(dirname "$0")/.."
kind="${1:?use unit or integration}"
case "$kind" in unit|integration) ;; *) echo 'Unknown test suite'; exit 2;; esac
mkdir -p reports
work_id="lab7-${kind}-${BUILD_ID:-$$}"
runner="${work_id}-go"
db="${work_id}-db"
network="${work_id}-net"
cleanup() {
 docker rm -f "$runner" >/dev/null 2>&1 || true
 if [ "$kind" = integration ]; then
  docker rm -fv "$db" >/dev/null 2>&1 || true
  docker network rm "$network" >/dev/null 2>&1 || true
 fi
}
trap cleanup EXIT INT TERM
args=()
if [ "$kind" = integration ]; then
 docker network create "$network" >/dev/null
 # Password is generated per run, used only on the isolated Docker network.
 password=$(cat /proc/sys/kernel/random/uuid)
 docker run -d --name "$db" --network "$network" --network-alias db \
  -e POSTGRES_DB=userapi_lab7_test -e POSTGRES_USER=lab7 -e POSTGRES_PASSWORD="$password" \
  postgres:16 >/dev/null
 ready=0
 for ((i=0;i<60;i++)); do
  if docker exec "$db" pg_isready -U lab7 -d userapi_lab7_test >/dev/null 2>&1; then ready=1;break;fi
  sleep 1
 done
 [ "$ready" = 1 ] || { docker logs "$db";exit 1; }
 docker exec -i "$db" psql -v ON_ERROR_STOP=1 -U lab7 -d userapi_lab7_test < migrations/000001_create_users_table.up.sql
 args=(--network "$network" -e USERAPI_TEST_DATABASE=1 -e DB_HOST=db -e DB_PORT=5432 -e DB_USER=lab7 -e DB_PASSWORD="$password" -e DB_NAME=userapi_lab7_test)
fi
# Named caches avoid recompiling Gin on every build and work with a containerized agent.
docker create --name "$runner" "${args[@]}" -w /src \
 -v lab7-go-mod:/go/pkg/mod -v lab7-go-build:/root/.cache/go-build \
 -e GOMAXPROCS=2 golang:1.26-alpine sh -c '
 set -eu
 mkdir -p /tmp/results
 if [ "$1" = integration ]; then
  go test -p 2 -tags=integration -run TestPostgresIntegration -count=1 -timeout=180s -coverpkg=./internal/... -coverprofile=/tmp/results/integration.out -v ./internal/handler > /tmp/results/integration.log 2>&1
 else
  go test -p 2 -coverpkg=./internal/... -count=1 -timeout=180s -coverprofile=/tmp/results/unit.out -v ./internal/... > /tmp/results/unit.log 2>&1
 fi
 go tool cover -func=/tmp/results/$1.out > /tmp/results/$1-coverage.txt
 go tool cover -html=/tmp/results/$1.out -o /tmp/results/$1-coverage.html
 ' sh "$kind" >/dev/null
docker cp . "$runner:/src"
set +e
docker start -a "$runner"
rc=$?
set -e
docker cp "$runner:/tmp/results/." reports/
cat "reports/$kind.log"
if [ "$rc" -ne 0 ]; then
 echo "##teamcity[buildProblem description='$kind tests failed' identity='lab7-$kind']"
 exit "$rc"
fi
cat "reports/$kind-coverage.txt"
printf '%s tests passed\n' "$kind"
