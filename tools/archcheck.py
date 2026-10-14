#!/usr/bin/env python3
"""Architecture check for Swift, Go, and TypeScript projects.

Reads architecture.json, maps every source file to a module, and checks
the import rules, banned calls, glossary terms in modules without business
meaning, and the shape of the composition root.

Usage:
  python3 tools/archcheck.py                      check against the baseline
  python3 tools/archcheck.py --write-baseline     accept all current violations
  python3 tools/archcheck.py --module-map FILE    write the module map as Markdown

Exit codes: 0 no new violations, 1 new violations, 2 configuration error.
Needs Python 3.9 or later and nothing else.
"""

import argparse
import json
import os
import re
import sys
from collections import defaultdict

SKIP_DIRS = {".git", ".build", ".swiftpm", "build", "DerivedData", "node_modules", "dist", "vendor", ".idea", ".vscode"}
DEFAULT_EXTENSIONS = {"swift": [".swift"], "go": [".go"], "typescript": [".ts", ".tsx", ".mts", ".cts"]}
CONTROL_FLOW = {
    "swift": ["if", "guard", "switch", "for", "while", "repeat", "catch"],
    "go": ["if", "for", "switch", "select", "goto"],
    "typescript": ["if", "for", "while", "switch", "catch"],
}
DOCS = "docs/ARCHITECTURE.md"


class ConfigError(Exception):
    pass


# ---------------------------------------------------------------- utilities

def load_json_with_comments(path):
    with open(path, encoding="utf-8") as f:
        lines = [line for line in f if not line.lstrip().startswith("//")]
    try:
        return json.loads("".join(lines))
    except json.JSONDecodeError as error:
        raise ConfigError(f"{path} is not valid JSON: {error}")


def glob_to_regex(pattern):
    out, i = [], 0
    while i < len(pattern):
        if pattern.startswith("**/", i):
            out.append("(?:.*/)?")
            i += 3
        elif pattern.startswith("**", i):
            out.append(".*")
            i += 2
        elif pattern[i] == "*":
            out.append("[^/]*")
            i += 1
        elif pattern[i] == "?":
            out.append("[^/]")
            i += 1
        else:
            out.append(re.escape(pattern[i]))
            i += 1
    return re.compile("^" + "".join(out) + "$")


def matches_any(path, patterns):
    return any(glob_to_regex(p).match(path) for p in patterns)


def name_pattern_matches(name, pattern):
    """External names: '*' matches any characters, including '/'."""
    if pattern == "*":
        return True
    regex = "^" + re.escape(pattern).replace(r"\*", ".*") + "$"
    return re.match(regex, name) is not None


def strip_source(code, language, keep_strings):
    """Blank out comments, and string contents unless keep_strings. Keeps line breaks."""
    out, i, n = [], 0, len(code)

    def blank(text):
        return "".join(ch if ch == "\n" else " " for ch in text)

    while i < n:
        ch = code[i]
        if code.startswith("//", i):
            end = code.find("\n", i)
            end = n if end == -1 else end
            out.append(blank(code[i:end]))
            i = end
        elif code.startswith("/*", i):
            depth, j = 1, i + 2
            while j < n and depth:
                if language == "swift" and code.startswith("/*", j):
                    depth, j = depth + 1, j + 2
                elif code.startswith("*/", j):
                    depth, j = depth - 1, j + 2
                else:
                    j += 1
            out.append(blank(code[i:j]))
            i = j
        elif language == "swift" and code.startswith('"""', i):
            end = code.find('"""', i + 3)
            end = n if end == -1 else end + 3
            out.append(code[i:end] if keep_strings else '"' + blank(code[i + 1:end - 1]) + '"')
            i = end
        elif ch == '"' or (ch == "'" and language in ("go", "typescript")) or (ch == "`" and language in ("go", "typescript")):
            quote, j = ch, i + 1
            while j < n:
                if code[j] == "\\" and quote != "`":
                    j += 2
                    continue
                if code[j] == quote:
                    break
                if code[j] == "\n" and quote != "`":
                    break
                j += 1
            end = min(j + 1, n)
            out.append(code[i:end] if keep_strings else quote + blank(code[i + 1:end - 1]) + (code[end - 1] if end - 1 > i else ""))
            i = end
        else:
            out.append(ch)
            i += 1
    return "".join(out)


