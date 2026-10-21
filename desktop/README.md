# Hevy Signal v3.2 source

Local training analytics companion for Hevy.

## Data sources
- Hevy Pro API: automatic workout + saved-routine sync.
- Hevy workout CSV: no API key required. CSV is parsed locally. Since the export does not contain saved routine definitions, current-program cards are explicitly inferred from recent workout history.

## Build/test

```bash
go test main.go main_test.go
go test -race main.go main_test.go
go build -o HevySignal main.go
```

The frontend is embedded from `static/`.

Independent project; not affiliated with or endorsed by Hevy.
