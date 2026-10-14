# Architecture

This document describes how the code is organized and why. It gives every agent and every person who works here the same picture: which kinds of modules exist, what each holds, which may depend on which, and how a change is verified.

The language guide in `docs/architecture/` shows how each part looks in this project's language. `docs/architecture/modules.md` lists every module with its kind; the check generates it. invox has no reference slice. When you write a kind of code for the first time, copy the existing code that `docs/architecture/go.md` names.

`./check` enforces most of this document. A forbidden import, a file outside every module, or feature code in the composition root fails the check with a message that names the fix. The rest of this document covers the judgment calls that no tool can make.

## The shape

The code is layered horizontally and sliced vertically. The layers say which code may depend on which. The slices keep the code of one capability together.

```
                composition root
                 │            │
                 ▼            ▼
         feature slices     adapters
        (UI → use cases)       │
           │       │           │
           ▼       ▼           ▼
       shared UI   domain ◄────┘
           │       │
           ▼       ▼
           utilities
```

An arrow points from a module to what it may depend on. Dependencies point inward, from details toward policy: the domain is policy, while UI, storage, and frameworks are details. Policy never depends on details.

## Module kinds

- **Domain.** The business model, the rules, the ports, and the middleware. It performs no I/O and uses no UI framework, database, or SDK. It depends only on utilities.
- **Use cases.** One module per slice. A use case carries out one operation for its slice: it loads what it needs through ports, lets domain rules decide, and saves or sends the result through ports. It depends on the domain and utilities.
- **UI.** One module per slice: screens and view models in an app, request handlers in a service, commands in a tool. It turns input into use case calls and results into output. It depends on its own slice's use cases, the domain, shared UI, and utilities.
- **Shared UI.** Presentation pieces that several slices use, such as components, styles, or HTTP response helpers. It holds no business types and depends only on utilities.
- **Adapters.** Implementations of the ports: databases, files, the network, platform services, third-party SDKs, the clock, and random sources. Each adapter maps outside formats to domain types. It depends on the domain and utilities.
- **Composition root.** The entry point. It creates the adapters, applies middleware, wires every slice, reads configuration, and starts the process. It is the only code that knows which implementation stands behind each port. Nothing depends on it.
- **Utilities.** Helpers with no business meaning, such as the logging interface or a result type. They depend on nothing in the project.
- **Test support.** Shared fakes, sample data, and contract suites. Only test code depends on it.

## Feature slices

A slice is one capability of the product, such as transactions, budgets, or recurring transactions. It has a UI module and a use case module. A slice can have several entry points: screens, endpoints, and background work that the composition root triggers, such as a scheduled job.

- Slices never depend on each other. When two slices need the same type or rule, it moves into the domain.
- Use cases are never shared. Each slice writes its own use cases, even when two of them look alike. The rules they use are shared, through the domain.
- A new capability gets a new slice. Do not grow an existing slice into something it was not made for.
- Create slices by hand, as `docs/architecture/go.md` describes. `tools/new.py` targets a layout invox does not have yet.

## The domain

The domain holds four things.

- **The model.** Business types and values. A value that must follow rules is created only through a function that checks them, so the rest of the code can trust it. States that exclude each other form one closed set of cases, not flags that can contradict each other.
- **The rules.** Every business decision, as pure functions of values: same input, same output, no I/O, no clock, no randomness. This includes rules that only one path uses today, such as "a manual entry needs a category", so that all rules live in one place.
- **The ports.** Interfaces for everything the use cases need from outside the process.
- **The middleware.** Decorators that wrap a port and add a step to every call through it. They depend only on ports and rules, so they perform no I/O themselves.

Each rule exists once, under the name that `GLOSSARY.md` gives the concept. A copy of a rule drifts, and an agent that fixes one copy does not know about the other.

## Use cases orchestrate

A use case makes no business decision. If you write an `if` about business meaning in a use case, it belongs in a domain rule that the use case calls.

Orchestration may repeat across slices. A manual entry, an import, and a scheduled job may each load, validate, and save a transaction in their own use case. That is acceptable, because the knowledge sits in the rules and the rules are shared.

Watch for steps that must happen on every path. If changing a step would force every path to change the same way, it is not orchestration. It belongs in a domain rule, a port method, or middleware, where the next path cannot forget it.

## Ports and dependency inversion

When code needs something from outside the process, the domain declares a port and an adapter implements it. The composition root passes the adapter in. The call goes outward at run time, while the source dependency points inward.

- Name a port for what the use case needs, such as `TransactionStore` or `PaymentGateway`, not for the technology behind it.
- A port uses domain types. Storage records, wire payloads, and SDK types stay inside the adapter.
- Every dependency arrives through a constructor or a parameter. There are no singletons, global registries, or service locators, so an agent can see what a piece of code depends on by reading it.
- Each third-party library that performs I/O sits behind one adapter.
- Add an interface only at a module boundary or to isolate something slow or nondeterministic. Inside a module, use concrete types.
- Create adapters by hand, as `docs/architecture/go.md` describes.

## Middleware

Middleware wraps a port, and the composition root decides which slices receive it. There are two kinds.

- **Mandatory middleware** keeps data consistent or performs a technical step that every write needs, such as refreshing a widget or a search index. The composition root applies it once, when it creates the adapter, so no slice can receive the bare adapter.
- **Optional middleware** depends on intent, such as a budget alert that manual entries trigger and imports do not. The composition root adds it per slice, on top of the mandatory middleware.

