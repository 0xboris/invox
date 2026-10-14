#!/usr/bin/env python3
"""Creates projects and adds slices, use cases, and ports in the shape that
docs/ARCHITECTURE.md describes.

From the kit (creates or adopts a project):
  python3 tools/new.py init swift|go|typescript DEST [--name AppName] [--module example.com/app] [--owner @you]
  python3 tools/new.py adopt swift|go|typescript PROJECT [--owner @you]

Inside a project:
  python3 tools/new.py slice Budgets [--usecase LoadBudgets]
  python3 tools/new.py usecase Budgets CreateBudget
  python3 tools/new.py port BudgetStore

Needs Python 3.9 or later and nothing else.
"""

import argparse
import json
import os
import re
import shutil
import subprocess
import sys

LANGUAGES = ("swift", "go", "typescript")
PASCAL = re.compile(r"^[A-Z][A-Za-z0-9]*$")


def fail(message):
    print(f"new.py: {message}", file=sys.stderr)
    sys.exit(1)


# ------------------------------------------------------------------ names

def camel(name):
    return name[:1].lower() + name[1:]


def separated(name, separator):
    return re.sub(r"(?<!^)(?=[A-Z])", separator, name).lower()


def name_values(slice_name=None, use_case=None, port=None):
    values = {}
    if slice_name:
        values.update({
            "__Slice__": slice_name,
            "__sliceCamel__": camel(slice_name),
            "__slice-kebab__": separated(slice_name, "-"),
            "__slice__": slice_name.lower(),
        })
    if use_case:
        values.update({
            "__UseCase__": use_case,
            "__useCaseCamel__": camel(use_case),
            "__use_case__": separated(use_case, "_"),
        })
    if port:
        values.update({
            "__Port__": port,
            "__portCamel__": camel(port),
            "__port_snake__": separated(port, "_"),
        })
    return values


def fill(text, values):
    for key in sorted(values, key=len, reverse=True):
        text = text.replace(key, values[key])
    return text


# ------------------------------------------------------------- files

class Project:
    def __init__(self, root):
        self.root = root
        path = os.path.join(root, "architecture.json")
        if not os.path.exists(path):
            fail("run this inside a project: architecture.json was not found")
        with open(path, encoding="utf-8") as f:
            self.config = json.loads("".join(l for l in f if not l.lstrip().startswith("//")))
        self.language = self.config["language"]
        self.values = {}
        if self.language == "go":
            self.values["__GO_MODULE__"] = self._go_module()
        self.created = []

    def _go_module(self):
        with open(os.path.join(self.root, "go.mod"), encoding="utf-8") as f:
            for line in f:
                if line.startswith("module "):
                    return line.split()[1]
        fail("go.mod has no module line")

    def path(self, rel):
        return os.path.join(self.root, rel)

    def write(self, rel, text, values):
        target = self.path(fill(rel, values))
        if os.path.exists(target):
            fail(f"{os.path.relpath(target, self.root)} already exists")
        os.makedirs(os.path.dirname(target), exist_ok=True)
        with open(target, "w", encoding="utf-8") as f:
            f.write(fill(text, {**self.values, **values}))
        self.created.append(os.path.relpath(target, self.root))

    def insert_before(self, rel, marker, text, values):
        target = self.path(rel)
        with open(target, encoding="utf-8") as f:
            content = f.read()
        index = content.find(marker)
        if index == -1:
            fail(f"{rel} has no '{marker}' line. Add the code by hand, or restore the marker.")
        line_start = content.rfind("\n", 0, index) + 1
        content = content[:line_start] + fill(text, {**self.values, **values}) + content[line_start:]
        with open(target, "w", encoding="utf-8") as f:
            f.write(content)
        self.created.append(f"{rel} (updated)")

    def finish(self, next_steps):
        if self.language == "go" and shutil.which("gofmt"):
            go_files = [self.path(p) for p in self.created if p.endswith(".go")]
            if go_files:
                subprocess.run(["gofmt", "-w", *go_files], check=False)
        print("Created:")
        for item in self.created:
            print(f"  {item}")
        print("\nNext:")
        for step in next_steps:
            print(f"  - {step}")


