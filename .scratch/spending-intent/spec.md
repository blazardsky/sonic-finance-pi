Status: ready-for-agent

# Spending Intent

## Problem Statement

The household's Category breakdown answers "where did the money go" but not "did we need to spend it" — a well-thought-out purchase, an impulse buy, and rent all land under whatever Category they happen to share, with no way to see the household's own gut judgment about them. There's also no way to distinguish an expense made *because* it was necessary from one made because it was wanted, and no way to separate a wanted purchase the household stands behind from one it regrets.

## Solution

An optional, off-by-default classification — **Spending intent** — that a household member can attach to an Expense or Recurring expense: first, whether it was a **Necessity** or a **Desire**; only a Desire can then optionally be further marked **Wise** or **Bullshit** ("necessity" isn't judged on wisdom — needing something isn't a question of whether buying it was a good call). A Category can carry its own default Spending intent, which seeds a new Expense's own value; from then on the Expense's value is its own, independent of later changes to the Category's default.

The whole feature sits behind a single settings toggle, off by default. Turning it off hides the UI; data already recorded is never touched or lost. This ships on its own branch as an experiment — not merged to `main` until the household has actually tried it.

Two views land on the year review (`YearlyReport.tsx`), desktop only, no mobile fallback:
1. A line chart — four lines (Necessity %, Desire %, Wise %, Bullshit %) across the year's months.
2. A soft, "by feel" blurred diagram — three overlapping blobs (Necessity, Desire+Wise, Desire+Bullshit) sized roughly by the year's € total in each.

## User Stories

**Settings toggle**

1. As a household member, I want a single on/off switch for Spending intent in settings, off by default, so trying the feature costs nothing and doesn't clutter the app for the other household member who may not use it.
2. As a household member, I want turning the switch off to hide every Spending intent control and view, without deleting or hiding any Spending intent already recorded on my Expenses, so I can turn it back on later and find my data intact.
3. As a household member, I want turning the switch back on to immediately show whatever Spending intent I'd already recorded while it was on before, so toggling it doesn't feel destructive.

**Category default**

4. As a household member, I want to set a default Spending intent on a Category (Necessity, Desire, Desire+Wise, Desire+Bullshit, or none), so Categories I always spend the same way on don't need re-tagging every time.
5. As a household member, I want a new Expense under a Category with a default Spending intent to start pre-filled with that default, so the common case takes zero extra taps.
6. As a household member, I want to freely change an Expense's own Spending intent away from its Category's default before saving, so an exception doesn't require me to fight the pre-fill.
7. As a household member, I want changing a Category's default Spending intent later to never retroactively change any Expense already saved under it, so editing a Category's default is safe and has no surprise blast radius.
8. As a household member, I want to leave a Category's default Spending intent unset, so Categories with genuinely mixed spending (e.g. general shopping) don't get a misleading default.

**Recording Spending intent on an Expense**

9. As a household member, I want to pick Necessity or Desire on an Expense with one tap each, presented as two badges I tap to select, so classifying an expense is as fast as picking a Category.
10. As a household member, I want picking Necessity or Desire to be mutually exclusive with its own pair — picking one dims/disables the other — so I can't end up with an Expense that's nonsensically both.
11. As a household member, I want the Wise/Bullshit badge pair to only appear once I've picked Desire, so I'm never asked to judge the wisdom of a Necessity.
12. As a household member, I want the Wise/Bullshit pair to disappear (and clear) if I switch an Expense from Desire back to Necessity, so the stored state never contradicts the tree's own rule.
13. As a household member, I want picking Wise or Bullshit to also be mutually exclusive within their own pair, for the same reason as Necessity/Desire.
14. As a household member, I want the whole Spending intent section to be genuinely optional — I can save an Expense with nothing picked, or with just Necessity/Desire and no Wise/Bullshit refinement — so using the feature never blocks or slows down a quick entry.
15. As a household member, I want to change or clear an Expense's Spending intent later by editing it, the same as any other field, so a wrong tap isn't permanent.

**Recurring expense**

16. As a household member, I want a Recurring expense to carry its own Spending intent, seeded from its Category's default the same way an Expense's is, so a recurring bill only needs classifying once.
17. As a household member, I want every Expense a Recurring expense generates to carry that Recurring expense's own Spending intent, the same way it already carries its Category, so I don't have to re-tag twelve identical rent payments by hand.