def line_of(code, index):
    return code.count("\n", 0, index) + 1


# ------------------------------------------------------------------ project

class Module:
    def __init__(self, name, kind, path, slice_name):
        self.name, self.kind, self.path, self.slice = name, kind, path, slice_name
        self.files = []

    def label(self):
        return f"{self.name} ({self.kind})"


class Project:
    def __init__(self, root, config):
        self.root = root
        self.config = config
        self.language = config.get("language")
        if self.language not in DEFAULT_EXTENSIONS:
            raise ConfigError("architecture.json: 'language' must be swift, go, or typescript")
        self.extensions = tuple(config.get("sourceExtensions", DEFAULT_EXTENSIONS[self.language]))
        self.kinds = config.get("kinds", {})
        self.modules = self._discover_modules()
        self.by_name = {m.name: m for m in self.modules}
        self.go_module = self._read_go_module() if self.language == "go" else None
        self.production_files, self.test_files, self.unmapped = self._collect_files()

    def _discover_modules(self):
        modules, seen = [], set()
        for spec in self.config.get("modules", []):
            for path, captures in self._expand(spec["path"]):
                if path in seen:
                    continue
                seen.add(path)
                kind = spec["kind"]
                if kind not in self.kinds:
                    raise ConfigError(f"architecture.json: module '{spec['path']}' uses unknown kind '{kind}'")
                name = self._fill(spec.get("name", path), captures)
                slice_name = self._fill(spec["slice"], captures) if "slice" in spec else None
                modules.append(Module(name, kind, path, slice_name))
        return modules

    @staticmethod
    def _fill(template, captures):
        for index, value in enumerate(captures, start=1):
            template = template.replace("{%d}" % index, value)
        return template

    def _expand(self, pattern):
        segments = [s for s in pattern.strip("/").split("/") if s and s != "."]
        results = [("", [])]
        for segment in segments:
            next_results = []
            for base, captures in results:
                directory = os.path.join(self.root, base)
                if "*" in segment:
                    if not os.path.isdir(directory):
                        continue
                    regex = glob_to_regex(segment)
                    for entry in sorted(os.listdir(directory)):
                        if entry.startswith(".") or entry in SKIP_DIRS:
                            continue
                        if os.path.isdir(os.path.join(directory, entry)) and regex.match(entry):
                            next_results.append((os.path.join(base, entry) if base else entry, captures + [entry]))
                else:
                    candidate = os.path.join(base, segment) if base else segment
                    if os.path.isdir(os.path.join(self.root, candidate)):
                        next_results.append((candidate, captures))
            results = next_results
        return [(p.replace(os.sep, "/"), c) for p, c in results]

    def _read_go_module(self):
        go_mod = os.path.join(self.root, self.config.get("goModFile", "go.mod"))
        try:
            with open(go_mod, encoding="utf-8") as f:
                for line in f:
                    if line.startswith("module "):
                        return line.split()[1].strip()
        except OSError:
            pass
        raise ConfigError("architecture.json: could not read the module path from go.mod")

    def module_for_path(self, path):
        """The module with the longest path that contains `path`. A module at the root ('.') contains everything."""
        best = None
        for module in self.modules:
            if module.path == "" or path == module.path or path.startswith(module.path + "/"):
                if best is None or len(module.path) > len(best.path):
                    best = module
        return best

    def _collect_files(self):
        roots = self.config.get("sourceRoots", ["."])
        tests = self.config.get("testFiles", [])
        ignore = self.config.get("ignore", [])
        production, test, unmapped = [], [], []
        for root in roots:
            for directory, dirs, files in os.walk(os.path.join(self.root, root)):
                dirs[:] = sorted(d for d in dirs if d not in SKIP_DIRS and not d.startswith("."))
                for file in sorted(files):
                    if not file.endswith(self.extensions) or file.endswith(".d.ts"):
                        continue
                    rel = os.path.relpath(os.path.join(directory, file), self.root).replace(os.sep, "/")
                    if matches_any(rel, ignore):
                        continue
                    if matches_any(rel, tests):
                        test.append(rel)
                        continue
                    module = self.module_for_path(rel)
                    if module is None:
                        unmapped.append(rel)
                    else:
                        module.files.append(rel)
                        production.append(rel)
        return production, test, unmapped

    def read(self, rel):
        with open(os.path.join(self.root, rel), encoding="utf-8", errors="replace") as f:
            return f.read()

    def glossary_terms(self):
        path = os.path.join(self.root, self.config.get("glossary", "GLOSSARY.md"))
        if not os.path.exists(path):
            return []
        terms = []
        with open(path, encoding="utf-8") as f:
            for line in f:
                if line.startswith("## "):
                    words = re.findall(r"[A-Za-z0-9]+", line[3:])
                    if words:
                        terms.append("".join(w[:1].upper() + w[1:] for w in words))
        return terms

    # ----------------------------------------------------------- imports

    def imports(self, rel, code):
        stripped = strip_source(code, self.language, keep_strings=True)
        if self.language == "swift":
            regex = re.compile(
                r"^[ \t]*(?:@\w+(?:\([^)]*\))?[ \t]+)*(?:(?:public|package|internal|fileprivate|private)[ \t]+)?"
                r"import[ \t]+(?:(?:typealias|struct|class|enum|protocol|let|var|func)[ \t]+)?([A-Za-z_]\w*)",
                re.M)
            return [(m.group(1), line_of(stripped, m.start(1))) for m in regex.finditer(stripped)]
        if self.language == "go":
            found = []
            for m in re.finditer(r'^[ \t]*import[ \t]+(?:[\w.]+[ \t]+)?"([^"]+)"', stripped, re.M):
                found.append((m.group(1), line_of(stripped, m.start(1))))
            for block in re.finditer(r"^[ \t]*import[ \t]*\((.*?)\)", stripped, re.M | re.S):
                for m in re.finditer(r'(?:[\w.]+[ \t]+)?"([^"]+)"', block.group(1)):
                    found.append((m.group(1), line_of(stripped, block.start(1) + m.start(1))))
            return found
        found = []
        for regex in (r"\bfrom\s+['\"]([^'\"]+)['\"]", r"\bimport\s+['\"]([^'\"]+)['\"]",
                      r"\bimport\s*\(\s*['\"]([^'\"]+)['\"]\s*\)", r"\brequire\s*\(\s*['\"]([^'\"]+)['\"]\s*\)"):
            for m in re.finditer(regex, stripped):
                found.append((m.group(1), line_of(stripped, m.start(1))))
        return found

    def resolve(self, rel, target):
        """Returns ('module', Module) for project imports, ('external', name) otherwise, ('missing', path)."""
        if self.language == "swift":
            module = self.by_name.get(target)
            return ("module", module) if module else ("external", target)
        if self.language == "go":
            prefix = self.go_module + "/"
            if target == self.go_module:
                module = self.module_for_path("")
                return ("module", module) if module else ("missing", ".")
            if target.startswith(prefix):
                module = self.module_for_path(target[len(prefix):])
                return ("module", module) if module else ("missing", target[len(prefix):])
            return ("external", target)
        aliases = self.config.get("typescript", {}).get("aliases", {})
        for alias, replacement in aliases.items():
            if target.startswith(alias):
                target = "./" + os.path.relpath(replacement + target[len(alias):], os.path.dirname(rel))
                break
        if target.startswith("."):
            resolved = os.path.normpath(os.path.join(os.path.dirname(rel), target)).replace(os.sep, "/")
            module = self.module_for_path(resolved)
            return ("module", module) if module else ("missing", resolved)
        return ("external", target)