# --------------------------------------------------------- swift templates

SWIFT_PACKAGE = "Packages/AppModules"

SWIFT_USE_CASE = """import Domain

/// Orchestration only: load through ports, let domain rules decide, save through ports.
public struct __UseCase__: Sendable {
    public init() {}

    public func callAsFunction() async throws {
    }
}
"""

SWIFT_USE_CASE_TEST = """import __Slice__UseCases
import Testing

@Test func __useCaseCamel__Completes() async throws {
    try await __UseCase__()()
}
"""

SWIFT_VIEW_MODEL = """import __Slice__UseCases
import Observation

@MainActor
@Observable
public final class __Slice__ViewModel {
    public enum State: Equatable {
        case loading
        case loaded
        case failed
    }

    public private(set) var state: State = .loading

    private let __useCaseCamel__: __UseCase__

    public init(__useCaseCamel__: __UseCase__) {
        self.__useCaseCamel__ = __useCaseCamel__
    }

    public func load() async {
        do {
            try await __useCaseCamel__()
            state = .loaded
        } catch {
            state = .failed
        }
    }
}
"""

SWIFT_SCREEN = """import SwiftUI

public struct __Slice__Screen: View {
    @State private var viewModel: __Slice__ViewModel

    public init(viewModel: __Slice__ViewModel) {
        _viewModel = State(initialValue: viewModel)
    }

    public var body: some View {
        Group {
            switch viewModel.state {
            case .loading:
                ProgressView()
            case .loaded:
                Text("__Slice__")
            case .failed:
                Text("This could not be loaded.")
            }
        }
        .task { await viewModel.load() }
    }
}
"""

SWIFT_VIEW_MODEL_TEST = """import __Slice__UI
import __Slice__UseCases
import Testing

@MainActor
@Test func loadingShowsTheLoadedState() async {
    let viewModel = __Slice__ViewModel(__useCaseCamel__: __UseCase__())

    await viewModel.load()

    #expect(viewModel.state == .loaded)
}
"""

SWIFT_COMPOSITION = """import __Slice__UI
import __Slice__UseCases

extension AppContainer {
    public func make__Slice__ViewModel() -> __Slice__ViewModel {
        __Slice__ViewModel(__useCaseCamel__: __UseCase__())
    }
}
"""

SWIFT_TARGETS = """        .target(name: "__Slice__UseCases", dependencies: ["Domain", "Utilities"], path: "Sources/Features/__Slice__/UseCases"),
        .target(name: "__Slice__UI", dependencies: ["__Slice__UseCases", "Domain", "DesignSystem", "Utilities"], path: "Sources/Features/__Slice__/UI"),
        .testTarget(name: "__Slice__UseCasesTests", dependencies: ["__Slice__UseCases", "Domain", "TestSupport"], path: "Tests/__Slice__UseCasesTests"),
        .testTarget(name: "__Slice__UITests", dependencies: ["__Slice__UI", "__Slice__UseCases", "Domain", "TestSupport"], path: "Tests/__Slice__UITests"),
"""

SWIFT_COMPOSITION_DEPS = """                "__Slice__UI", "__Slice__UseCases",
"""

SWIFT_PORT = """/// Declare the operations that the use cases need, in domain terms and with
/// domain types. An adapter implements it, `Fake__Port__` fakes it, and
/// `__Port__Contract` checks that both behave the same.
public protocol __Port__: Sendable {
}
"""

SWIFT_FAKE = """import Domain

/// The shared fake for `__Port__`. Add failure injection for each operation.
public actor Fake__Port__: __Port__ {
    public init() {}
}
"""

SWIFT_CONTRACT = """import Domain

/// The behavior every `__Port__` must have. TestSupportTests runs it against
/// the fake; the adapter's tests run it against the real implementation.
public enum __Port__Contract {
    public struct Failure: Error, CustomStringConvertible {
        public let description: String
    }

    public static func verify(_ makeImplementation: () async throws -> any __Port__) async throws {
        _ = try await makeImplementation()
    }
}
"""

