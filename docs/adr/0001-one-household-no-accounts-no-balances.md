# One household, no accounts, no balances between people

This app runs on a Raspberry Pi in one home and tracks that home's money. There is no Household entity, no tenant column, and no user accounts: the device *is* the household. Payer is a label chosen from a short configured list ("Person 1", "Both", "Someone else"), not an identity — the app never computes who owes whom.

We considered Splitwise-style settling-up and rejected it: it requires shares, splits, and settlement records, and this household does not need to know who is in debt to whom. Recording who paid is memory, not arithmetic.

## Consequences

Adding real multi-user support later is not additive — it means a user model, auth per person, and revisiting Payer everywhere. That is the price of a model with no dead weight in it.
