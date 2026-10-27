# invox verification map

This directory lists what invox does for its users, and how to prove that each feature works. Read it before you drive invox, then follow the matching feature file. `$app` is `.claude/skills/verify-invox/scripts/app`, run from the repository root.

## Baseline preconditions

- Start a run with `run=$($app start)`, or `run=$($app start --fake-tectonic)` where `tectonic` is missing. Its config directory and work directory start empty.
- Run `$app doctor "$run"` and require `ok`.
- Never drive a run that you did not start.

## Driving conventions

- Start every recipe from the baseline state unless its preconditions say otherwise.
- Run every invox command with `$app run`, so that it is saved as evidence.
- Treat every command as literal. Keep quoted arguments unchanged.

## Proof and skip reporting

- Capture the command and the resulting state, not only the output of a write.
- The proof of a write includes a second step that reads the result back.
- Name the evidence file of each step in the report.
- Report an unreachable path with the attempted command and the unmet precondition, such as a missing `tectonic`.

## Feature entry contract

Each feature file starts with an H1 title and one paragraph that describes what a user can do. It then has four H2 sections in this order: `Sub-features`, `How to get to it (user POV)`, `Driving it with scripts/app`, and `Gotchas`.

## Features

- [Set up and customers](./setup-and-customers.md) covers `init`, `customer list`, and the editor gate of `customer edit`.
- [Create and check an invoice](./create-invoice.md) covers `new`, `validate`, and `render`.
- [Build an invoice](./build-invoice.md) covers `build` and the status it sets.
- [Email and archive](./email-and-archive.md) covers `email -o` and `archive add` and `archive list`.

When a change adds or changes what users can do, change the feature files in the same pull request.
