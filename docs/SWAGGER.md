# Swagger API Documentation

Этот проект использует Swagger/OpenAPI для документирования REST API.

## Доступ к документации

После запуска Gateway, Swagger UI будет доступен по адресу:

```
http://localhost:8080/swagger/index.html
```

## Генерация документации

Для генерации/обновления Swagger документации используйте команду:

```bash
make swagger
```

Или напрямую:

```bash
swag init -g cmd/gateway/main.go -o docs/swagger --parseDependency --parseInternal
```

## Структура документации

- **docs/swagger/docs.go** - Go код с документацией
- **docs/swagger/swagger.json** - OpenAPI спецификация в JSON
- **docs/swagger/swagger.yaml** - OpenAPI спецификация в YAML

## Доступные эндпоинты

### Auth Service
- `POST /auth/request-code` - Запрос кода верификации
- `POST /auth/verify-code` - Верификация кода и создание сессии
- `POST /auth/logout` - Выход из системы
- `PATCH /auth/users/:id` - Обновление пользователя
- `GET /auth/users/:id/request-delete-code` - Запрос кода удаления
- `POST /auth/users/:id/delete` - Удаление пользователя

### Posts Service
- `POST /api/posts` - Создание поста
- `GET /api/posts` - Получение всех постов (с пагинацией)
- `GET /api/posts/:id` - Получение поста по ID
- `GET /api/users/:user_id/posts` - Получение постов пользователя
- `PATCH /api/posts/:id` - Обновление поста
- `DELETE /api/posts/:id` - Удаление поста

### Likes Service
- `POST /api/likes` - Поставить лайк
- `DELETE /api/likes/:post_id` - Убрать лайк
- `GET /api/posts/:post_id/likes/count` - Количество лайков
- `GET /api/posts/:post_id/likes/me` - Проверка лайка пользователя

### Files Service
- `POST /api/files/upload-url` - Генерация presigned URL для загрузки
- `POST /api/files/download-url` - Генерация presigned URL для скачивания
- `DELETE /api/files/:key` - Удаление файла

### Comments Service
- `POST /api/comments` - Создание комментария
- `PATCH /api/comments/:id` - Обновление комментария
- `DELETE /api/comments/:id` - Удаление комментария
- `GET /api/comments/post/:post_id` - Получение комментариев поста

### Follow Service
- `POST /api/follow` - Подписаться на пользователя
- `DELETE /api/follow/:user_id` - Отписаться от пользователя
- `GET /api/follow/:user_id/followers` - Получить подписчиков
- `GET /api/follow/:user_id/following` - Получить подписки

## Аутентификация

API использует cookie-based аутентификацию через сессии. Большинство эндпоинтов требуют наличия валидной сессии.

1. Запросите код верификации через `/auth/request-code`
2. Подтвердите код через `/auth/verify-code` - получите session cookie
3. Используйте session cookie для последующих запросов

## Примечания

- Swagger UI автоматически обновляется при перезапуске Gateway
- После изменения аннотаций необходимо перегенерировать документацию командой `make swagger`
- Для добавления новых эндпоинтов используйте Swagger аннотации в handler-функциях
