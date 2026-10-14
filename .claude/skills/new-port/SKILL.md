---
name: new-port
description: Adds a port with its shared fake, contract test, and adapter. Use when domain or use case code needs storage, the network, the time, ids, randomness, the file system, or a platform or third-party service.
---

1. Check `docs/architecture/modules.md` and the domain's ports first. Reuse a port that already covers the need, and add a method to it if the method belongs to the same capability.
2. Run `python3 tools/new.py port <Name>`. Name the port for what the use case needs, such as `BudgetStore` or `PaymentGateway`, not for the technology behind it.
3. Declare the operations with domain types only. Storage records, wire payloads, and SDK types stay inside the adapter.
4. Add each operation to the fake, with a way to make it fail, and add a case for each operation to the contract.
5. Implement the port in an adapter module. Map outside data to domain types inside the adapter, and run the contract against the real implementation in the adapter's tests.
6. Add the real implementation to the composition root's live dependencies.
7. Run `./check`.