SWIFT_FAKE_TEST = """import Testing
import TestSupport

@Test func fake__Port__MeetsTheContract() async throws {
    try await __Port__Contract.verify { Fake__Port__() }
}
"""


def swift_slice(project, values):
    base = f"{SWIFT_PACKAGE}/Sources/Features/__Slice__"
    project.write(f"{base}/UseCases/__UseCase__.swift", SWIFT_USE_CASE, values)
    project.write(f"{base}/UI/__Slice__ViewModel.swift", SWIFT_VIEW_MODEL, values)
    project.write(f"{base}/UI/__Slice__Screen.swift", SWIFT_SCREEN, values)
    project.write(f"{SWIFT_PACKAGE}/Tests/__Slice__UseCasesTests/__UseCase__Tests.swift", SWIFT_USE_CASE_TEST, values)
    project.write(f"{SWIFT_PACKAGE}/Tests/__Slice__UITests/__Slice__ViewModelTests.swift", SWIFT_VIEW_MODEL_TEST, values)
    project.write(f"{SWIFT_PACKAGE}/Sources/AppComposition/AppContainer+__Slice__.swift", SWIFT_COMPOSITION, values)
    project.insert_before(f"{SWIFT_PACKAGE}/Package.swift", "// scaffold:slices", SWIFT_TARGETS, values)
    project.insert_before(f"{SWIFT_PACKAGE}/Package.swift", "// scaffold:composition", SWIFT_COMPOSITION_DEPS, values)
    return [
        f"Show {values['__Slice__']}Screen from RootView or a navigation destination, with container.make{values['__Slice__']}ViewModel().",
        "Implement the use case. Copy the structure of the Notes slice.",
    ]


def swift_use_case(project, values):
    if not os.path.isdir(project.path(fill(f"{SWIFT_PACKAGE}/Sources/Features/__Slice__/UseCases", values))):
        fail(f"there is no slice named {values['__Slice__']}")
    project.write(f"{SWIFT_PACKAGE}/Sources/Features/__Slice__/UseCases/__UseCase__.swift", SWIFT_USE_CASE, values)
    project.write(f"{SWIFT_PACKAGE}/Tests/__Slice__UseCasesTests/__UseCase__Tests.swift", SWIFT_USE_CASE_TEST, values)
    return ["Pass the use case to the view model, and construct it in AppContainer+" + values["__Slice__"] + ".swift."]


def swift_port(project, values):
    project.write(f"{SWIFT_PACKAGE}/Sources/Domain/Ports/__Port__.swift", SWIFT_PORT, values)
    project.write(f"{SWIFT_PACKAGE}/Sources/TestSupport/Fake__Port__.swift", SWIFT_FAKE, values)
    project.write(f"{SWIFT_PACKAGE}/Sources/TestSupport/__Port__Contract.swift", SWIFT_CONTRACT, values)
    project.write(f"{SWIFT_PACKAGE}/Tests/TestSupportTests/Fake__Port__Tests.swift", SWIFT_FAKE_TEST, values)
    return [
        "Declare the operations in the protocol, then add them to the fake and a case for each to the contract.",
        "Implement the port in an adapter, and run the contract against it in that adapter's tests.",
        "Add the real implementation to AppDependencies and LiveDependencies.",
    ]


# ------------------------------------------------------------ go templates

GO_USE_CASE_DOC = """// Package usecases holds the use cases of the __slice__ slice. They orchestrate
// only: the rules live in the domain, and I/O sits behind ports.
"""

GO_USE_CASE = """package usecases

import "context"

type __UseCase__ struct{}

func New__UseCase__() __UseCase__ {
	return __UseCase__{}
}

func (u __UseCase__) Run(ctx context.Context) error {
	return nil
}
"""

GO_USE_CASE_TEST = """package usecases_test

import (
	"context"
	"testing"

	"__GO_MODULE__/internal/features/__slice__/usecases"
)

func Test__UseCase__Completes(t *testing.T) {
	if err := usecases.New__UseCase__().Run(context.Background()); err != nil {
		t.Fatal(err)
	}
}
"""