# --------------------------------------------------------------- violations

class Violation:
    def __init__(self, rule, key, file, line, message, fix):
        self.rule, self.key, self.file, self.line, self.message, self.fix = rule, key, file, line, message, fix

    def render(self):
        where = f"{self.file}:{self.line}" if self.line else (self.file or "")
        text = f"{self.rule:<13} {where}\n  {self.message}"
        if self.fix:
            text += f"\n  Fix: {self.fix}"
        return text


SOURCE_FIXES = {
    "domain": "The domain depends only on utilities. When it needs something from outside, "
              "declare a port in the domain and let an adapter implement it.",
    "utilities": "Utilities depend on nothing in the project.",
    "shared-ui": "Shared UI depends only on utilities. Pass plain values in instead of domain types.",
    "adapter": "Adapters depend only on the domain and utilities. The composition root connects them to the rest.",
    "test-support": "Test support depends only on the domain and utilities.",
}
TARGET_FIXES = {
    "adapter": "Depend on a port declared in the domain. The composition root passes the adapter in.",
    "composition": "Nothing imports the composition root. Receive what you need through your constructor.",
    "test-support": "Only test code imports test support.",
    "ui": "Slices never depend on each other. Move shared types and rules into the domain.",
    "usecases": "Use cases are never shared between slices. Write the use case in this slice, "
                "and move shared rules into the domain.",
}


