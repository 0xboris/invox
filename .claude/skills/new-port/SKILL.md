---
name: new-port
description: Adds an adapter for a program or service invox needs, in the shape of the existing adapters. Use when command or domain code needs to run a program, reach the network, or use a platform or third-party service.
---

1. Check `docs/architecture/modules.md` and `internal/adapters/` first. Reuse an adapter that already covers the need, and add a method to it if the method belongs to the same program.
2. Create `internal/adapters/<name>` by hand. Copy `internal/adapters/tectonic`: one program, wrapped on top of `run.Runner`. Only `internal/adapters/run` starts programs. Do not run `tools/new.py`.
3. Name the adapter's operations for what the caller needs. Map the program's output to plain values or domain types inside the adapter, and return typed errors.
4. Add it to the `cmdutil.Factory` and build it in `cmdutil.NewFactory` from the `run.Runner` it receives.
5. Test it with a `run.Stub` that registers the program, as the existing adapter tests do. Tests must not run real programs.
6. Run `python3 tools/archcheck.py --module-map docs/architecture/modules.md`, then `./check`.
