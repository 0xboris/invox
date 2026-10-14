#!/usr/bin/env bash
# Claude Code Stop hook. When the session changed files, it runs ./check
# before the agent may finish. Exit code 2 sends the failures back to the
# agent. If the agent was already sent back once, the hook lets it stop, so
# it can report a failure it cannot fix instead of looping.
input=$(cat)
already_continued=$(printf '%s' "$input" | python3 -c 'import json, sys; print(json.load(sys.stdin).get("stop_hook_active", False))' 2>/dev/null)
[ "$already_continued" = "True" ] && exit 0

cd "${CLAUDE_PROJECT_DIR:-.}" || exit 0
base=$(git merge-base HEAD main 2>/dev/null || git rev-parse HEAD 2>/dev/null) || exit 0
if git diff --quiet "$base" 2>/dev/null && [ -z "$(git status --porcelain 2>/dev/null)" ]; then
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
