# 01: Saldo di partenza in Impostazioni

**What to build:** The Savings starting balance is set once and then left alone, so it belongs with the other household-set numbers rather than on Risparmi, where it reads as something to keep adjusting. It becomes a field in Impostazioni's main form, next to Target and Goal, saved by the same button. Risparmi loses its card entirely; Net worth target stays where it is.

**Blocked by:** None (can start immediately)

**Status:** ready-for-agent

- [x] Impostazioni's main form has a "Saldo di partenza" amount field, prefilled from the saved value, with the existing hint ("Il risparmio accumulato prima di usare l'app.")
- [x] Saving the form stores the starting balance alongside payers, payment methods, Target, Goal and Spending intent, without disturbing any of them
- [x] An invalid amount is rejected with the same message Target and Goal use
- [x] Risparmi no longer shows the starting balance card; the Net worth target card is unchanged
- [x] Savings (total and combined) still include the starting balance
- [x] Test: the starting balance round-trips through the settings payload and leaves every other setting unchanged
