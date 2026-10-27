#!/usr/bin/env python3
"""Keeps a pull request from loosening a check without an exception the owner approves.

.github/CODEOWNERS sends changes to the files that define the checks to the
owner. Some files need no owner, because most changes to them only tighten a
check: architecture-baseline.json, the project checks in checks/, the glossary,
and the source files. This guard catches the changes to them that loosen a check:

- architecture-baseline.json gains an entry or a higher count, including
  when it comes back after a pull request deleted it.
- an existing file in checks/ changes, loses its executable bit, or goes away.
- the glossary loses a term, which modules without business meaning may then use.
- a production source file gains a lint or type-check suppression, such as
  oxlint-disable or @ts-expect-error.

Such a pull request passes only when it adds a row to the exceptions table of
AGENTS.md, with a reason and an expiry date in the future, and the base
branch's CODEOWNERS gives AGENTS.md an owner, so that the owner reviews it.
A new row without a reason or a future date fails. A row whose expiry date has
passed fails every pull request until someone fixes what it allowed and removes
it. In GitHub Actions, each loosening is also posted as an annotation on the
pull request.

With --base-checks, it runs the project checks in checks/ as BASE has them,
against the files of HEAD. CI runs that before any code from the pull request
runs, so the pull request cannot change an existing check, or what it sees,
first. It skips a check that the pull request changes or removes with an
approved exception.

Usage:
  python3 tools/gateguard.py BASE                 compare HEAD with BASE, such as HEAD^1 in CI
  python3 tools/gateguard.py --base-checks BASE   run the checks in checks/ as BASE has them

Exit codes: 0 nothing loosened, or with an exception. 1 loosened without one, an
invalid or expired exception, or a failing check. 2 configuration error.
Needs Python 3.9 or later, Git, and nothing else.
"""

import datetime
import os
import re
import shutil
import stat
import subprocess
import sys
import tempfile

sys.dont_write_bytecode = True  # importing archcheck must not leave __pycache__ in tools/
sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
from archcheck import DEFAULT_EXTENSIONS, ConfigError, matches_any, parse_json_with_comments  # noqa: E402

BASELINE = "architecture-baseline.json"
CHECKS = "checks/"
CODEOWNERS = ".github/CODEOWNERS"
EXCEPTIONS = "AGENTS.md"
EXCEPTIONS_HEADING = "## Exceptions"
SUPPRESSION = re.compile(r"oxlint-disable|eslint-disable|@ts-ignore|@ts-expect-error|@ts-nocheck|\bnolint\b|\blint:ignore\b|swiftlint:disable")
DATE = re.compile(r"\d{4}-\d{2}-\d{2}")


def git(root, *args):
    return subprocess.run(["git", "-c", "core.quotePath=false", *args], cwd=root, capture_output=True, text=True)


def file_at(root, revision, path):
    shown = git(root, "show", f"{revision}:{path}")
    return shown.stdout if shown.returncode == 0 else None


def printable(text):
    """Text from a pull request, with control characters replaced, so it cannot start a line or a workflow command."""
    return "".join(c if c.isprintable() else "?" for c in text)


def annotation(level, path, message):
    """A GitHub Actions workflow command, escaped as the Actions toolkit escapes it."""
    def data(text):
        return text.replace("%", "%25").replace("\r", "%0D").replace("\n", "%0A")
    return f"::{level} file={data(path).replace(':', '%3A').replace(',', '%2C')}::{data(message)}"


def changed_files(root, base):
    """(status, path) pairs. -z keeps paths with unusual characters unquoted."""
    diff = git(root, "diff", "-z", "--name-status", "--no-renames", base, "HEAD")
    if diff.returncode != 0:
        raise ConfigError(f"git diff against '{base}' failed: {diff.stderr.strip()}")
    fields = diff.stdout.split("\0")
    return list(zip(fields[0::2], fields[1::2]))


def base_config(root, base):
    config = file_at(root, base, "architecture.json")
    return parse_json_with_comments(config, f"{base}:architecture.json") if config is not None else {}


def baseline_raises(root, base):
    new_text = file_at(root, "HEAD", BASELINE)
    if new_text is None:
        return []
    old_text = file_at(root, base, BASELINE)
    if old_text is None and file_at(root, base, "tools/gateguard.py") is None:
        return []  # this pull request adds the kit, and its baseline accepts the existing violations
    old = parse_json_with_comments(old_text, f"{base}:{BASELINE}") if old_text is not None else {}
    new = parse_json_with_comments(new_text, BASELINE)
    return [f"raises the baseline ({key}: {old.get(key, 0)} -> {count})"
            for key, count in sorted(new.items()) if count > old.get(key, 0)]


