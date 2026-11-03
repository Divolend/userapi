# Автоматизированные тесты UserAPI — ЛР7

Используются стандартные пакеты Go testing и net/http/httptest. Сторонний тестовый фреймворк не требуется: есть табличные подтесты, проверки ошибок и воспроизводимый запуск go test. Unit-тесты сервиса используют repositoryStub; HTTP-сценарии проверяют обработчики совместно с сервисом и хранилищем в памяти. Интеграционный набор проходит через HTTP-обработчики, сервис, настоящий PostgreSQL-репозиторий и временную PostgreSQL 16.

## Запуск

```sh
bash ci/test.sh unit
bash ci/test.sh integration
```

Нужны Bash и доступ к Docker. В TeamCity команды запускаются агентом на TEST. В обычной SSH-сессии pkm на TEST требуется sudo. Файлы копируются docker cp, поскольку агент сам работает в контейнере. Для интеграционных тестов создаются отдельные сеть, БД userapi_lab7_test и пароль на один запуск. Порт БД не публикуется. После выполнения удаляются тестовые контейнеры, сеть и том PostgreSQL. Постоянные БД test/stage/prod не используются. Общие кеши Go — lab7-go-mod и lab7-go-build.

Без тега integration PostgreSQL-тест не компилируется и не запускается. При ручном запуске с этим тегом нужна явная переменная USERAPI_TEST_DATABASE=1 и DB_NAME=userapi_lab7_test. Любая ошибка подключения означает падение теста, а не пропуск.

Отчеты: reports/unit.log, unit.out, unit-coverage.txt, unit-coverage.html и аналогичные integration-файлы. Ошибка тестов дает ненулевой код шага и блокирует последующие push/deploy. Команда go test использует -count=1, чтобы результаты не брались из кеша. HTML показывает покрытие операторов пакетов internal; это не процент требований ТЗ.

## Сценарии