GO_HANDLER = """// Package ui holds the HTTP handlers of the __slice__ slice.
package ui

import (
	"net/http"

	"__GO_MODULE__/internal/features/__slice__/usecases"
	"__GO_MODULE__/internal/sharedui/httpjson"
	"__GO_MODULE__/internal/utilities"
)

type Handler struct {
	__useCaseCamel__ usecases.__UseCase__
	logger utilities.Logger
}

func NewHandler(__useCaseCamel__ usecases.__UseCase__, logger utilities.Logger) Handler {
	return Handler{__useCaseCamel__: __useCaseCamel__, logger: logger}
}

func (h Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /__slice__", h.get)
}

func (h Handler) get(w http.ResponseWriter, r *http.Request) {
	if err := h.__useCaseCamel__.Run(r.Context()); err != nil {
		h.logger.Error("__slice__ request failed", "error", err)
		httpjson.Error(w, http.StatusInternalServerError, "request_failed", "The request could not be completed.")
		return
	}
	httpjson.Write(w, http.StatusOK, map[string]string{})
}
"""

GO_HANDLER_TEST = """package ui_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"__GO_MODULE__/internal/features/__slice__/ui"
	"__GO_MODULE__/internal/features/__slice__/usecases"
	"__GO_MODULE__/internal/testsupport"
)

func TestGetReturnsOK(t *testing.T) {
	mux := http.NewServeMux()
	ui.NewHandler(usecases.New__UseCase__(), &testsupport.RecordingLogger{}).Register(mux)

	response := httptest.NewRecorder()
	mux.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/__slice__", nil))

	if response.Code != http.StatusOK {
		t.Fatalf("got %d %s", response.Code, response.Body)
	}
}
"""

GO_WIRE = """package app

import (
	__slice__ui "__GO_MODULE__/internal/features/__slice__/ui"
	__slice__usecases "__GO_MODULE__/internal/features/__slice__/usecases"
)

func (a *App) wire__Slice__() {
	__slice__ui.NewHandler(__slice__usecases.New__UseCase__(), a.deps.Logger).Register(a.mux)
}
"""

GO_WIRE_CALL = """	app.wire__Slice__()
"""

GO_PORT = """package domain

// __Port__ declares what the use cases need, in domain terms and with domain
// types. An adapter implements it, testsupport.Fake__Port__ fakes it, and
// testsupport.__Port__Contract checks that both behave the same.
type __Port__ interface {
}
"""

GO_FAKE = """package testsupport

import "__GO_MODULE__/internal/domain"

// Fake__Port__ is the shared fake for domain.__Port__. Add failure injection
// for each operation.
type Fake__Port__ struct{}

var _ domain.__Port__ = (*Fake__Port__)(nil)

func NewFake__Port__() *Fake__Port__ {
	return &Fake__Port__{}
}
"""

GO_CONTRACT = """package testsupport

import (
	"testing"

	"__GO_MODULE__/internal/domain"
)

// __Port__Contract checks the behavior that every domain.__Port__ must have.
// The fake runs it in its test, and the adapter runs it in its own tests.
func __Port__Contract(t *testing.T, newImplementation func(t *testing.T) domain.__Port__) {
	t.Run("can be created", func(t *testing.T) {
		if newImplementation(t) == nil {
			t.Fatal("got nil")
		}
	})
}
"""

GO_FAKE_TEST = """package testsupport_test

import (
	"testing"

	"__GO_MODULE__/internal/domain"
	"__GO_MODULE__/internal/testsupport"
)

func TestFake__Port__MeetsTheContract(t *testing.T) {
	testsupport.__Port__Contract(t, func(t *testing.T) domain.__Port__ {
		return testsupport.NewFake__Port__()
	})
}
"""


def go_slice(project, values):
    base = "internal/features/__slice__"
    project.write(f"{base}/usecases/__use_case__.go", GO_USE_CASE_DOC + GO_USE_CASE, values)
    project.write(f"{base}/usecases/__use_case___test.go", GO_USE_CASE_TEST, values)
    project.write(f"{base}/ui/handler.go", GO_HANDLER, values)
    project.write(f"{base}/ui/handler_test.go", GO_HANDLER_TEST, values)
    project.write("internal/app/wire___slice__.go", GO_WIRE, values)
    project.insert_before("internal/app/app.go", "// scaffold:wire", GO_WIRE_CALL, values)
    return ["Implement the use case and the handler. Copy the structure of the notes slice."]


