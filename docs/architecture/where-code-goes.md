# Where new code goes

Use this page when you add code and are not sure which module it belongs in. `docs/ARCHITECTURE.md` explains why each kind of module exists.

## Pick the module

Ask these questions in order. The first yes decides.

1. Does it decide something by business rules, or compute a business value? A domain rule.
2. Does it describe a business concept? The domain model, with its name in `GLOSSARY.md`.
3. Does it need something from outside the process? A port in the domain and an adapter.
4. Must it happen on every write through a port? Middleware in the domain, applied in the composition root.
5. Does it carry out an operation for one capability? A use case in that capability's slice.
6. Is it a new capability? A new slice, in the shape of the reference slice.
7. Does it show output or receive input? The UI module of its slice, or shared UI when several slices use it and it has no business meaning.
8. Does it create objects, read configuration, or start the process? The composition root.
9. Is it a helper with no business meaning? Utilities.
10. Does it change or check the code instead of running in the product, such as a codemod or a generator? A script in `scripts/`. A check that `./check` runs goes into `checks/`.

In a repository with several apps, ask two questions before the list:

- Does another app in the repository need the same rule or model? The shared domain, without ports.
- Does it arrange the UI of several slices on one page, with no data or decisions of its own? A screen.

## Reach what the module may not import

- If the code needs an outside capability, add a port and pass the adapter in through the composition root.
- If it needs something from another slice, move the shared part into the domain.
- If a use case needs data in a form that only an adapter knows, map it to domain types in the adapter.
