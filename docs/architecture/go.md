# The architecture in Go

This guide shows how each part of `docs/ARCHITECTURE.md` looks in this Go project. The notes slice is the working example for every section.

## Layout

```
cmd/server/main.go                    composition root: entry point
internal/app/                         composition root: app.go, config.go, live.go, wire_<slice>.go
internal/domain/                      domain: model, rules, ports
internal/domain/middleware/           domain: middleware
internal/features/<slice>/ui/         UI of a slice (HTTP handlers)
internal/features/<slice>/usecases/   use cases of a slice
internal/sharedui/                    shared UI, such as httpjson
internal/adapters/<name>/             one directory per adapter
internal/utilities/
internal/testsupport/
```

A module is a directory and every package below it. `internal/domain/middleware` belongs to the domain module. Slice directories use lowercase names without separators, such as `recurringtransactions`. `python3 tools/new.py slice <Name>` creates the directories, their tests, and the wiring file, and adds the wiring call to `internal/app/app.go`.

## Domain

- **Model.** Structs and named types. A value with rules has unexported fields and a constructor that returns an error, like `NewNoteTitle`. Treat its zero value as invalid.
- **Rules.** Plain functions, like `NewestFirst`.
- **Errors.** Exported sentinel errors or error types, like `ErrTitleEmpty`. Callers check them with `errors.Is` or `errors.As`, and code wraps underlying errors with `fmt.Errorf("...: %w", err)`.
- **Ports.** Interfaces in `ports.go` or in a file per port. Methods that may do I/O take a `context.Context` first.
- **Middleware.** A struct in `domain/middleware` that wraps a port and implements it, like `LoggingNoteStore`.

The domain may import the standard library except the I/O packages that `architecture.json` lists under `io-packages`, such as `net/...`, `os`, and `database/sql`. It may use the `time.Time` type but must not call `time.Now`, `time.Sleep`, or similar functions. Take the time from the `Clock` port.

## Slices

- **Use cases** are structs with unexported port fields, a `NewX` constructor, and a `Run(ctx, ...)` method.
- **Handlers** live in the slice's `ui` package. A `Handler` takes its use cases and a logger in `NewHandler` and registers its routes in `Register(mux)`. Request and response types stay in the handler file and map to and from domain types.
- A handler logs the errors it handles and maps domain errors to status codes with `errors.Is`.
- Use `sharedui/httpjson` to read and write JSON. Reading ignores unknown fields, so newer clients can send fields that this version does not know yet.

UI code may not make outgoing HTTP requests or touch the file system. Put that behind a port.

## Adapters

An adapter package imports the domain and whatever libraries it needs. It keeps its stored or wire records unexported and maps them to domain types. Guard mutable state with a `sync.Mutex`. The `filestore` adapter shows atomic writes: it writes a temporary file and then renames it.

For SQL databases, keep numbered migrations in the adapter, and never edit a migration that has shipped.

## Composition root

- `app.go` defines `Dependencies` and `New`. `New` applies the mandatory middleware and calls one `wireX` method per slice.
- `wire_<slice>.go` builds the slice's use cases and handler, and decides which optional middleware the slice gets.
- `live.go` creates the real adapters, and `config.go` reads the environment.
- `cmd/server/main.go` reads the config, builds the app, and starts the server.
- Control flow is allowed in `config.go`. Elsewhere only `if err != nil {` and `if err := ...; err != nil {` are allowed, for error handling.

## Test support and tests

- Tests use the standard `testing` package. Domain and middleware tests that need test support live in external test packages, such as `domain_test`, to avoid an import cycle.
- Fakes guard their state with a mutex and can inject a failure, like `FakeNoteStore.FailNextCall`. Like the real store, the fake returns notes in no particular order, so tests must not depend on map order.
- A contract is a function that takes `*testing.T` and a constructor, like `NoteStoreContract`. `fakes_test.go` runs it against the fake, and the adapter's test runs it against the real implementation, with `t.TempDir()` for files.
- `internal/app/app_test.go` is the composition test. It builds the app with fakes and calls it through `httptest`.

`./check` runs `gofmt`, `go vet`, and `go test -race -shuffle=on`, so tests must pass in any order and without data races.