def glossary_removals(root, base, glossary):
    def terms(text):
        return {line[3:].strip() for line in (text or "").splitlines() if line.startswith("## ")}
    removed = terms(file_at(root, base, glossary)) - terms(file_at(root, "HEAD", glossary))
    return [f"removes the glossary term '{term}', which modules without business meaning may then use"
            for term in sorted(removed)]


def added_suppressions(root, base, config, changes):
    """Suppression comments that the pull request adds to production source files."""
    extensions = tuple(config.get("sourceExtensions", DEFAULT_EXTENSIONS.get(config.get("language"), [])))
    exempt = config.get("testFiles", []) + config.get("scriptFiles", [])
    found = []
    for status, path in changes:
        if status == "D" or not path.endswith(extensions) or matches_any(path, exempt):
            continue
        diff = git(root, "--literal-pathspecs", "diff", "-U0", "--no-color", base, "HEAD", "--", path)
        line_number = None  # the file header comes before the first hunk
        for line in diff.stdout.splitlines():
            if line.startswith("@@"):
                match = re.search(r"\+(\d+)", line)
                line_number = int(match.group(1)) if match else 0
            elif line.startswith("+") and line_number is not None:
                if SUPPRESSION.search(line):
                    found.append((path, f"adds a lint or type-check suppression at {path}:{line_number}"))
                line_number += 1
    return found


def exception_rows(text):
    """The rows of the exceptions table in AGENTS.md, as lists of cells, without the header and separator rows."""
    rows, inside = [], False
    for line in (text or "").splitlines():
        if line.startswith("## "):
            inside = line.strip() == EXCEPTIONS_HEADING
        elif inside and line.lstrip().startswith("|"):
            cells = [cell.strip() for cell in line.strip().strip("|").split("|")]
            if cells[0] != "Exception" and not set("".join(cells)) <= set("-: "):
                rows.append(cells)
    return rows


def expiry(row):
    """The date in the row's third cell, or None when that cell holds no valid YYYY-MM-DD date."""
    if len(row) < 3 or not DATE.fullmatch(row[2]):
        return None
    try:
        return datetime.date.fromisoformat(row[2])
    except ValueError:
        return None


def assess(root, base):
    """The changed files, the loosenings, whether an approved exception covers them, and the bad rows."""
    config = base_config(root, base)
    changes = changed_files(root, base)
    changed = {path for _, path in changes}
    problems = []
    for status, path in changes:
        if path.startswith(CHECKS) and status != "A":
            verb = "deletes" if status == "D" else "changes"
            problems.append((path, f"{verb} the existing check {path}"))
        if path == BASELINE:
            problems += [(BASELINE, problem) for problem in baseline_raises(root, base)]
    glossary = config.get("glossary")
    if glossary in changed:
        problems += [(glossary, problem) for problem in glossary_removals(root, base, glossary)]
    problems += added_suppressions(root, base, config, changes)

    today = datetime.date.today()
    head_rows = exception_rows(file_at(root, "HEAD", EXCEPTIONS))
    base_rows = exception_rows(file_at(root, base, EXCEPTIONS))
    added = [row for row in head_rows if row not in base_rows]
    invalid = [row for row in added if len(row) < 3 or not row[0] or not row[1] or not expiry(row) or expiry(row) <= today]
    expired = [row for row in head_rows if expiry(row) is not None and expiry(row) <= today]
    owned = is_owned(owner_rules(file_at(root, base, CODEOWNERS) or ""), EXCEPTIONS)
    return changes, problems, bool(added) and not invalid and owned and not expired, invalid, expired


def owner_rules(text):
    """CODEOWNERS lines as (regex, has_owner). The last matching line decides, as on GitHub."""
    rules = []
    for line in text.splitlines():
        fields = line.split("#", 1)[0].split()
        if fields:
            rules.append((pattern_regex(fields[0]), len(fields) > 1))
    return rules


