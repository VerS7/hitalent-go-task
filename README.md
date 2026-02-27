# Тестовое задание от [**hitalent**](https://hh.ru/employer/11599905)

REST API для управления деревом подразделений и сотрудниками.

## Стек

- **Go** (`net/http`)
- **GORM**
- **PostgreSQL**
- **goose** _(migrations)_
- **Docker** & **docker-compose**

## Запуск

```bash
docker-compose up --build
```

После старта:

- API: `http://localhost:8080`
- healthcheck: `GET http://localhost:8080/healthcheck`

При запуске контейнера API автоматически:

1. ждёт PostgreSQL;
2. применяет миграции `goose up`;
3. запускает HTTP сервер.

## Структура проекта

## API

### 1. Создать подразделение

`POST /departments`

Body:

```json
{
    "name": "Backend",
    "parent_id": null
}
```

### 2. Создать сотрудника в подразделении

`POST /departments/{id}/employees`

Body:

```json
{
    "full_name": "Иван Петров",
    "position": "Senior Developer",
    "hired_at": "2023-09-01"
}
```

### 3. Получить подразделение (детали + сотрудники + поддерево)

`GET /departments/{id}?depth=1&include_employees=true`

- `depth`: по умолчанию `1`, диапазон `0..5`
- `include_employees`: по умолчанию `true`

### 4. Изменить подразделение

`PATCH /departments/{id}`

Body:

```json
{
    "name": "Platform",
    "parent_id": 2
}
```

### 5. Удалить подразделение

`DELETE /departments/{id}?mode=cascade`

или

`DELETE /departments/{id}?mode=reassign&reassign_to_department_id=3`

## Тесты

Запуск:

```bash
go test ./...
```
