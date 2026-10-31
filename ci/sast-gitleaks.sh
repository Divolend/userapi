#!/usr/bin/env bash
# Поиск секретов (пароли, токены, ключи) в исходниках. Любая находка -> билд падает.
cd "$(dirname "$0")/.." || exit 2
. ci/sast-lib.sh

rm -rf "$REPORTS_DIR"   # первый шаг анализа: убираем отчёты прошлой сборки

echo "== gitleaks: поиск секретов"
sast_run "${GITLEAKS_IMAGE:-zricethezav/gitleaks:latest}" \
  /tmp/gitleaks.json:gitleaks.json \
  dir /src --no-banner --redact -v \
  --report-format=json --report-path=/tmp/gitleaks.json --exit-code=1
rc=$?

[ $rc -eq 0 ] || fail_build gitleaks "gitleaks: найдены секреты в коде (код $rc)"
echo "gitleaks: секретов не найдено"