def go_use_case(project, values):
    if not os.path.isdir(project.path(fill("internal/features/__slice__/usecases", values))):
        fail(f"there is no slice named {values['__Slice__']}")
    project.write("internal/features/__slice__/usecases/__use_case__.go", GO_USE_CASE, values)
    project.write("internal/features/__slice__/usecases/__use_case___test.go", GO_USE_CASE_TEST, values)
    return [f"Pass the use case to the handler, and construct it in internal/app/wire_{values['__slice__']}.go."]


def go_port(project, values):
    project.write("internal/domain/__port_snake__.go", GO_PORT, values)
    project.write("internal/testsupport/fake___port_snake__.go", GO_FAKE, values)
    project.write("internal/testsupport/__port_snake___contract.go", GO_CONTRACT, values)
    project.write("internal/testsupport/fake___port_snake___test.go", GO_FAKE_TEST, values)
    return [
        "Declare the methods in the interface, then add them to the fake and a case for each to the contract.",
        "Implement the port in an adapter under internal/adapters, and run the contract against it.",
        "Add the real implementation to Dependencies and LiveDependencies in internal/app.",
    ]


# ---------------------------------------------------- typescript templates

TS_USE_CASE = """/** Orchestration only: load through ports, let domain rules decide, save through ports. */
export class __UseCase__ {
  async run(): Promise<void> {}
}
"""

TS_USE_CASE_TEST = """import assert from 'node:assert/strict';
import { test } from 'node:test';
import { __UseCase__ } from './__useCaseCamel__.ts';

test('__UseCase__ completes', async () => {
  await assert.doesNotReject(new __UseCase__().run());
});
"""

TS_ROUTES = """import { sendError, sendJson, type Route } from '../../../sharedui/http.ts';
import type { Logger } from '../../../utilities/logger.ts';
import type { __UseCase__ } from '../usecases/__useCaseCamel__.ts';

export function __sliceCamel__Routes(deps: { __useCaseCamel__: __UseCase__; logger: Logger }): Route[] {
  return [
    {
      method: 'GET',
      path: '/__slice-kebab__',
      handle: async (_request, response) => {
        try {
          await deps.__useCaseCamel__.run();
          sendJson(response, 200, {});
        } catch (error) {
          deps.logger.error('__slice-kebab__ request failed', { error: String(error) });
          sendError(response, 500, 'request_failed', 'The request could not be completed.');
        }
      },
    },
  ];
}
"""

TS_ROUTES_TEST = """import assert from 'node:assert/strict';
import { test } from 'node:test';
import { createRouter } from '../../../sharedui/http.ts';
import { RecordingLogger } from '../../../testsupport/fakes.ts';
import { startTestServer } from '../../../testsupport/http.ts';
import { __UseCase__ } from '../usecases/__useCaseCamel__.ts';
import { __sliceCamel__Routes } from './__sliceCamel__Routes.ts';

test('GET /__slice-kebab__ returns ok', async () => {
  const routes = __sliceCamel__Routes({ __useCaseCamel__: new __UseCase__(), logger: new RecordingLogger() });
  const server = await startTestServer(createRouter(routes, () => undefined));
  try {
    const response = await fetch(`${server.url}/__slice-kebab__`);
    assert.equal(response.status, 200);
  } finally {
    await server.close();
  }
});
"""

TS_WIRE = """import { __sliceCamel__Routes } from '../features/__slice-kebab__/ui/__sliceCamel__Routes.ts';
import { __UseCase__ } from '../features/__slice-kebab__/usecases/__useCaseCamel__.ts';
import type { Route } from '../sharedui/http.ts';
import type { Dependencies } from './container.ts';

export function wire__Slice__(deps: Dependencies): Route[] {
  return __sliceCamel__Routes({ __useCaseCamel__: new __UseCase__(), logger: deps.logger });
}
"""