def pattern_regex(pattern):
    anchored = pattern.startswith("/") or "/" in pattern.rstrip("/")
    directory = pattern.endswith("/")
    body, i, text = [], 0, pattern.strip("/")
    while i < len(text):
        if text.startswith("**/", i):
            body.append("(?:.*/)?")
            i += 3
        elif text.startswith("**", i):
            body.append(".*")
            i += 2
        elif text[i] == "*":
            body.append("[^/]*")
            i += 1
        elif text[i] == "?":
            body.append("[^/]")
            i += 1
        else:
            body.append(re.escape(text[i]))
            i += 1
    start = "^" if anchored else "^(?:.*/)?"
    end = "/.*$" if directory else "(?:/.*)?$"
    return re.compile(start + "".join(body) + end)


def is_owned(rules, path):
    owned = False
    for regex, has_owner in rules:
        if regex.match(path):
            owned = has_owner
    return owned


def guard(root, base):
    _, problems, approved, invalid, expired = assess(root, base)
    annotate = os.environ.get("GITHUB_ACTIONS") == "true"

    def report(message):
        print(f"Gate guard: {printable(message)}")
        if annotate:
            print(annotation("error", EXCEPTIONS, message))

    for row in expired:
        report(f"The exception '{row[0]}' expired on {row[2]}. Fix what it allowed, then remove its row from {EXCEPTIONS}.")
    for row in invalid:
        if row not in expired:
            report(f"The new row '{' | '.join(row)}' in the exceptions table of {EXCEPTIONS} needs what it allows, "
                   "a reason, and an expiry date in the future, written as YYYY-MM-DD.")
    failed = bool(expired or invalid)
    if not problems:
        if not failed:
            print("Gate guard passed: no check was loosened.")
        return 1 if failed else 0

    print(f"Gate guard: this pull request loosens {len(problems)} check{'' if len(problems) == 1 else 's'}.\n")
    for path, problem in problems:
        print(f"  It {printable(problem)}.")
        if annotate:
            print(annotation("warning" if approved else "error", path, f"This pull request {problem}."))
    if approved:
        print(f"\nIt adds an exception to {EXCEPTIONS}, so the owner reviews it.")
        return 0
    print(f"\nTo loosen a check, add a row to the exceptions table of {EXCEPTIONS} in the same pull request, "
          f"with the reason and the date it expires. {EXCEPTIONS} is a gate file, so the owner reviews it. "
          "Otherwise fix the violation, or add a new check.")
    return 1


def base_checks(root, base):
    listing = git(root, "ls-tree", "-r", "-z", base, "--", CHECKS)
    entries = [entry for entry in listing.stdout.split("\0") if entry]
    if listing.returncode != 0 or not entries:
        print("The base has no project checks.")
        return 0
    changes, _, approved, _, _ = assess(root, base)
    skip = {path for status, path in changes if path.startswith(CHECKS) and status != "A"} if approved else set()
    copy = tempfile.mkdtemp()
    try:
        for entry in entries:
            info, path = entry.split("\t", 1)
            mode, kind, _ = info.split()
            if kind != "blob":
                continue
            target = os.path.join(copy, path)
            os.makedirs(os.path.dirname(target), exist_ok=True)
            with open(target, "wb") as f:
                f.write(subprocess.run(["git", "show", f"{base}:{path}"], cwd=root, capture_output=True).stdout)
            if mode == "100755":
                os.chmod(target, os.stat(target).st_mode | stat.S_IXUSR | stat.S_IXGRP | stat.S_IXOTH)
        status = 0
        for name in sorted(os.listdir(os.path.join(copy, "checks"))):
            path = os.path.join(copy, "checks", name)
            if not (os.path.isfile(path) and os.access(path, os.X_OK)):
                continue
            label = printable(name)
            if f"{CHECKS}{name}" in skip:
                print(f"skip  {label} (the pull request changes it, with an exception in {EXCEPTIONS})")
            elif subprocess.run([path], cwd=root).returncode == 0:
                print(f"ok    {label} (as the base has it)")
            else:
                print(f"FAIL  {label} (as the base has it)")
                status = 1
        return status
    finally:
        shutil.rmtree(copy, ignore_errors=True)


def main():
    args = sys.argv[1:]
    run_checks = args[:1] == ["--base-checks"]
    if run_checks:
        args = args[1:]
    if len(args) != 1:
        print(__doc__.strip(), file=sys.stderr)
        return 2
    root = git(".", "rev-parse", "--show-toplevel").stdout.strip()
    if not root:
        print("Gate guard: not inside a Git repository.", file=sys.stderr)
        return 2
    try:
        return base_checks(root, args[0]) if run_checks else guard(root, args[0])
    except ConfigError as error:
        print(f"Gate guard: {error}", file=sys.stderr)
        return 2


if __name__ == "__main__":
    sys.exit(main())