Four conditions keep middleware safe.

1. The decision stays in a domain rule. The middleware calls the rule and acts on the result.
2. Writes that must succeed together happen in one port call or one transaction.
3. Middleware sees writes, not intent. When intent matters, the composition root applies the middleware only to the slices with that intent. When the decision depends on the data, such as an alert only for today's transactions, a domain rule decides.
4. The composition test covers each choice that matters. Unit tests of a slice use fakes and never see the wiring.

Before adding middleware, ask whether the data needs updating at all. A total that can be computed from stored data when it is read needs no middleware to keep it in sync.

## The composition root

The composition root holds wiring and nothing else.

- It has no control flow, apart from one configuration file and the error handling that the language requires. Decisions belong in the domain or in a use case.
- It consists of a fixed set of files: the entry point, the container, the live dependencies, the configuration, and one wiring file per slice.
- It has a size limit. Feature code that lands here fails the check.
- Configuration and the environment are read here and passed in as values.

## Boundaries

- **Parse at the edge.** Data from outside the process becomes domain types where it enters. This includes requests, responses, database rows, files, stored settings, and deep links. Inside, code trusts its types.
- **One model per boundary.** Wire types, storage types, and domain types stay separate even when they look alike. Adapters and UI map between them.
- **Data outlives code.** Data that older versions stored, and clients that are still installed, will meet the new code. Change stored formats and wire contracts by adding: add optional fields, keep old fields until nothing uses them, and map unknown enumeration values to an unknown case. Renaming or removing a stored or sent field is a deliberate decision with a migration path.
- **Errors are explicit.** No error is swallowed. An error that leaves a module tells the caller what it can do and carries its cause. Log each error once, where it is handled, with internal ids and no personal data.
- **Authorization is checked where requests enter.** Every entry point that acts for a caller checks that the caller may act on each record that the request touches.

## State and concurrency

- Prefer immutable values. Shared mutable state has one owner that serializes access, and each check happens in the same call as the action that depends on it.
- A decision based on stored state that others can change uses a transaction or a conditional write.
- A handler for messages that can arrive more than once is idempotent. Generated records get deterministic ids, so a second run creates nothing new.
- Every background task has an owner that awaits or cancels it.

## Verification

`./check` runs everything: the architecture check, the build, the type checker, and the tests. It runs the same way locally, in the agent's Stop hook, and in CI. The main branch always passes it.

- **Machines check first.** A failure at build time beats a failing test, and a failing test beats a review comment.
- **Most tests are domain tests.** Rules are pure, so their tests need no fakes and run in milliseconds.
- **Use case and UI tests use the shared fakes** from test support. There is one fake per port.
- **Contract suites keep the fakes honest.** Each port's contract runs against the fake and against the real adapter.
- **Composition tests check the wiring**, such as which middleware a slice receives.
- **Tests follow behavior.** They use a module's public interface and assert on results and state, not on internal calls. A refactor that keeps behavior keeps every test green.
- **Tests are deterministic.** They use a fake clock and fixed ids, never real time, the network, or unseeded randomness. They pass in any order. A flaky test is a bug.
- **Verification stays honest.** Every bug fix starts with a test that reproduces the bug. No check is made to pass by deleting, skipping, or loosening a test, or by changing a gate file.

## Where new code goes

Ask these questions in order. The first yes decides.

1. Does it decide something by business rules, or compute a business value? A domain rule.
2. Does it describe a business concept? The domain model, with its name in `GLOSSARY.md`.
3. Does it need something from outside the process? A port in the domain and an adapter.
4. Must it happen on every write through a port? Middleware in the domain, applied in the composition root.
5. Does it carry out an operation for one capability? A use case in that capability's slice.
6. Is it a new capability? A new slice.
7. Does it show output or receive input? The UI module of its slice, or shared UI when several slices use it and it has no business meaning.
8. Does it create objects, read configuration, or start the process? The composition root.
9. Is it a helper with no business meaning? Utilities.

When the code needs something its module may not import:

- If it needs an outside capability, add a port and pass the adapter in through the composition root.
- If it needs something from another slice, move the shared part into the domain.
- If a use case needs data in a form that only an adapter knows, map it to domain types in the adapter.

## How this is enforced

- **The architecture check** (`tools/archcheck.py`, configured in `architecture.json`) maps every source file to a module and checks these things:
  - imports between modules
  - frameworks and libraries per kind
  - I/O calls in modules that must not perform I/O
  - glossary terms in utilities and shared UI
  - the shape of the composition root
  - cycles
- **The baseline** (`architecture-baseline.json`) records violations that existed before the rules were adopted. The check fails only when a change adds a violation. When code is fixed, the baseline is lowered.
- **Gate files** define the checks: `architecture.json`, the baseline, `check`, `tools/`, `.claude/`, `.github/`, and the instruction files. A code-owner rule requires the owner's review for every change to them. Never change a gate file to make a check pass.
- **The Stop hook** runs `./check` before an agent finishes a task that changed files.
- **The architecture reviewer** subagent checks the judgment calls in this document before a pull request is opened.

## Changing code safely

- Work in small steps that each leave the check green. Keep refactoring and behavior changes in separate steps.
- Change a contract by expanding first and contracting later: add the new form, move the callers, then remove the old form.
- Build what the task needs and no more. Add options, modules, and dependencies when a need exists.
- When existing code breaks these rules, write new code by the rules instead of copying the old code, and fix the old code in a separate change.
