# 2025_2_A4Code
Backend команды A4 Code. Проект почта
## Генерация кода

### EasyJSON

Проект использует [EasyJSON](https://github.com/mailru/easyjson) для высокопроизводительной сериализации/десериализации JSON.

#### Использование

При добавлении новой структуры или изменении существующей в `internal/domain/` или `internal/lib/api/response/`, необходимо:

1. **Добавить директиву `//go:generate`** в начало файла с определением структуры:
   ```go
   package domain

   //go:generate go run github.com/mailru/easyjson/easyjson@v0.9.1 -all <filename>.go

   type MyStruct struct {
       Field string `json:"field"`
   }
   ```

2. **Запустить генерацию кода**:
   ```bash
   go generate ./...
   ```

   Или для конкретного пакета:
   ```bash
   go generate ./internal/domain/...
   ```

#### Что происходит

При запуске `go generate` для каждого файла с директивой создается соответствующий файл `*_easyjson.go` с методами:
- `MarshalJSON()` — сериализация структуры в JSON
- `UnmarshalJSON()` — десериализация JSON в структуру

#### Исключение из версионирования

Сгенерированные файлы автоматически исключены из Git:
```
*_easyjson.go
```

Также исключены из отчета о покрытии тестами (`.codecovignore`).

#### Требования к структурам

Все поля, которые должны сериализоваться, **обязаны иметь JSON tags**:
```go
type Profile struct {
    ID       int64     `json:"id"`
    Username string    `json:"username"`
    Email    string    `json:"email"`
}
```

**Важно**: избегайте встроенных (embedded) структур при использовании EasyJSON, так как это может привести к конфликтам в JSON tags. Вместо этого явно указывайте все поля.

#### CI/CD

В CI pipeline (`.github/workflows/ci.yml`) добавлена стадия `generate`, которая запускает `go generate ./...` перед линтингом и тестированием. Это гарантирует, что все необходимые файлы сгенерированы перед проверкой кода.