# TCP-Scanner

Go-пакет для TCP Connect-сканирования портов.

# Возможности
IPv4, IPv6 и DNS-имена.
DNS-имена с несколькими IP.
Отдельные порты и диапазоны.
Настраиваемые concurrency и timeout.
Отмена через context.Context.
Worker pool для ограничения количества подключений.
Состояния: open, closed, timeout, unreachable, canceled, error.
Структурированный результат с IP, портом, состоянием, временем и ошибкой.

# Использование

```go
scanner, err := tcpscanner.New(
    tcpscanner.WithConcurrency(100),
    tcpscanner.WithTimeout(500 * time.Millisecond),
)
if err != nil {
    log.Fatal(err)
}

hosts := tcpscanner.Hosts(
    "127.0.0.1",
    "example.com",
)

ports := tcpscanner.List(
    "22",
    "80",
    "443",
    "8000-8010",
)

results, err := scanner.Scan(
    context.Background(),
    hosts,
    ports,
)
if err != nil {
    log.Fatal(err)
}

for _, r := range results {
    fmt.Printf(
        "%s:%d %s %v\n",
        r.Host,
        r.Port,
        r.State,
        r.Duration,
    )
}
```

# API

`New`

Создаёт сканер.

```go
New(
    WithConcurrency(100),
    WithTimeout(500 * time.Millisecond),
)
```

`Hosts`

Принимает IPv4, IPv6 или DNS-имена. DNS-имя с несколькими IP создаёт отдельную цель для каждого IP.

`List`

Создаёт список портов и диапазонов:

```go
List("22", "80", "8000-8010")
```

`Range`

Создаёт диапазон:

```go
Range(1, 1000)
```

`Scan`

Запускает сканирование и возвращает []Result.

Result
```go
type Result struct {
    Host     string
    IP       net.IP
    Port     uint16
    State    State
    Duration time.Duration
    Err      error
}
```

State.String() возвращает строковое представление состояния.

# Тестирование

```bash
go test ./...
```

Проверка race conditions:

```bash
go test -race ./...
```

# Ограничения

Реализовано только TCP Connect-сканирование.

Не поддерживаются SYN/UDP-сканирование, raw sockets, определение ОС и функции полноценного Nmap.

Для определения некоторых сетевых ошибок используются системные сетевые коды, поэтому их классификация может отличаться между операционными системами.