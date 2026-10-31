#!/usr/bin/env bash
# Анализатор качества кода golangci-lint. Находка уровня BLOCKER (см. .golangci.yml) -> билд падает.
cd "$(dirname "$0")/.." || exit 2
. ci/sast-lib.sh
IMAGE="${GOLANGCI_IMAGE:-golangci/golangci-lint:latest}"

echo "== golangci-lint: анализ кода"
# --issues-exit-code=0: сами решаем, какие находки считать ошибкой сборки
sast_run "$IMAGE" /tmp/golangci.json:golangci.json,/tmp/golangci.html:golangci.html \
  golangci-lint run --issues-exit-code=0 \
  --output.text.path=stdout \
  --output.code-climate.path=/tmp/golangci.json \
  --output.html.path=/tmp/golangci.html ./...
rc=$?
[ $rc -eq 0 ] || fail_build golangci-run "golangci-lint завершился с ошибкой (код $rc)"
[ -f "$REPORTS_DIR/golangci.json" ] || fail_build golangci-report "golangci-lint: отчёт не создан"

blockers=$(grep -o '"severity":"blocker"' "$REPORTS_DIR/golangci.json" | wc -l)
echo "golangci-lint: находок уровня BLOCKER: $blockers"
[ "$blockers" -eq 0 ] || fail_build golangci-blocker "golangci-lint: найдено BLOCKER: $blockers"