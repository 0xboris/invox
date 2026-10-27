---
name: verify-invox
description: Runs the invox CLI the way a user does, against a disposable config directory, and keeps each command, its output, and its exit code as evidence. Use it to show that a change works in the built binary, to reproduce a bug before fixing it, and wherever a pstack playbook proves a change in the running app.
---

# Verify invox

`scripts/app` in this directory builds invox, gives each run its own home, config directory, and work directory under `.verify/<run>/`, runs invox commands there, and records every command with its stdout, stderr, and exit code. Run it from the repository root. Runs never share state, and git ignores `.verify/`.

```bash
app=.claude/skills/verify-invox/scripts/app
```

## Launch

```bash
run=$($app start)
```

`start` builds `./cmd/invox` and prints the run id. The config directory and the work directory start empty. The programs that would open a window, `xdg-open`, `open`, and `osascript`, are stand-ins that append what they were asked to open to `.verify/<run>/opened.log`. Prompts are disabled, so a command that needs a terminal, such as `customer edit`, refuses with exit code 2 instead of waiting.

`build` needs `tectonic`. Where it is not installed, start the run with `$app start --fake-tectonic`, which puts a stand-in on the run's PATH that writes a placeholder PDF next to the `.tex` file. That proves invox's side of a build, its paths, status change, and output, but not the typesetting. Say which one you used in the report.

## Doctor

```bash
$app doctor "$run"
```

Run it first whenever anything looks off. It prints `ok` with the invox version, the run's directories, and whether `tectonic` is real, fake, or missing. Otherwise it prints `FAIL` with the reason. Drive only a run that `doctor` reports as `ok`.

## Drive

```bash
$app run "$run" <name> <invox arguments...>
$app show "$run" <name> <path>
```

`run` runs invox in the run's work directory and prints `> invox <arguments>`, each stdout line as `stdout: ...`, each stderr line as `stderr: ...`, and `< exit <code>`. `show` prints a file, with its path relative to the work directory, such as `CUST-001-001.yaml` or `../opened.log`. `features/README.md` lists what invox does, and each feature file gives the exact commands and the result that proves the feature works.

## Evidence

Every `run` and `show` saves its output as `.verify/<run>/evidence/<name>.txt`. A proof shows the action and the resulting state, so after a command that writes a file, read the file back with `show` or with a command that reads it, such as `validate` or `archive list`. Cite the evidence files in the reply or the pull request.

## Cleanup

```bash
$app stop "$run"
```

`stop` deletes the run's binary, home, config directory, and work directory. The evidence stays in `.verify/<run>/evidence/`.

## Helpers

- `$app start [--fake-tectonic]` builds invox and prints a run id.
- `$app doctor <run>` checks the run.
- `$app run <run> <name> [args...]` runs invox and saves it as evidence.
- `$app show <run> <name> <path>` prints a file of the work directory and saves it as evidence.
- `$app stop <run>` removes the run and keeps the evidence.
- `$app --help` prints this list.