def import_fix(source, target):
    if target.kind in ("composition", "test-support"):
        return TARGET_FIXES[target.kind]
    return SOURCE_FIXES.get(source.kind) or TARGET_FIXES.get(target.kind) or f"See the dependency rule in {DOCS}."


def describe_allowed(kind_config):
    allowed = kind_config.get("mayImport", [])
    if not allowed:
        return "nothing in the project"
    if "*" in allowed:
        return "every module except test support"
    return ", ".join("its own slice's " + a.split("@")[0] if a.endswith("@same-slice") else a for a in allowed)


def import_allowed(project, source, target):
    if target is source:
        return True
    for entry in project.kinds[source.kind].get("mayImport", []):
        if entry == "*":
            if target.kind != "test-support":
                return True
        elif entry.endswith("@same-slice"):
            if target.kind == entry.split("@")[0] and target.slice is not None and target.slice == source.slice:
                return True
        elif entry == target.kind:
            return True
    return False


def expand_sets(project, patterns):
    """Replaces '@name' entries with the patterns listed under externalSets.name."""
    expanded = []
    for pattern in patterns:
        if pattern.startswith("@"):
            expanded.extend(project.config.get("externalSets", {}).get(pattern[1:], []))
        else:
            expanded.append(pattern)
    return expanded


def external_allowed(project, kind_config, name):
    rules = kind_config.get("external", {})
    def match(pattern):
        if pattern == "std" and project.language == "go":
            return "." not in name.split("/")[0]
        return name_pattern_matches(name, pattern)
    allow = expand_sets(project, rules.get("allow", []))
    deny = expand_sets(project, rules.get("deny", []))
    return any(match(p) for p in allow) and not any(match(p) for p in deny)


