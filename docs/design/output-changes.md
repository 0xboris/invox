# Output changes

Round 2 lets CLI output change when that makes the code cleaner (see
`docs/design/target/BASELINE`). Every changed testscript golden under
`cmd/invox/testdata`, every changed page under `docs/cli` or `share/man`, and every
changed or removed test fixture is named here with the reason. `scripts/verify-target.sh`
fails on any such change that is not listed.

Format: one section per commit or unit, listing each changed file path and what changed
for the user.
