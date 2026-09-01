# 07: Items on an Expense

**What to build:** A book bought during the grocery shop stops counting as Food. Part of an Expense can be broken out under a different Category, without ever having to itemise the whole receipt.

**Blocked by:** 06

**Status:** ready-for-agent

- [ ] An Item has a name, an amount, and its own Category
- [ ] Items are optional — a €7 coffee stays a single record with no Items
- [ ] Items are partial: they never have to account for the whole Expense
- [ ] The Expense's own amount stays authoritative and is never derived from its Items — see ADR-0002
- [ ] Items summing to more than the Expense total are rejected on save with a clear message
- [ ] An Item sharing the Expense's Category is permitted; no validation against it
- [ ] Items are saved with the Expense in one request; they have no endpoints of their own
