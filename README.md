# GrabLenta

Парсер товаров с **lenta.com** на Go.  
Использует JSON API сайта — не парсит HTML, устойчив к изменениям вёрстки.

## Структура проекта

```
GrabLenta/
├── cmd/parser/main.go               # Точка входа
├── internal/
│   ├── client/client.go             # HTTP-клиент с прокси и middleware
│   ├── config/config.go             # Загрузка конфига (YAML + ENV)
│   ├── config/config_test.go
│   ├── exporter/csv.go              # Экспорт в CSV
│   ├── exporter/csv_test.go
│   ├── middleware/middleware.go      # Retry, Logging, RateLimit, Headers
│   ├── middleware/middleware_test.go
│   ├── model/product.go             # Доменные типы
│   ├── model/product_test.go
│   ├── parser/parser.go             # Бизнес-логика парсинга
│   └── parser/parser_test.go
├── pkg/logger/logger.go             # JSON-логгер (stdout + файл)
├── config.yaml                      # Конфигурация
├── Makefile
├── go.mod
└── go.sum
```

## Требования

- Go 1.21+
- Российский прокси или VPN с российским сервером

> ⚠️ Сайт lenta.com блокирует запросы с зарубежных IP-адресов.  
> Для запуска из-за рубежа обязательно укажите российский прокси.

## Быстрый старт

```bash
# 1. Установить зависимости
go mod tidy

# 2. Запустить тесты
go test ./...

# 3. Запустить парсер с прокси
PROXY_URL=http://user:pass@host:8080 go run ./cmd/parser

# Или через config.yaml — прописать proxy.url
go run ./cmd/parser
```

## Конфигурация

Все параметры в `config.yaml`. Переменные окружения переопределяют YAML:

| ENV          | Что переопределяет |
|--------------|--------------------|
| `PROXY_URL`  | `proxy.url`        |
| `STORE_SLUG` | `store.slug`       |

### Добавить категорию

```yaml
categories:
  - slug: "myaso-i-ptica"
    name: "Мясо и птица"
```

### Как найти slug магазина

1. Открыть lenta.com в браузере, выбрать адрес доставки
2. DevTools → Network → любой запрос к `/api/v1/catalog/`
3. В заголовке запроса найти `X-Store-Slug`

## Middleware стек

Запросы проходят через цепочку (внешний → внутренний):

```
Logging → Retry → RateLimit → Headers → Transport(proxy)
```

| Middleware   | Что делает                                         |
|--------------|----------------------------------------------------|
| `Logging`    | Логирует каждый запрос: URL, статус, время         |
| `Retry`      | Повтор при 429/5xx с экспоненциальным back-off     |
| `RateLimit`  | Случайная задержка [min_ms, max_ms] перед запросом |
| `Headers`    | Инжектирует User-Agent, Accept, X-Store-Slug       |

## Тесты

```bash
go test ./...          # все тесты
go test ./... -v       # с подробным выводом
go test ./... -race    # с детектором гонок
```

Покрыты все пакеты: `model`, `config`, `middleware`, `parser`, `exporter`.

## Выходной формат (products.csv)

```
Категория;Наименование товара;Цена (руб.);Ссылка
Молочная продукция;Молоко Простоквашино 3.2% 950мл;89.90;https://lenta.com/...
Хлебобулочные изделия;Батон Нарезной 400г;44.90;https://lenta.com/...
```

UTF-8 с BOM — корректно открывается в Excel и LibreOffice.

## Логи

Параллельно с `products.csv` создаётся `parser.log` с JSON-логами всех запросов.