**Year review — line chart**

18. As a household member, with the toggle on, I want a line chart on the year review showing, per month, what % of that month's tagged spend was Necessity, Desire, Wise, and Bullshit, so I can see how the balance shifts across the year.
19. As a household member, I want the Necessity and Desire lines to mirror each other around 100% of that month's *tagged* spend (Expenses with a Spending intent set at all), so the pair reads as a real split rather than two independent, incomparable numbers.
20. As a household member, I want the Wise and Bullshit lines computed against that month's *Desire* spend specifically (not total spend), so a month with very little Desire spend doesn't make the Wise/Bullshit lines misleadingly tiny against the whole month.
21. As a household member, I want a month with no Spending intent recorded at all to simply show flat/empty lines for that month rather than an error or a misleading zero-means-"all necessity" reading.

**Year review — diagram**

22. As a household member, I want a soft, blurred diagram on the year review (desktop only) showing three overlapping regions — Necessity, Desire+Wise, Desire+Bullshit — sized roughly by how much of the year's € total fell into each, so I get an at-a-glance feel for the year's spending character.
23. As a household member, I want this diagram to simply not appear on mobile, with no alternate fallback view, so building it doesn't require a second design for a screen size it's not meant for.
24. As a household member, I want the diagram's regions sized only by what's actually classified — an untagged Expense contributes to none of the three regions — so the diagram never implies a judgment I never made.

## Implementation Decisions

### Schema (schemaVersion 21 → 22, one migration step)

- `category` gains `spending_intent TEXT CHECK (spending_intent IN ('necessity','desire','desire_wise','desire_bullshit'))`, nullable, no default (NULL = no default set) — the Category's own seed value, read only by the client at Expense-creation time, never resolved server-side against any Expense.
- `expense` gains the identical column and CHECK.
- `recurring_expense` gains the identical column and CHECK; the monthly materialise step copies it onto each generated Expense the same way Category is copied today.
- The single-column enum directly encodes the conditional tree (Necessity is never paired with Wise/Bullshit) rather than two separate columns needing app-level cross-validation — an invalid combination like "necessity, but also wise" is simply not a representable value.
- New `setting` key (existing key/value `setting` table, same pattern as Payers/Payment methods) for the on/off switch, exposed on the `Lists` payload as `spending_intent_enabled: boolean`, off (`false`) on a fresh database and until explicitly turned on.

### API

- `POST`/`PATCH` `/api/categories` accept and return `spending_intent` (nullable enum).
- `POST`/`PATCH` `/api/expenses` accept and return `spending_intent` (nullable enum).
- `POST`/`PATCH` `/api/recurring` accept and return `spending_intent` (nullable enum); the existing monthly-generation handler copies it onto each generated Expense.
- `GET`/`PUT` `/api/settings` gain `spending_intent_enabled` (boolean), read/written the same way the two label lists already are.
- `GET /api/reports/year/{year}/full` (`fullYearReport`/`handleFullYearReport`, `cmd/yearreport.go`) — the existing "opened on purpose, heaviest query in the app" report (ADR-0011) — gains a new field, a per-month grouped total in the same shape `ByMonth`/`monthCategoryTotal` already uses: month, necessity_cents, desire_cents, wise_cents, bullshit_cents (Wise/Bullshit counted from `desire_wise`/`desire_bullshit` rows only; a plain `desire` row with no refinement counts toward `desire_cents` but neither of the other two). The frontend derives both the line chart's percentages and the diagram's three blob totals from this one new field — no second endpoint or second query shape needed.
- Validation (`validate()`, same convention as `holding.validate()`): an unrecognized `spending_intent` string is a 400, matching the DB CHECK rather than surfacing as a 500.

### Frontend

