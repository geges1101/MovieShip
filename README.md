# MovieShip — Онлайн-кинотеатр на Go + Keycloak + MinIO

## Описание

Упрощённый монолитный сервис для онлайн-кинотеатра:
- Аутентификация через Keycloak (OIDC)
- Хранение фильмов и истории просмотров в PostgreSQL
- Загрузка и выдача видео через MinIO (S3)
- REST API на Gin

## Переменные окружения

- DB_HOST, DB_PORT, DB_USER, DB_PASSWORD, DB_NAME — параметры PostgreSQL
- OIDC_ISSUER — URL OIDC-провайдера (Keycloak, например: http://keycloak:8080/realms/movieship)
- OIDC_CLIENT_ID — client_id для OIDC
- MINIO_ENDPOINT — адрес MinIO (minio:9000)
- MINIO_ACCESS_KEY, MINIO_SECRET_KEY — учётные данные MinIO
- MINIO_BUCKET — имя бакета (например, videos)

## Запуск

```bash
docker-compose up --build
```

## Настройка Keycloak
1. Создайте Realm (например, movieship)
2. Создайте Client (например, movieship-client, тип public)
3. Создайте пользователей и назначьте роли (admin для загрузки видео)

## Примеры запросов

### Получить список фильмов
```
GET /api/movies
Authorization: Bearer <access_token>
```

### Загрузить видео (только admin)
```
POST /api/upload
Authorization: Bearer <admin_token>
Content-Type: multipart/form-data
Поля: file (файл), title (название), description (описание)
```

### Получить ссылку на просмотр видео
```
GET /api/movies/{id}/stream
Authorization: Bearer <access_token>
```

### Получить историю просмотров
```
GET /api/history
Authorization: Bearer <access_token>
```

### Обновить прогресс просмотра
```
POST /api/history
Authorization: Bearer <access_token>
Content-Type: application/json
{
  "movie_id": 1,
  "progress": 120
}
```

## Структура API

- `GET /api/movies` — список фильмов
- `GET /api/movies/{id}` — информация о фильме
- `POST /api/upload` — загрузка видео (admin)
- `GET /api/movies/{id}/stream` — получить ссылку на видео
- `GET /api/history` — история просмотров
- `POST /api/history` — обновить прогресс

---
