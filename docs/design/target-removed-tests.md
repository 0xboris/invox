# Removed tests

`scripts/verify-target.sh` requires every test and fuzz function in
`docs/design/target/baseline-tests.txt` to still exist. A test that no longer applies after the
refactor is listed here, one per line, as `- TestName: reason`. Prefer moving and adapting a test
over removing it.
