# Subscription Service

REST-сервис для хранения онлайн-подписок и расчёта их суммарной стоимости за выбранный период. Реализован на Go и PostgreSQL по [тестовому заданию StackBridge](https://stackbridge-it.ru/test-tasks/GO).

## Возможности

- полный CRUDL: создание, получение, список, изменение и удаление подписок;
- включительный расчёт стоимости по месяцам с учётом пересечения периодов;
- опциональная фильтрация списка и суммы по `user_id` и `service_name`;
- PostgreSQL-миграции с защитой от параллельного запуска;
- единые JSON-ошибки и соответствующие HTTP-коды;
- структурированные JSON-логи с уровнями `debug`, `info`, `warn`, `error`;
- request ID, access log и восстановление после panic;
- graceful shutdown по `SIGINT`/`SIGTERM`;
- Swagger UI, health checks, Docker Compose и CI.

## Быстрый запуск

Понадобится Docker с Compose plugin.

```bash
docker compose up --build
```

После запуска доступны:

- API: `http://localhost:8080/api/v1/subscriptions`;
- Swagger UI: `http://localhost:8080/swagger/`;
- OpenAPI: `http://localhost:8080/openapi.yaml`;
- liveness: `http://localhost:8080/health/live`;
- readiness: `http://localhost:8080/health/ready`.

Compose сначала дожидается PostgreSQL, затем применяет миграции и только после этого запускает API. Настройки можно изменить через переменные из `.env.example` или `config/config.yaml`.

Для запуска без Docker потребуется Go 1.25 и PostgreSQL 17.

## API

| Метод | Путь | Результат |
| --- | --- | --- |
| `POST` | `/api/v1/subscriptions` | создать подписку |
| `GET` | `/api/v1/subscriptions/{id}` | получить подписку |
| `GET` | `/api/v1/subscriptions` | получить список |
| `PUT` | `/api/v1/subscriptions/{id}` | полностью изменить подписку |
| `DELETE` | `/api/v1/subscriptions/{id}` | удалить подписку |
| `GET` | `/api/v1/subscriptions/total` | рассчитать сумму |

Пример создания:

```bash
curl -i -X POST http://localhost:8080/api/v1/subscriptions \
  -H 'Content-Type: application/json' \
  -d '{
    "service_name": "Yandex Plus",
    "price": 400,
    "user_id": "60601fee-2bf1-4721-ae6f-7636e79a0cba",
    "start_date": "07-2025",
    "end_date": "12-2025"
  }'
```

Список поддерживает `user_id`, `service_name`, `limit` (1–200) и `offset`:

```bash
curl 'http://localhost:8080/api/v1/subscriptions?user_id=60601fee-2bf1-4721-ae6f-7636e79a0cba&limit=20'
```

Изменение несуществующей подписки возвращает `404`:

```json
{
  "error": {
    "code": "subscription_not_found",
    "message": "subscription not found"
  }
}
```

## Расчёт стоимости

Начальный и конечный месяцы включаются в расчёт. Для каждой подписки берётся только пересечение её периода с запрошенным диапазоном:

```text
стоимость подписки = месячная цена × число месяцев в пересечении
итог = сумма стоимостей всех пересекающихся подписок
```

Например, подписка стоимостью 400 ₽ действует с `01-2025` по `03-2025`, а запрос сделан за `02-2025`–`04-2025`. Пересечение содержит февраль и март, поэтому результат для этой подписки равен `400 × 2 = 800 ₽`.

Параметры `user_id` и `service_name` опциональны и могут применяться вместе:

```bash
curl 'http://localhost:8080/api/v1/subscriptions/total?period_start=01-2025&period_end=06-2025&user_id=60601fee-2bf1-4721-ae6f-7636e79a0cba&service_name=Yandex%20Plus'
```

Ответ:

```json
{
  "total_cost": 2400,
  "currency": "RUB",
  "period_start": "01-2025",
  "period_end": "06-2025"
}
```

## HTTP-коды и ошибки

| Код | Когда возвращается |
| --- | --- |
| `200 OK` | успешное чтение, список, изменение или расчёт |
| `201 Created` | подписка создана; заголовок `Location` содержит её URL |
| `204 No Content` | подписка удалена |
| `400 Bad Request` | неверный JSON, UUID, дата, период, цена или пагинация |
| `404 Not Found` | подписка для чтения, изменения или удаления не существует |
| `500 Internal Server Error` | внутренняя ошибка без раскрытия технических деталей |
| `503 Service Unavailable` | БД недоступна при readiness-проверке |

Все ошибки API имеют форму `{"error":{"code":"...","message":"..."}}`.

## Конфигурация и логирование

Значения читаются из `config/config.yaml` и переопределяются переменными окружения. Полный перечень находится в `.env.example`. Уровень задаётся через `LOG_LEVEL`: `debug`, `info`, `warn` или `error`.

- `debug`: начало запроса, чтение списка и расчёт суммы;
- `info`: успешные запросы и изменения данных, запуск и остановка сервера;
- `warn`: клиентские HTTP-ошибки и обращение к отсутствующей записи;
- `error`: серверные ошибки, ошибки хранилища и panic.

Каждый HTTP-ответ содержит `X-Request-ID`; тот же идентификатор записывается в access log.

## Проверка

Модульные тесты:

```bash
go test -race ./...
```

Интеграционный тест PostgreSQL запускается при наличии отдельной тестовой БД:

```bash
TEST_DATABASE_URL='postgres://subscriptions:subscriptions@localhost:5432/subscriptions_test?sslmode=disable' \
  go test -race -run Integration ./internal/repository/postgres
```

Тест очищает таблицу `subscriptions`, поэтому `TEST_DATABASE_URL` должен указывать только на тестовую БД.

## Структура

```text
cmd/api                    запуск HTTP-сервера
cmd/migrate                запуск миграций
internal/config            конфигурация
internal/domain            модель и работа с месяцами
internal/httpapi           HTTP-ручки, DTO и middleware
internal/migrations        мигратор
internal/observability     настройка slog
internal/repository        PostgreSQL-репозиторий
internal/service           бизнес-логика
migrations                 SQL-миграции up/down
docs/openapi.yaml          спецификация OpenAPI
```