TS_WIRE_IMPORT = """import { wire__Slice__ } from './wire-__slice-kebab__.ts';
"""

TS_WIRE_ROUTES = """    ...wire__Slice__(wired),
"""

TS_PORT = """/**
 * Declare what the use cases need, in domain terms and with domain types. An
 * adapter implements it, Fake__Port__ fakes it, and __portCamel__Contract checks both.
 */
export interface __Port__ {}
"""

TS_FAKE = """import type { __Port__ } from '../domain/__portCamel__.ts';

/** The shared fake for __Port__. Add failure injection for each operation. */
export class Fake__Port__ implements __Port__ {}
"""

TS_CONTRACT = """import assert from 'node:assert/strict';
import { describe, test } from 'node:test';
import type { __Port__ } from '../domain/__portCamel__.ts';

/** The behavior every __Port__ must have. Run it against the fake and against the real adapter. */
export function __portCamel__Contract(name: string, make: () => __Port__ | Promise<__Port__>): void {
  describe(`${name} meets the __Port__ contract`, () => {
    test('can be created', async () => {
      assert.ok(await make());
    });
  });
}
"""

TS_FAKE_TEST = """import { __portCamel__Contract } from './__portCamel__Contract.ts';
import { Fake__Port__ } from './fake__Port__.ts';

__portCamel__Contract('Fake__Port__', () => new Fake__Port__());
"""


def ts_slice(project, values):
    base = "src/features/__slice-kebab__"
    project.write(f"{base}/usecases/__useCaseCamel__.ts", TS_USE_CASE, values)
    project.write(f"{base}/usecases/__useCaseCamel__.test.ts", TS_USE_CASE_TEST, values)
    project.write(f"{base}/ui/__sliceCamel__Routes.ts", TS_ROUTES, values)
    project.write(f"{base}/ui/__sliceCamel__Routes.test.ts", TS_ROUTES_TEST, values)
    project.write("src/app/wire-__slice-kebab__.ts", TS_WIRE, values)
    project.insert_before("src/app/container.ts", "// scaffold:imports", TS_WIRE_IMPORT, values)
    project.insert_before("src/app/container.ts", "// scaffold:routes", TS_WIRE_ROUTES, values)
    return ["Implement the use case and the routes. Copy the structure of the notes slice."]


def ts_use_case(project, values):
    if not os.path.isdir(project.path(fill("src/features/__slice-kebab__/usecases", values))):
        fail(f"there is no slice named {values['__Slice__']}")
    project.write("src/features/__slice-kebab__/usecases/__useCaseCamel__.ts", TS_USE_CASE, values)
    project.write("src/features/__slice-kebab__/usecases/__useCaseCamel__.test.ts", TS_USE_CASE_TEST, values)
    return [f"Pass the use case to the routes, and construct it in src/app/wire-{values['__slice-kebab__']}.ts."]


def ts_port(project, values):
    project.write("src/domain/__portCamel__.ts", TS_PORT, values)
    project.write("src/testsupport/fake__Port__.ts", TS_FAKE, values)
    project.write("src/testsupport/__portCamel__Contract.ts", TS_CONTRACT, values)
    project.write("src/testsupport/fake__Port__.test.ts", TS_FAKE_TEST, values)
    return [
        "Declare the methods in the interface, then add them to the fake and a test for each to the contract.",
        "Implement the port in an adapter under src/adapters, and run the contract against it.",
        "Add the real implementation to Dependencies in src/app/container.ts and to liveDependencies.",
    ]


GENERATORS = {
    "swift": {"slice": swift_slice, "usecase": swift_use_case, "port": swift_port},
    "go": {"slice": go_slice, "usecase": go_use_case, "port": go_port},
    "typescript": {"slice": ts_slice, "usecase": ts_use_case, "port": ts_port},
}


# ----------------------------------------------------------- init, adopt

COMPOSITION_PATHS = {
    "swift": ["/App/", "/Packages/AppModules/Sources/AppComposition/"],
    "go": ["/cmd/", "/internal/app/"],
    "typescript": ["/src/app/"],
}