| Тест | Данные | Ожидаемый результат |
|---|---|---|
| TestCreateUser/new_user | Diana, 21; имя свободно | Create вызван, пользователь возвращен |
| TestCreateUser/duplicate_name | Diana уже существует | ошибка, Create не вызывается |
| TestCreateUser/lookup_failure | ошибка FindByName | ошибка передана вызывающему коду |
| TestCreateUser/insert_failure | ошибка Create | ошибка передана вызывающему коду |
| TestFindByID/found | id=7 найден | возвращен нужный пользователь |
| TestFindByID/missing | id=7 отсутствует | ErrUserNotFound |
| TestFindByID/storage_failure | ошибка хранилища | исходная ошибка |
| TestUpdateUser/positive_age | id=7, age=22 | возраст сохранен |
| TestUpdateUser/zero_age_rejected | age=0 | ErrInvalidAge, UpdateField не вызывается |
| TestUpdateUser/negative_age_rejected | age=-1 | ErrInvalidAge, UpdateField не вызывается |
| TestUpdateUser/rename | новое свободное имя | имя сохранено |
| TestUpdateUser/same_name_allowed | прежнее имя | обновление допускается |
| TestUpdateUser/duplicate_rejected | имя другого пользователя | ErrNameAlreadyExists, запись не меняется |
| TestUpdateUser/missing_user | отсутствующий id, новое имя | ErrUserNotFound |
| TestDeleteUser/existing | id=7 | удаленный пользователь возвращен |
| TestDeleteUser/missing | отсутствующий id | ErrUserNotFound |
| TestAuthMiddleware/missing_header | нет Authorization | 401, защищенный обработчик не вызван |
| TestAuthMiddleware/nonempty_header | непустой Authorization | защищенный обработчик вызван |
| TestHTTPScenarios/create | POST Nastya, age=22, auth | 200, имя в ответе |
| TestHTTPScenarios/public_read | GET существующего id без auth | 200, имя в ответе |
| TestHTTPScenarios/missing_read | GET /users/999 | 404 |
| TestHTTPScenarios/bad_read_ID | GET /users/abc | 400 |
| TestHTTPScenarios/create_without_auth | POST без auth | 401 |
| TestHTTPScenarios/duplicate_name | POST Diana повторно | 409 |
| TestHTTPScenarios/invalid_JSON | POST с телом { | 400 |
| TestHTTPScenarios/missing_name | POST только age | 400 |
| TestHTTPScenarios/missing_age | POST только name | 400 |
| TestHTTPScenarios/zero_age | POST age=0 | 400 |
| TestHTTPScenarios/negative_age | POST age=-1 | 400 |
| TestHTTPScenarios/update_age | PATCH age=23 | 200, новый возраст |
| TestHTTPScenarios/rename | PATCH name=Nastya | 200, новое имя |
| TestHTTPScenarios/duplicate_rename | PATCH name=Yaroslava | 409 |
| TestHTTPScenarios/invalid_update_age | PATCH age=-1 | 400 |
| TestHTTPScenarios/missing_update | PATCH /users/999 | 404 |
| TestHTTPScenarios/update_without_auth | PATCH без auth | 401 |
| TestHTTPScenarios/invalid_update_JSON | PATCH с телом { | 400 |
| TestHTTPScenarios/bad_update_ID | PATCH /users/abc | 400 |
| TestHTTPScenarios/delete | DELETE существующего id | 200, удаленный пользователь |
| TestHTTPScenarios/missing_delete | DELETE /users/999 | 404 |
| TestHTTPScenarios/delete_without_auth | DELETE без auth | 401 |
| TestHTTPScenarios/bad_delete_ID | DELETE /users/abc | 400 |
| TestPostgresIntegration/HTTP_CRUD_persisted_in_PostgreSQL | создание IntegrationUser, повтор, чтение, переименование/возраст, удаление | 200/409/200/200/200/404; реальное состояние PostgreSQL проверено |
| TestPostgresIntegration/database_constraints | повтор ConstraintUser и age=0 напрямую через репозиторий | UNIQUE и CHECK отклоняют записи |
| TestPostgresIntegration/authorization_prevents_database_write | POST UnauthorizedUser без auth | 401, в БД нет пользователя |

Всего 41 подтест в обычном наборе и 3 в PostgreSQL-наборе. Родительские Test-функции отдельно в это число не включены. Каждый HTTP-сценарий обычного набора создает собственную фикстуру Diana/21 и Yaroslava/20; при отклонении запроса дополнительно проверяется неизменность исходной Diana. Тесты с глобальным handler.UserService не запускаются параллельно, исходное значение восстанавливается t.Cleanup.

## Связь с техническим заданием

Для функциональной части ТЗ выделены 14 проверяемых критериев. Это явная декомпозиция требований для отчета, не официальный перечень преподавателя. Автоматизированы 11 из 14 (78,6%), что больше требуемых 40%; это покрытие сценариями, а не доказательство отсутствия всех возможных дефектов.

| № | Функциональный критерий | Подтверждение |
|---|---|---|
| 1 | Создание пользователя | service, HTTP, PostgreSQL CRUD |
| 2 | Чтение по идентификатору и отсутствие пользователя | FindByID, HTTP, PostgreSQL CRUD |
| 3 | Изменение имени и возраста | UpdateUser, HTTP, PostgreSQL CRUD |
| 4 | Удаление пользователя | DeleteUser, HTTP, PostgreSQL CRUD |
| 5 | Непустой Authorization для изменения | middleware и POST/PATCH/DELETE без auth |
| 6 | Публичное чтение | public_read без auth |
| 7 | Обязательные поля создания | missing_name, missing_age |
| 8 | Положительный возраст при создании/изменении | HTTP, сервис, CHECK PostgreSQL |
| 9 | Уникальность имени при создании/переименовании | HTTP, сервис, UNIQUE PostgreSQL |
| 10 | Отказ при неправильном JSON | POST/PATCH invalid_JSON |
| 11 | Работа взаимозаменяемых реализаций UserRepository | обычные тесты с памятью и интеграционные с PostgreSQL |
| 12 | Журналирование метода и пути каждого запроса | не проверено утверждениями теста |
| 13 | Интерактивная документация Swagger | не автоматизировано в ЛР7 |
| 14 | Применение и откат схемы миграциями | создание тестовой схемы используется; полного теста up/down нет, критерий не засчитан |

Нагрузочные показатели, доступность, восстановление и резервное копирование в эту функциональную матрицу не входят и этими тестами не доказаны. Проверки name при частичном обновлении и ошибок инфраструктуры можно расширить отдельно; покрытие не означает полного соответствия всем формулировкам ТЗ. Текущие успешные ответы API являются текстовыми, поэтому тесты не объявляют подтвержденным JSON-формат ответов из ТЗ.
