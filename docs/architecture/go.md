# The architecture in Go

This guide shows how each part of `docs/ARCHITECTURE.md` looks in Go. It holds the rules that come from the language. `docs/architecture/stack.md` names the libraries this project uses, such as the HTTP router and the logger, and how the reference slice uses them.

This guide belongs to the kit, and a kit upgrade replaces it, so the project does not edit it. Its examples name the kit's Notes slice. Once Notes is gone, the reference slice that `AGENTS.md` names shows the same patterns.

## Layout

```
cmd/server/main.go                    composition root: entry point
internal/app/                         composition root: app.go, config.go, live.go, wire_<slice>.go
internal/domain/                      domain: model, rules, ports
internal/domain/middleware/           domain: middleware
internal/features/<slice>/ui/         UI of a slice
internal/features/<slice>/usecases/   use cases of a slice
internal/sharedui/                    shared UI
internal/adapters/<name>/             one directory per adapter
internal/utilities/
internal/testsupport/
```

A module is a directory and every package below it. `internal/domain/middleware` belongs to the domain module. Slice directories use lowercase names without separators, such as `recurringtransactions`. A new slice copies the files of the reference slice: its `ui` and `usecases` packages, their tests, and `internal/app/wire_<slice>.go`, which `New` in `internal/app/app.go` calls.

## Domain

- **Model.** Structs and named types. A value with rules has unexported fields and a constructor that returns an error, like `NewNoteTitle`. Treat its zero value as invalid.
- **Rules.** Plain functions, like `NewestFirst`.
- **Errors.** Exported sentinel errors or error types, like `ErrTitleEmpty`. Callers check them with `errors.Is` or `errors.As`, and code wraps underlying errors with `fmt.Errorf("...: %w", err)`. A port documents the errors it returns, so callers and their tests can handle each one.
- **Ports.** Interfaces in `ports.go` or in a file per port. Methods that may do I/O take a `context.Context` first.
- **Middleware.** A struct in `domain/middleware` that wraps a port and implements it, like `LoggingNoteStore`.

The domain may import the standard library except the I/O packages that `architecture.json` lists under `io-packages`, such as `net/...`, `os`, and `database/sql`. It may use the `time.Time` type but must not call `time.Now`, `time.Sleep`, or similar functions. Take the time from the `Clock` port.

## Slices

- **Use cases** are structs with unexported port fields, a `NewX` constructor, and a `Run(ctx, ...)` method. Settings are plain values, such as a retention period, that `config.go` reads and the wiring file passes to the constructor.
- **The UI package** takes the slice's use cases and a logger in its constructor. Request and response types stay in the UI package and map to and from domain types.
- The UI package logs the errors it handles and maps domain errors to output with `errors.Is`.

UI code may not make outgoing requests or touch the file system. Put that behind a port.

## Other entry points

- A background job or a scheduled task is an entry point of its slice, like a handler. Its code lives in the slice's UI package and calls the slice's use cases.
- The queue or scheduler that runs jobs is an adapter. The composition root registers each slice's jobs with it and sets their schedules from configuration.
- A use case that puts work on a queue does so through a port, which the queue adapter implements.
- When the service runs as more than one process, such as an API and a worker, each process gets its own directory under `cmd/`, such as `cmd/server` and `cmd/worker`.
- When an adapter holds a resource that must be closed, such as a database pool, `live.go` returns it with the dependencies, and `main.go` closes it on shutdown. Migrations run before the process serves, at start or from their own command under `cmd/`.

## Adapters

An adapter package imports the domain and whatever libraries it needs. It keeps its stored or wire records unexported and maps them to domain types. Guard mutable state with a `sync.Mutex`.

For SQL databases, keep numbered migrations in the adapter, and never edit a migration that has shipped. A helper that creates a real database for tests lives in the adapter package, next to the adapter's tests, because test support may not import adapters.

## Composition root

- `app.go` defines `Dependencies` and `New`. `New` applies the mandatory middleware and calls one `wireX` method per slice.
- `wire_<slice>.go` builds the slice's use cases and UI, and decides which optional middleware the slice gets.
- `live.go` creates the real adapters, and `config.go` reads the environment.
- `cmd/server/main.go` reads the config, builds the app, and starts it.
- Control flow is allowed in `config.go`. Elsewhere the check allows only `if err != nil {` and `if err := ...; err != nil {`, for error handling.

## Tests

- Tests use the standard `testing` package. Domain and middleware tests that need test support live in external test packages, such as `domain_test`, to avoid an import cycle.
- Use case, UI, middleware, and composition tests use the real adapter, such as `filestore.New(filepath.Join(t.TempDir(), "notes.json"))`, and assert what the store returns afterwards.
- Test support holds sample data, like `SampleNote`, small fakes for ports whose state a test drives, like `FakeClock`, and implementations that fail, like `FailingNoteStore`. Each asserts at compile time that it implements the port: `var _ domain.NoteStore = FailingNoteStore{}`.
- The adapter's own tests run against the real system, such as a real database or `t.TempDir()`. A stored format has a file in the adapter's `testdata/`, like `notes-v1.json`, and a test that loads it.
- `internal/app/app_test.go` is the composition test. It builds the app on a temporary store and checks which middleware the slices receive.

Tests must pass in any order and without data races.