def kit_root():
    root = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
    if not os.path.isdir(os.path.join(root, "templates")):
        fail("init and adopt run from the kit, which has a templates/ directory next to tools/")
    return root


def copy_tree(source, dest, values, overwrite, report):
    for directory, _, files in os.walk(source):
        for file in files:
            if file == ".DS_Store":
                continue
            rel = os.path.relpath(os.path.join(directory, file), source)
            target = os.path.join(dest, fill(rel, values))
            if os.path.exists(target) and not overwrite:
                report.append(f"kept existing {os.path.relpath(target, dest)}")
                continue
            os.makedirs(os.path.dirname(target), exist_ok=True)
            with open(os.path.join(directory, file), encoding="utf-8") as f:
                text = f.read()
            with open(target, "w", encoding="utf-8") as f:
                f.write(fill(text, values))
            shutil.copymode(os.path.join(directory, file), target)


def copy_kit_files(kit, dest, language, values, overwrite, report):
    copy_tree(os.path.join(kit, "templates", "common"), dest, values, overwrite, report)
    os.makedirs(os.path.join(dest, "tools"), exist_ok=True)
    for tool in ("archcheck.py", "new.py", "agent-stop-check.sh"):
        target = os.path.join(dest, "tools", tool)
        shutil.copy2(os.path.join(kit, "tools", tool), target)
        os.chmod(target, 0o755)
    os.makedirs(os.path.join(dest, "docs", "architecture"), exist_ok=True)
    shutil.copy2(os.path.join(kit, "docs", "ARCHITECTURE.md"), os.path.join(dest, "docs", "ARCHITECTURE.md"))
    shutil.copy2(os.path.join(kit, "docs", "architecture", f"{language}.md"),
                 os.path.join(dest, "docs", "architecture", f"{language}.md"))


def placeholder_values(language, args):
    return {
        "__LANGUAGE__": language,
        "__OWNER__": args.owner,
        "__COMPOSITION_LINES__": "\n".join(f"{path:<30}{args.owner}" for path in COMPOSITION_PATHS[language]),
        "__APP_NAME__": args.name,
        "__PROJECT_NAME__": separated(args.name, "-"),
        "__GO_MODULE__": args.module or f"example.com/{args.name.lower()}",
    }


def init(args):
    kit = kit_root()
    dest = os.path.abspath(args.dest)
    if os.path.exists(dest) and os.listdir(dest):
        fail(f"{dest} is not empty. Use `adopt` for an existing project.")
    if not PASCAL.match(args.name):
        fail("--name must be a type-style name such as ExpenseTracker")
    values = placeholder_values(args.language, args)
    report = []
    copy_tree(os.path.join(kit, "templates", args.language), dest, values, True, report)
    copy_kit_files(kit, dest, args.language, values, True, report)
    with open(os.path.join(dest, "architecture-baseline.json"), "w", encoding="utf-8") as f:
        f.write("{}\n")
    subprocess.run([sys.executable, "tools/archcheck.py", "--module-map", "docs/architecture/modules.md"], cwd=dest, check=False)
    print(f"Created a {args.language} project in {dest}.\n\nNext:")
    steps = {
        "swift": [
            f"Create an iOS app project named {args.name} in Xcode, saved in {dest}.",
            f"Delete the Swift files Xcode generated, and add App/{args.name}App.swift to the app target.",
            "Add Packages/AppModules as a local package, and link the AppComposition product to the app target.",
        ],
        "go": ["Run `go mod tidy` if you add dependencies."],
        "typescript": ["Run `npm install`."],
    }[args.language]
    steps.append("Run ./check.")
    if args.owner == "@OWNER":
        steps.append("Replace @OWNER in .github/CODEOWNERS with your GitHub user or team.")
    steps.append("Commit, and turn on branch protection with required code-owner reviews and the check workflow.")
    steps.append("Replace the Notes reference slice with your first real slice once agents have copied its patterns.")
    for step in steps:
        print(f"  - {step}")


