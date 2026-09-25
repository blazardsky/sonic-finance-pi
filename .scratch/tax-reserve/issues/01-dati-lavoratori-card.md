# 01: "Dati lavoratori" card in Impostazioni

**What to build:** A new "Dati lavoratori" card in Impostazioni, with its own save button like the password card. It holds a switch "C'è almeno un lavoratore autonomo in famiglia" which, when on, shows the fallback tax percentage (default 33) and "mesi in cui si pagano le tasse" as twelve shadcn Toggle month buttons; and a switch "Mensilità aggiuntive" which, when on, shows "mesi con mensilità aggiuntiva" as twelve month toggles. All five values are stored as settings (no new table, no schema change) and carried on the existing settings payload; saving the card sends only its own fields and leaves every other setting untouched. Dependent fields are hidden, not cleared, while their switch is off.

**Blocked by:** None (can start immediately)

**Status:** ready-for-agent

- [ ] Defaults on a fresh database: both switches off, fallback 33%, no tax months, no bonus months
- [ ] The five values round-trip through the settings API; month sets come back sorted and without duplicates
- [ ] A partial update with only these fields leaves the lists, Target, Goal, starting balance, Net worth target and Spending intent untouched, and the reverse
- [ ] A month outside 1–12 or a percentage outside 0–100 is refused
- [ ] The card shows the percentage and tax months only with the self-employed switch on, and the bonus months only with the extra-paycheck switch on; turning a switch off keeps what was entered
- [ ] Type-check passes; manual look at the card