def check(project):
    violations = []
    terms = project.glossary_terms()
    graph = defaultdict(set)
    call_sets = project.config.get("forbiddenCallSets", {})

    for rel in project.unmapped:
        violations.append(Violation(
            "unmapped", f"unmapped:{rel}", rel, None,
            "This file belongs to no module in architecture.json.",
            "Move it into an existing module, or create a slice with `python3 tools/new.py slice <Name>`."))

    for module in project.modules:
        kind_config = project.kinds[module.kind]
        calls = []
        for entry in kind_config.get("forbiddenCalls", []):
            calls.extend(call_sets.get(entry[1:], []) if isinstance(entry, str) and entry.startswith("@") else [entry])
        banned_terms = terms if kind_config.get("bannedTerms") == "glossary" else kind_config.get("bannedTerms", [])
        term_regexes = [(t, re.compile(re.escape(t) + "|\\b" + re.escape(t[:1].lower() + t[1:]))) for t in banned_terms]
        composition = kind_config.get("composition")
        total_lines = 0

        for rel in module.files:
            code = project.read(rel)
            total_lines += code.count("\n") + 1
            clean = strip_source(code, project.language, keep_strings=False)

            for target, line in project.imports(rel, code):
                kind_of, value = project.resolve(rel, target)
                if kind_of == "module":
                    if value is not module:
                        graph[module.name].add(value.name)
                    if not import_allowed(project, module, value):
                        violations.append(Violation(
                            "import", f"import:{module.name}->{value.name}", rel, line,
                            f"{module.label()} imports {value.label()}. A {module.kind} module may import {describe_allowed(kind_config)}.",
                            import_fix(module, value)))
                elif kind_of == "external":
                    if not external_allowed(project, kind_config, value):
                        violations.append(Violation(
                            "external", f"external:{module.name}->{value}", rel, line,
                            f"{module.label()} imports '{value}', which its kind does not allow.",
                            "Put code that needs this framework or library into an adapter behind a port, "
                            "or into the layer that architecture.json allows it in."))

            for call in calls:
                for m in re.finditer(call["pattern"], clean):
                    violations.append(Violation(
                        "call", f"call:{module.name}:{call['pattern']}", rel, line_of(clean, m.start()),
                        f"{module.label()} uses `{m.group(0).strip()}`.", call.get("fix", "")))

            for term, regex in term_regexes:
                for number in sorted({line_of(clean, m.start()) for m in regex.finditer(clean)}):
                    violations.append(Violation(
                        "term", f"term:{module.name}:{term}", rel, number,
                        f"{module.label()} has no business meaning, but uses the glossary term '{term}'.",
                        "Move business code into the domain or into a slice."))

            if composition:
                inner = rel[len(module.path) + 1:] if module.path else rel
                if not matches_any(inner, composition.get("allowedFiles", ["**"])):
                    violations.append(Violation(
                        "composition", f"composition-file:{rel}", rel, None,
                        f"{module.label()} holds wiring only, in the files that architecture.json lists.",
                        "Put feature code into a slice: `python3 tools/new.py slice <Name>`."))
                if not matches_any(inner, composition.get("controlFlowAllowedIn", [])):
                    allowed_lines = [re.compile(p) for p in composition.get("controlFlowAllowedPatterns", [])]
                    keywords = "|".join(CONTROL_FLOW[project.language])
                    clean_lines = clean.split("\n")
                    for m in re.finditer(r"(?<![#\w.])\b(" + keywords + r")\b", clean):
                        number = line_of(clean, m.start())
                        if any(p.search(clean_lines[number - 1]) for p in allowed_lines):
                            continue
                        violations.append(Violation(
                            "control-flow", f"control-flow:{module.name}", rel, number,
                            f"{module.label()} uses `{m.group(1)}`. The composition root holds straight-line wiring.",
                            "Move the decision into the domain or into a slice's use case."))

        if composition and total_lines > composition.get("maxLines", 10**9):
            violations.append(Violation(
                "composition", f"composition-size:{module.name}", module.path, None,
                f"{module.label()} has {total_lines} lines, and the limit is {composition['maxLines']}.",
                "The composition root holds wiring only. Move feature code into a slice."))

    for cycle in find_cycles(graph):
        names = " -> ".join(cycle + [cycle[0]])
        violations.append(Violation(
            "cycle", "cycle:" + ",".join(sorted(cycle)), None, None,
            f"The modules form a cycle: {names}.", "Move the shared part down into the domain or utilities."))
    return violations