def adopt(args):
    kit = kit_root()
    dest = os.path.abspath(args.dest)
    if not os.path.isdir(dest):
        fail(f"{dest} does not exist")
    args.name = args.name or "App"
    values = placeholder_values(args.language, args)
    if args.language == "go" and os.path.exists(os.path.join(dest, "go.mod")):
        with open(os.path.join(dest, "go.mod"), encoding="utf-8") as f:
            values["__GO_MODULE__"] = next((l.split()[1] for l in f if l.startswith("module ")), values["__GO_MODULE__"])
    report = []
    copy_kit_files(kit, dest, args.language, values, False, report)
    language_dir = os.path.join(kit, "templates", args.language)
    for rel in ("architecture.json", ".claude/rules/architecture.md"):
        target = os.path.join(dest, rel)
        if os.path.exists(target):
            report.append(f"kept existing {rel}")
            continue
        os.makedirs(os.path.dirname(target), exist_ok=True)
        with open(os.path.join(language_dir, rel), encoding="utf-8") as f:
            text = fill(f.read(), values)
        with open(target, "w", encoding="utf-8") as f:
            f.write(text)
    check_target = "check" if not os.path.exists(os.path.join(dest, "check")) else "check.kit"
    shutil.copy2(os.path.join(language_dir, "check"), os.path.join(dest, check_target))
    for line in report:
        print(line)
    print(f"\nAdded the architecture kit to {dest}.\n\nNext:")
    for step in [
        "Edit the modules in architecture.json so that every source directory maps to a kind.",
        "Run `python3 tools/archcheck.py` until no file is reported as unmapped.",
        "Run `python3 tools/archcheck.py --write-baseline` to accept the existing violations.",
        f"Wire {check_target} into your build and CI, and adjust its build and test commands to this project.",
        "If CLAUDE.md already existed, add the line `@AGENTS.md` at its top.",
        "Fill in GLOSSARY.md, then follow the adoption steps in the kit's README.",
    ]:
        print(f"  - {step}")


# ---------------------------------------------------------------- main

def main():
    parser = argparse.ArgumentParser(description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter)
    commands = parser.add_subparsers(dest="command", required=True)

    p = commands.add_parser("init", help="create a project from the kit")
    p.add_argument("language", choices=LANGUAGES)
    p.add_argument("dest")
    p.add_argument("--name", default="App", help="app or service name, such as ExpenseTracker")
    p.add_argument("--module", help="Go module path, such as github.com/you/app")
    p.add_argument("--owner", default="@OWNER", help="code owner for gate files, such as @you")

    p = commands.add_parser("adopt", help="add the kit to an existing project")
    p.add_argument("language", choices=LANGUAGES)
    p.add_argument("dest")
    p.add_argument("--name")
    p.add_argument("--module")
    p.add_argument("--owner", default="@OWNER")

    p = commands.add_parser("slice", help="add a feature slice")
    p.add_argument("name")
    p.add_argument("--usecase", help="name of the first use case (default: Load<Name>)")

    p = commands.add_parser("usecase", help="add a use case to a slice")
    p.add_argument("slice")
    p.add_argument("name")

    p = commands.add_parser("port", help="add a port with its fake and contract")
    p.add_argument("name")

    args = parser.parse_args()
    if args.command == "init":
        return init(args)
    if args.command == "adopt":
        return adopt(args)

    for value in (getattr(args, "name", None), getattr(args, "slice", None), getattr(args, "usecase", None)):
        if value and not PASCAL.match(value):
            fail(f"'{value}' must be a type-style name such as Budgets or CreateBudget")
    project = Project(os.getcwd())
    generators = GENERATORS[project.language]
    if args.command == "slice":
        values = name_values(slice_name=args.name, use_case=args.usecase or f"Load{args.name}")
        steps = generators["slice"](project, values)
    elif args.command == "usecase":
        values = name_values(slice_name=args.slice, use_case=args.name)
        steps = generators["usecase"](project, values)
    else:
        values = name_values(port=args.name)
        steps = generators["port"](project, values)
    project.finish(steps + ["Run ./check."])


if __name__ == "__main__":
    main()