- Expense form and Recurring expense form: a new "Spending intent" section, rendered only when `spending_intent_enabled` is true. Two badge pairs (Necessity/Desire, then conditionally Wise/Bullshit once Desire is picked) — plain click-to-toggle badges in a wrapping flex row, each pair behaving like a radio group (selecting one dims/disables its pair-partner), matching the existing Badge component and its `data-variant` styling rather than introducing a new control type. Selecting Necessity clears and hides the Wise/Bullshit pair if it was previously shown.
- On create, the badges pre-fill from the chosen Category's `spending_intent` default when a Category is picked; picking a different Category re-seeds the pre-fill, but manual edits to the badges themselves are never overwritten mid-edit by a Category change once the user has touched them this session — same "first is honest, second overwrite would be surprising" spirit as other prefill fields in this codebase.
- Categories management page (`Categories.tsx`): the same two-pair badge control added to the Category edit form, storing the Category's default.
- Settings page: a single toggle switch, "Spending intent," wired to `spending_intent_enabled`.
- `YearlyReport.tsx`: behind the toggle,
  - a new line chart reusing `FinancesPathChart`'s existing `recharts` `LineChart` pattern, four series instead of three, computed client-side from the new `SpendingIntentByMonth` field (percentages, not the raw cents the API returns — matches how `Savings.tsx` already derives simple client-side sums from raw cents rather than asking the server for a pre-divided percentage).
  - a new diagram component, three soft/blurred overlapping regions (not a mathematically exact Venn/Euler layout — "by feel," sized roughly by proportion), hidden below a desktop breakpoint with no alternate mobile view.

## Testing Decisions

Tests only external behavior through the existing HTTP API seam (`testApp` helpers — `a.get`/`a.post`/`a.patch`/`a.put`), the same pattern `savings_test.go` and `holding_test.go` already use. No frontend unit tests; no new seam.

- Settings: `spending_intent_enabled` round-trips through `PUT`/`GET /api/settings`, defaults to `false` on a fresh database, and setting it doesn't disturb the other settings on the same payload (same pattern `TestStartingBalanceRoundTripsThroughSettingsAndFoldsIntoSavings` already exercises for a different field).
- Category: `spending_intent` round-trips through create/update; an invalid value (not one of the four, and not null) is rejected with 400; a Category with no `spending_intent` set reads back `null`.
- Expense: `spending_intent` round-trips through create/update, independently settable regardless of the Category's own default; an invalid value is rejected with 400; a `desire_wise`/`desire_bullshit` value is accepted without ever having passed through plain `desire` first (each of the four states is directly valid on its own).
- Recurring expense: `spending_intent` round-trips through create/update; materialising a month copies the Recurring expense's `spending_intent` onto the generated Expense, mirroring however the existing Category-copy is already tested for materialise.
- `fullYearReport`: `SpendingIntentByMonth` sums Expense amounts into the correct month/bucket; a `desire` Expense with no Wise/Bullshit refinement counts toward `desire_cents` only, not either of the other two; an Expense with no `spending_intent` set contributes to none of the four buckets; Incomes never contribute (Spending intent is Expense/Recurring-only, confirmed out of Income's scope entirely).

## Out of Scope

- Items ("voci") carrying their own Spending intent independent of their parent Expense — real precedent exists (an Item can already override the parent Expense's Category), but this ships Expense/Recurring-level only; revisit only if the Expense-level version actually gets used.
- Any general app-wide modularity/plugin system. This ships as exactly one settings boolean gating one feature's UI — not a framework for gating future features.
- A five-point (1–5) slider or any other multi-value scale — rejected in favor of the binary conditional tree during design.
- A mobile fallback for the year-review diagram — deliberately nothing; the diagram simply doesn't render below the desktop breakpoint.
- Any automatic/suggested classification (e.g. inferring Spending intent from Category, Store, or past entries) — every Spending intent value is hand-picked, on the same "never fetched or computed on the household's behalf" principle Holding pricing already follows.
- A bulk-edit or backfill tool for existing Expenses — the migration adds a nullable column; every existing row simply starts unclassified, with no tool built here to retroactively tag historical data.

## Further Notes

- "Wise" is a deliberately distinct concept from the existing `Investments` base category. Investments is literal money moved into a Holding (stocks, ETFs, crypto); Wise is a subjective judgment about an ordinary Desire purchase (e.g. paying more for a durable, better-made item). The two are unrelated and this spec does not connect them.
- Ship on its own branch, not `main` — this is explicitly an experiment the household wants to live-test (particularly whether both household members actually use it) before it's treated as a permanent part of the app.
- The Category default (Implementation Decisions → Schema) is a pure client-side seed, resolved once at Expense-creation time — it is never read by any report or aggregation query. `SpendingIntentByMonth` only ever sums what's actually stored on Expenses, never falls back to a Category's default for an Expense that left its own value unset.
