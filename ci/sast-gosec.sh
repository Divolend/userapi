#!/usr/bin/env bash
# Анализатор безопасности gosec. Находка уровня HIGH (наивысший у gosec) -> билд падает.
# Уровни gosec: LOW, MEDIUM, HIGH (уровня CRITICAL нет, ему соответствует HIGH).
cd "$(dirname "$0")/.." || exit 2
. ci/sast-lib.sh
IMAGE="${GOSEC_IMAGE:-securego/gosec:latest}"

echo "== gosec: полный отчёт (все уровни)"
sast_run "$IMAGE" /tmp/gosec.html:gosec.html \
  -no-fail -fmt=html -out=/tmp/gosec.html -exclude-dir=docs ./...

echo "== gosec: проверка порога (только HIGH)"
sast_run "$IMAGE" - -severity=high -fmt=text -exclude-dir=docs ./...
rc=$?

[ $rc -eq 0 ] || fail_build gosec-high "gosec: найдены проблемы уровня HIGH (код $rc)"
echo "gosec: проблем уровня HIGH нет"