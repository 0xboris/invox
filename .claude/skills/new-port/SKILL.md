---
name: new-port
description: Adds a port and its adapter for a program or service invox needs, in the shape of the existing adapters. Use when use case or domain code needs to run a program, reach the network, or use a platform or third-party service.
---

1. Check `docs/architecture/modules.md` and `internal/domain/ports` first. Reuse a port that already covers the need, and add a method to it if the method belongs to the same program.
2. Declare the port in `internal/domain/ports`, which `portFiles` in `architecture.json` lists, so the check keeps commands away from it. Name it for what the use case needs, not for the technology behind it, with plain values or domain types in its signatures.
3. Make each operation's errors a closed set: typed errors that callers can match with `errors.As`, like `ports.CompilerNotInstalledError`.
4. Create `internal/adapters/programs/<name>` by hand. Copy `internal/adapters/programs/tectonic`: one program, wrapped on top of `run.Runner`, with `var _ ports.X = (*Adapter)(nil)`. Only `internal/adapters/programs/run` starts programs. Map the program's output to plain values or domain types inside the adapter.
5. Test the adapter with a `run.Stub` that registers the program, as the existing adapter tests do. Tests must not run real programs.
6. Add the port to `app.Dependencies`, build the adapter in `app.NewDependencies` from the `run.Runner` it receives, and pass it to the constructors of the use cases that need it in `internal/app/wire_<slice>.go`. Commands reach it only through those use cases.
7. When a use case test needs the port to fail, add a failing implementation to `internal/testsupport`, like `FailingOpener`.
8. Run `python3 tools/archcheck.py --module-map docs/architecture/modules.md`, then `./check`.
