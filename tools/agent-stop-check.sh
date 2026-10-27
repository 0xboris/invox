#!/usr/bin/env bash
# Claude Code Stop hook. When the session changed files, it runs ./check
# before the agent may finish. Exit code 2 sends the failures back to the
# agent. If the agent was already sent back once, the hook lets it stop, so
# it can report a failure it cannot fix instead of looping.
input=$(cat)
already_continued=$(printf '%s' "$input" | python3 -c 'import json, sys; print(json.load(sys.stdin).get("stop_hook_active", False))' 2>/dev/null)
[ "$already_continued" = "True" ] && exit 0

cd "${CLAUDE_PROJECT_DIR:-.}" || exit 0
git rev-parse --git-dir > /dev/null 2>&1 || exit 0

# An agent that works in a git worktree of this repository is checked there,
# since its changes are not in the main checkout.
cwd=$(printf '%s' "$input" | python3 -c 'import json, sys; print(json.load(sys.stdin).get("cwd", ""))' 2>/dev/null)
if [ -n "$cwd" ] && worktree=$(git -C "$cwd" rev-parse --show-toplevel 2>/dev/null) &&
  [ "$(git -C "$worktree" rev-parse --path-format=absolute --git-common-dir)" = "$(git rev-parse --path-format=absolute --git-common-dir)" ]; then
  cd "$worktree" || exit 0
fi

# Changes count against the default branch, so committed work on a feature
# branch is checked too. A repository without commits has only uncommitted changes.
if git rev-parse --verify --quiet HEAD > /dev/null; then
  default=$(git symbolic-ref --quiet --short refs/remotes/origin/HEAD 2>/dev/null || echo main)
  base=$(git merge-base HEAD "$default" 2>/dev/null || git rev-parse HEAD)
  if git diff --quiet "$base" && [ -z "$(git status --porcelain)" ]; then
    exit 0
  fi
elif [ -z "$(git status --porcelain)" ]; then
  exit 0
fi

if ! output=$(./check 2>&1); then
  {
    echo "./check fails. Fix the failures before you finish, or report the ones you cannot fix:"
    echo "$output" | tail -80
  } >&2
  exit 2
fi
exit 0
