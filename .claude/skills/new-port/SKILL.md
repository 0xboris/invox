---
name: new-port
description: Adds an adapter for a program or service invox needs, in the shape of the existing adapters. Use when command or domain code needs to run a program, reach the network, or use a platform or third-party service.
---

1. Check `docs/architecture/modules.md` and `internal/adapters/` first. Reuse an adapter that already covers the need, and add a method to it if the method belongs to the same program.
2. Declare the port in `internal/domain/ports`, named for what the caller needs, with plain values or domain types in its signatures.
3. Create `internal/adapters/programs/<name>` by hand. Copy `internal/adapters/programs/tectonic`: one program, wrapped on top of `run.Runner`, with `var _ ports.X = (*Adapter)(nil)`. Only `internal/adapters/programs/run` starts programs. Do not run `tools/new.py`.
4. Add the port to `app.Dependencies`, build the adapter in `app.NewDependencies` from the `run.Runner` it receives, and copy it into the Factory of each slice that uses it in `internal/app/wire_<slice>.go`.
5. Map the program's output to plain values or domain types inside the adapter, and return typed errors. Test it with a `run.Stub` that registers the program, as the existing adapter tests do. Tests must not run real programs.
6. Run `python3 tools/archcheck.py --module-map docs/architecture/modules.md`, then `./check`.
