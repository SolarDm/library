# Library Server

Небольшой сервис, который представляет собой электронную библиотеку.
На данный момент он содержит простейшие команды для работы с книгами,
подробнее будет рассказано ниже.

# Запуск

Перед запуском проекта следует прописать в командной строке

```bash
make generate
```

В случае если вы работаете на Windows рекомендуется использовать WSL.

[Стартовая точка проекта](cmd/library/main.go)

# Реализованные запросы

* AddBook - добавляет книгу в библиотеку
* UpdateBook - изменяет данные у книги в библиотеке
* GetBookInfo - возвращает данные книги, находящейся в библиотеке
* RegisterAuthor - добавляет данные автора в библиотеку
* ChangeAuthorInfo - обновляет информацию об авторе
* GetAuthorInfo - возвращает данные об авторе
* GetAuthorBooks - возвращает все книги определённого автора

Более подробно с каждым из запросов можно ознакомится в [файле](
../api/library/library.proto).

# Тесты

К данному проекту написаны unit тесты, для которых были сгенерированы моки.
Кроме того, присутствуют integration тесты.

# Особенности

В проекте были использованы следующие технологии

* Структура проекта [go-clean-template](https://github.com/evrone/go-clean-template)
* Для генерации кода [easyp](https://easyp.tech/)
* Для логирования [zap](https://github.com/uber-go/zap)
* Для валидации [protoc-gen-validate](https://github.com/bufbuild/protoc-gen-validate)
* Для поддержики REST-to-gRPC API [gRPC gateway](https://grpc-ecosystem.github.io/grpc-gateway/)
* Для миграций [goose](https://github.com/pressly/goose)
* [pgx](https://github.com/jackc/pgx) как драйвер для postgres

