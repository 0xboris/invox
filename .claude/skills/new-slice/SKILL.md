---
name: new-slice
description: Creates a feature slice with the project's generator and builds the feature in the shape of the Notes reference slice. Use when a task adds a new capability, such as a new screen group, a group of endpoints, or a background job with its settings.
---

1. Pick the slice name from `GLOSSARY.md`. Add the concept to the glossary if it is new.
2. Run `python3 tools/new.py slice <Name> --usecase <FirstUseCase>`. It creates the UI module, the use case module, their tests, and the wiring file in the composition root.
3. Read the Notes slice and copy its structure: the view model or handler, the use case, and their tests.
4. Put every business decision in a domain rule and call it from the use case. Use cases only orchestrate.
5. When the use case needs something from outside the process, follow the new-port skill.
6. When another slice needs the same rule or type, move it into the domain. Slices never import each other, and use cases are never shared.
7. Wire the slice in its composition file. Decide which middleware this slice receives, and extend the composition test if the choice matters.
8. Run `./check` after each step, and finish when it passes.
