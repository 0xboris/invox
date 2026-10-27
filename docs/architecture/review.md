# Architecture review

Every review of a change in this repository answers these questions, in addition to its own lenses. They cover the judgment calls in `docs/ARCHITECTURE.md` that `./check` cannot make. `AGENTS.md` points every reviewer here, people and agents alike, and tells agents to pass this file to the reviewers of `/pstack:interrogate`.

Read the diff against the default branch, such as `git diff main...HEAD`, and the uncommitted changes. For each problem, quote the file and line and recommend a fix.

1. Does a use case make a business decision? Any `if` or `switch` on business meaning belongs in a domain rule that the use case calls.
2. Is the same business rule written in two places, for example in two slices, or in a slice and the domain?
3. Does a step that must happen on every path live in a single use case, where the next path will miss it? It belongs in a domain rule, a port method, or middleware.
4. Does middleware decide something that depends on intent, such as manual entry versus import? Intent-dependent reactions belong in the use case or in per-slice composition. Does the change remove mandatory middleware from the composition root, or take it away from a slice?
5. Is each new port named for what the use case needs, with domain types in its signatures and no storage, wire, or SDK types? Is it declared in a file that `portFiles` in `architecture.json` lists? The check cannot keep UI away from a port it does not know.
6. Does a new capability have its own slice, or was it added to an existing slice that does something else?
7. Does a change to a stored format or a wire contract only add things? Renamed or removed fields need a migration path, and unknown enumeration values need an unknown case. Does each new stored format have a committed sample and a test that loads it?
8. Does each new behavior have proof that it works in the running app, and a test where a cheap one can fail without the change? Tests check results, errors, and state that a caller can read back, never which internal part or port method did the work.
9. Do use case tests run against the real adapter where it runs locally, and replace it only when it is remote, slow, or nondeterministic, or to make it fail?
10. Are business concepts named as in `GLOSSARY.md`?
11. Does the change add a layer, wrapper, port, or interface that no reason in `docs/ARCHITECTURE.md` calls for? Does it remove one that a reason calls for? A port with one adapter and a use case with one caller have such a reason, even where a general rule, such as pstack's Minimize Reader Load, would inline them.
12. Does an API inside the repository keep its old form next to the new one? Callers move and the old form goes in the same change, unless it is a stored format or a wire contract.
13. Does code outside the domain create a domain value, such as a note title, without the domain's check? In TypeScript, look for a type guard outside the domain that claims a domain type, such as `value is NoteTitle`.