def find_cycles(graph):
    index, low, stack, on_stack, result, counter = {}, {}, [], set(), [], [0]

    def connect(node):
        index[node] = low[node] = counter[0]
        counter[0] += 1
        stack.append(node)
        on_stack.add(node)
        for nxt in sorted(graph.get(node, ())):
            if nxt not in index:
                connect(nxt)
                low[node] = min(low[node], low[nxt])
            elif nxt in on_stack:
                low[node] = min(low[node], index[nxt])
        if low[node] == index[node]:
            component = []
            while True:
                item = stack.pop()
                on_stack.discard(item)
                component.append(item)
                if item == node:
                    break
            if len(component) > 1:
                result.append(sorted(component))

    for node in sorted(graph):
        if node not in index:
            connect(node)
    return result


# --------------------------------------------------------------------- main

def module_map(project):
    lines = ["# Module map", "", "Generated by `python3 tools/archcheck.py --module-map`. Do not edit by hand.", ""]
    for kind in project.kinds:
        members = [m for m in project.modules if m.kind == kind]
        if not members:
            continue
        lines.append(f"## {kind}")
        lines.append("")
        allowed = describe_allowed(project.kinds[kind])
        lines.append(f"May import: {allowed}.")
        lines.append("")
        for m in sorted(members, key=lambda m: m.name):
            slice_note = f", slice {m.slice}" if m.slice else ""
            lines.append(f"- `{m.name}`{slice_note}: `{m.path}`")
        lines.append("")
    return "\n".join(lines)


def main():
    parser = argparse.ArgumentParser(description="Check the module graph against architecture.json.")
    parser.add_argument("--config", default="architecture.json")
    parser.add_argument("--baseline", default="architecture-baseline.json")
    parser.add_argument("--write-baseline", action="store_true", help="accept all current violations")
    parser.add_argument("--module-map", metavar="FILE", help="write the module map as Markdown and exit")
    args = parser.parse_args()

    root = os.path.dirname(os.path.abspath(args.config))
    try:
        project = Project(root, load_json_with_comments(args.config))
    except (ConfigError, OSError) as error:
        print(f"Architecture check: {error}", file=sys.stderr)
        return 2

    if args.module_map:
        with open(os.path.join(root, args.module_map), "w", encoding="utf-8") as f:
            f.write(module_map(project) + "\n")
        return 0

    violations = check(project)
    counts = defaultdict(int)
    for v in violations:
        counts[v.key] += 1
    baseline_path = os.path.join(root, args.baseline)

    if args.write_baseline:
        with open(baseline_path, "w", encoding="utf-8") as f:
            json.dump(dict(sorted(counts.items())), f, indent=2)
            f.write("\n")
        print(f"Architecture check: wrote {len(violations)} accepted violations to {args.baseline}.")
        return 0

    baseline = {}
    if os.path.exists(baseline_path):
        baseline = load_json_with_comments(baseline_path)
    new_keys = {k for k, c in counts.items() if c > baseline.get(k, 0)}
    new = [v for v in violations if v.key in new_keys]
    stale = sorted(k for k, c in baseline.items() if counts.get(k, 0) < c)
    accepted = len(violations) - len(new)

    summary = f"{len(project.modules)} modules, {len(project.production_files)} source files"
    if new:
        print(f"Architecture check: {len(new)} new violations ({summary}).\n")
        for v in new:
            allowed = baseline.get(v.key, 0)
            print(v.render() + (f"\n  The baseline accepts {allowed} of these." if allowed else "") + "\n")
        print(f"The dependency rule and the module kinds are explained in {DOCS}.")
    else:
        print(f"Architecture check passed ({summary}" + (f", {accepted} accepted by the baseline" if accepted else "") + ").")
    if stale:
        print(f"\nThe baseline accepts more violations than exist for {len(stale)} entries. "
              "Run `python3 tools/archcheck.py --write-baseline` to lower it.")
    return 1 if new else 0


if __name__ == "__main__":
    sys.exit(main())
