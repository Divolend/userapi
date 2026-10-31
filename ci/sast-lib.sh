#!/usr/bin/env bash
# Общие функции для шагов статического анализа в TeamCity.
#
# Анализатор запускается в одноразовом контейнере. Исходники копируются внутрь
# командой docker cp, отчёты копируются обратно. Так не нужны «-v <путь>:/src»:
# агент TeamCity сам работает в контейнере, и путь его рабочей папки на хосте
# отличается от пути внутри агента.

REPORTS_DIR="${REPORTS_DIR:-reports}"

# sast_run <образ> <отчёты|-> <аргументы контейнера...>
#   отчёты: "путь_в_контейнере:имя_на_агенте[,путь:имя...]" или "-"
# Возвращает код завершения анализатора (125 - не удалось запустить контейнер).
sast_run() {
  local image="$1" reports="$2"; shift 2
  local cid rc pair
  mkdir -p "$REPORTS_DIR"

  cid=$(docker create -u 0 -w /src "$image" "$@") || {
    echo "Не удалось создать контейнер из образа $image"; return 125; }

  if ! docker cp "$PWD/." "$cid:/src" >/dev/null; then
    echo "Не удалось скопировать исходники в контейнер"
    docker rm -f "$cid" >/dev/null 2>&1
    return 125
  fi

  docker start -a "$cid"
  rc=$?

  if [ "$reports" != "-" ]; then
    local IFS=','
    for pair in $reports; do
      docker cp "$cid:${pair%%:*}" "$REPORTS_DIR/${pair##*:}" >/dev/null 2>&1 \
        || echo "Отчёт ${pair%%:*} не создан"
    done
  fi

  docker rm -f "$cid" >/dev/null 2>&1
  return $rc
}

# fail_build <идентификатор> <сообщение> - помечает сборку как упавшую
fail_build() {
  echo "##teamcity[buildProblem description='$2' identity='$1']"
  exit 1
}