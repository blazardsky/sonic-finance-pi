# Sonic Finance v1

Status: ready-for-agent

## Problem Statement

A two-person household tracks its money in a spreadsheet. It works, but it has the problems spreadsheets always have: the categories drift ("Groceries", "groceries", "grocery"), the monthly totals have to be rebuilt by hand, the recurring costs get retyped twelve times a year, and nothing is reachable from a phone at the till.

One member is a freelancer, which adds a second problem the spreadsheet handles badly. Invoices go out and payments come back weeks later, so "has this client paid me yet" is a question the spreadsheet can only answer by reading it carefully. And while the invoicing software forecasts what tax will be owed, nothing anywhere records what was *actually* received and *actually* paid — the real figure, after the fact.

## Solution

A single Go binary running on a Raspberry Pi Zero W in the house, serving a React app, reachable from any phone over Tailscale. It stores Expenses and Incomes in SQLite, categorises them, generates the recurring ones by itself, and answers three questions the spreadsheet cannot:

- **This month**: what came in, what went out, and where it went.
- **Who owes me**: which Incomes are unpaid, and which regular Clients have not been invoiced yet this month.
- **This year, in tax terms**: received X in freelance income, paid Y in tax, kept Z%.

It is deliberately one household's private tool. There are no user accounts, no balances between people, and no multi-tenancy — see ADR-0001.

## User Stories

**Logging expenses**

1. As a household member, I want to log an Expense with an amount, a date, and a Category, so that the month's spending is recorded in the three taps it takes at a till.
2. As a household member, I want the Expense date to default to today but stay editable, so that I can enter Saturday's shopping on Sunday without it landing in the wrong day.
3. As a household member, I want to record which Store an Expense happened at as free text, so that I can recognise a receipt later.
4. As a household member, I want the Store field to suggest stores I have used before, so that I get consistency without a fixed list to maintain.
5. As a household member, I want to record the Payer of an Expense from a short configured list, so that I remember whose money it was.
6. As a household member, I want "Both" and "Someone else" available as Payers, so that shared and third-party payments do not need a fake person.
7. As a household member, I want to record the Payment method (cash, credit card, debit card), so that I can match an Expense against a bank statement.
8. As a household member, I want to add a free-text note to an Expense, so that anything the structured fields miss is still captured.
9. As a household member, I want to edit any Expense after saving it, so that a mistake is a correction rather than a delete-and-retype.
10. As a household member, I want to delete an Expense, so that a duplicate entry does not distort the month.

**Items**

11. As a household member, I want to break part of an Expense out as an Item with its own Category, so that a book bought during the grocery run does not count as Food.
12. As a household member, I want Items to be optional, so that a €7 coffee stays a single record.
13. As a household member, I want to itemise only the parts that differ, so that I never have to type nine lines to record one shop.
14. As a household member, I want whatever the Items do not cover to stay under the Expense's own Category, so that a partial breakdown never loses money from the totals.
15. As a household member, I want to give an Item a name, so that "€12, Books" still means something in six months.
16. As a household member, I want to be stopped when my Items add up to more than the Expense total, so that a typo is caught at the point I make it.

**Recurring expenses**

17. As a household member, I want to define a Recurring expense, so that the rent is not something I retype every month.
18. As a household member, I want a Recurring expense to produce a real Expense each month automatically, so that my totals are right whether or not I remembered.
19. As a household member, I want a generated Expense to behave exactly like a typed one, so that I can edit or delete it without learning a second set of rules.
20. As a household member, I want a Recurring expense to have a start month, so that switching one on in March does not invent January and February.
21. As a household member, I want to set the start month back deliberately, so that I *can* backfill when I actually want to.
22. As a household member, I want to deactivate a Recurring expense, so that it stops producing Expenses when the contract ends.
23. As a household member, I want reports over past months to still generate the Expenses that belonged to them, so that a month nobody opened at the time is not silently empty forever.
24. As a household member, I want nothing generated beyond the current month, so that next month's rent never appears in my totals as if it had been paid.
25. As a household member, I want deleting a generated Expense to make it stay deleted, so that the month I did not pay the rent does not keep coming back.
26. As a household member, I want to change a Recurring expense's amount by ending it and creating a new one, so that history keeps the amount that was actually true at the time.
27. As a household member, I want a screen listing my Recurring expenses and whether each is still active, so that I can see my fixed monthly costs in one place.

**Income**

28. As a freelancer, I want to record an Income before it is paid, so that an invoice I have sent is tracked from the moment it goes out.
29. As a freelancer, I want to record when I sent the invoice, so that I know how long a Client has been sitting on it.
30. As a freelancer, I want to record the payment date when the money lands, so that the Income becomes real and starts counting toward totals.
31. As a household member, I want unpaid Incomes excluded from every total, so that the app never tells me I have money I have not received.
32. As a freelancer, I want to record which Client an Income came from, so that I can ask per-Client questions later.
33. As a household member, I want a Client to work for non-freelance income too, so that "Mum" can be the source of a birthday gift.
34. As a household member, I want to classify an Income by reason — freelance, employment, gift, reimbursement, investments, other — so that different kinds of money are not lumped together.
35. As a household member, I want to record who in the household received an Income, so that the same Payer list works in both directions.
36. As a household member, I want to add a note to an Income, so that the circumstances are recorded.
37. As a household member, I want to edit and delete Incomes, for the same reasons as Expenses.

**Pending payments**

38. As a freelancer, I want a list of every unpaid Income oldest first, so that I can see exactly what I am owed.
39. As a freelancer, I want to see how many days each unpaid Income has been waiting, so that chasing is based on a number rather than a feeling.
40. As a freelancer, I want to be shown Clients I have billed in the last three months but not this month, so that an invoice I forgot to send surfaces on its own.
41. As a freelancer, I want that nudge limited to freelance-category income, so that a one-off gift from a relative is not treated as a missing invoice.
42. As a freelancer, I want these to be notices inside the app rather than emails or push notifications, so that nothing depends on a Pi that might be switched off.

**Categories and clients**

43. As a household member, I want to create and rename Categories, so that the list matches how this household actually thinks about money.
44. As a household member, I want renaming a Category to fix every past entry at once, so that tidying the list is not archaeology.
45. As a household member, I want each Category to apply to Expenses, Incomes, or both, so that the expense picker never offers me "Salary".
46. As a household member, I want to hide a Category I no longer use, so that the picker stays short while old reports still resolve.
47. As a household member, I want the Categories the app itself depends on protected from rename and deletion, so that the tax summary cannot be broken by tidying.
48. As a household member, I want to hide a base Category too, so that quitting freelancing does not leave dead options in my picker forever.
49. As a household member, I want to create, rename, and hide Clients, so that the same Client is one row rather than three spellings.

**Reports**

50. As a household member, I want a current-month view showing money in, money out, and the difference, so that I know where the month stands at a glance.
51. As a household member, I want a breakdown of the month's Expenses by Category, so that I can see what the money went on.
52. As a household member, I want Item categories reflected in that breakdown, so that the book bought at the supermarket appears under Books.
53. As a household member, I want to see the most recent entries on the home screen, so that I can confirm what I just typed.
54. As a household member, I want a prominent add button, so that logging an Expense is the fastest thing the app does.
55. As a household member, I want to move between months and see any past month, so that comparison is possible.
56. As a household member, I want a year view, so that I can see the shape of a whole year.
57. As a freelancer, I want a yearly summary reading "received X, paid Y in tax, net Z%", so that I have the actual figures rather than my invoicing software's forecasts.
58. As a freelancer, I want that summary computed from freelance income only, so that already-taxed employment income and untaxed gifts do not make the percentage meaningless.
59. As a freelancer, I want to record which Tax year a tax payment relates to, so that tax paid in 2027 on 2026's income is attributed to 2026.
60. As a freelancer, I want the Tax year to default to the payment year but stay editable, so that the common case is free and the real case is possible.

**Settings, access, and data**

61. As a household member, I want to edit the Payer list in the app, so that adding a person does not mean SSHing into a Pi.
62. As a household member, I want to edit the Payment method list in the app, for the same reason.
63. As a household member, I want an Expense to keep the label text it was saved with, so that renaming a Payer later does not rewrite what old entries say.
64. As a household member, I want the app behind a single shared password, so that being on the tailnet is not the same as being able to read the household's finances.
65. As a household member, I want to stay logged in for 30 days, so that a tracker I open daily does not ask for a password daily.
66. As a household member, I want to change the password from the settings screen, so that rotating it needs no redeploy.
67. As a household member, I want to download a consistent copy of the database with one click, so that backing up to Proton Drive is a habit rather than a project.
68. As a household member, I want to export Expenses and Incomes as CSV, so that I am never locked into this app the way I was locked into the spreadsheet.
69. As a household member, I want the interface in Italian, so that it reads naturally.
70. As a household member, I want amounts and dates formatted Italian-style, so that €1.234,56 and 01/09/2026 look right.
71. As the maintainer, I want the binary to bring its own database schema up to date on startup, so that deploying a new version to the Pi never means hand-editing the only copy of my data.
72. As the maintainer, I want the app to refuse to generate Recurring expenses when the Pi's clock is obviously wrong, so that a boot without network time does not write 1970 into my records.

## Implementation Decisions

### Shape

- One Go binary. Backend and frontend live in this repository; the React app moves in from `sonic-finance-frontend` as a plain copy into `web/` (no git history — it is an unmodified scaffold), builds into `static/`, and is embedded with `go:embed`. See ADR-0006.
- Go code stays in `package main`, split across files by area (expenses, incomes, categories, clients, recurring, reports, settings, auth, backup, migrate). No `internal/` packages: the layer boundaries are not known yet, and exporting interfaces before they are is ceremony bought on credit.
- Routing is stdlib `net/http` using Go 1.22+ method-and-wildcard patterns. No router dependency.
- The app is constructed as `newApp(db *sql.DB, now func() time.Time) http.Handler`. The clock is injected — it is the only dependency tests cannot work around, since half the recurring logic is "which month is it".

### Money, dates, and months

- All money is `INTEGER` minor units (cents). Floats never touch money.
- Single currency, EUR. No currency column.
- Dates are `TEXT` `YYYY-MM-DD`; months are `TEXT` `YYYY-MM`. SQLite has no date type and these sort correctly as strings.
- Formatting and UI language are frontend concerns: Italian strings in a config file, Italian number and date formatting. User-entered content is stored and shown in whatever language it was typed.

### Schema

Applied at startup via `PRAGMA user_version` and a switch statement — no migration library. Each schema change bumps the version and adds a case. `PRAGMA journal_mode = WAL` and `PRAGMA synchronous = NORMAL` are set on open, per the README's SD-card concern.

- **`setting`** — key/value. Holds the bcrypt password hash, the Payer list, and the Payment method list.
- **`category`** — id, name, `applies_to` (`expense` | `income` | `both`), `hidden`, and a nullable stable `code`. A non-null `code` marks a Base category: protected from rename and deletion, hideable. Seeded base categories are `freelance` (income) and `taxes` (expense). Other categories are seeded as a convenience but are fully editable, because nothing in the code resolves them. See ADR-0008.
- **`client`** — id, name, `hidden`.
- **`expense`** — id, `occurred_on`, `amount_cents`, `category_id`, `store`, `payer`, `payment_method`, `note`, nullable `tax_year`, nullable `recurring_id`, `created_at`. `payer` and `payment_method` are **stored as text**, not foreign keys: they are labels, and renaming one must not rewrite history. `category_id` **is** a foreign key, because renaming a Category *should* fix every past entry.
- **`item`** — id, `expense_id`, `name`, `amount_cents`, `category_id`.
- **`income`** — id, `amount_cents`, `category_id`, nullable `client_id`, `payer`, nullable `payment_date`, nullable `invoice_sent_date`, `note`, `created_at`. An Income with a null `payment_date` is unpaid and counts toward no total.
- **`recurring_expense`** — id, template fields mirroring `expense`, `day_of_month`, `start_month`, nullable `end_month`, `created_at`.
- **`recurring_skip`** — (`recurring_id`, `month`), primary key on both.

### Expense totals and items

The Expense's `amount_cents` is authoritative and always present. Items are an optional partial breakdown. Category reporting attributes each Item's amount to the Item's Category and the remainder to the Expense's Category. Items summing to more than the Expense total is rejected on save. An Item sharing the Expense's Category is pointless but permitted — no validation. See ADR-0002.

### Recurring generation

Materialisation is a single function called before any read that covers a month: month views, year views, and reports. For each month M in the requested range and each Recurring expense where `start_month <= M <= coalesce(end_month, M)`, an Expense is created unless one already exists for that (recurring, month) or a `recurring_skip` row covers it. Generation never runs for a month after the current one, and is idempotent. Deleting an Expense that carries a `recurring_id` writes a skip row for its month. See ADR-0005.

The generated Expense's date is `day_of_month` within M, clamped to the month's length (a 31 becomes the 30th in April). `day_of_month` defaults to the day the Recurring expense was created. **This was decided while writing the spec rather than during design — flag it if you would rather every generated Expense simply landed on the 1st.**

### Clock

`now()` is injected. On startup, and before any generation, the app compares it against a build-time constant. If the clock reads earlier than that constant the Pi has booted without network time: the app logs loudly, refuses to materialise Recurring expenses, and exposes the condition so the UI can warn. Refusing is correct — generating against a wrong clock writes bad data, while generating nothing is recoverable by restarting once NTP has synced.

Note that date *defaulting* happens in the browser, on a phone with a correct clock, so the no-RTC problem only affects the server's own sense of "which month is it".

### Tax summary

`X` = received Income (payment date not null) in the `freelance` base Category for the year. `Y` = Expenses in the `taxes` base Category whose `tax_year` matches the year — attributed by Tax year, not payment date. `tax_year` defaults to the year of `occurred_on` and is editable, and is only surfaced in the UI for Expenses in a tax Category. One total; no per-tax-type breakdown. See ADR-0008.

### Pending payments

Two lists from one endpoint. **Outstanding**: Incomes with a null payment date, oldest first by invoice-sent date (falling back to creation), with days waiting. **Not yet invoiced**: Clients with at least one `freelance` Income in the previous three calendar months and nothing recorded this month. The three-month window is a code constant, not a setting.

### API

JSON over `/api/`. Items have no endpoints of their own — they are nested in the Expense payload and saved with it, because an Item has no life outside its Expense.

```
POST/GET      /api/expenses            GET/PATCH/DELETE /api/expenses/{id}
POST/GET      /api/incomes             GET/PATCH/DELETE /api/incomes/{id}
POST/GET      /api/categories          PATCH/DELETE     /api/categories/{id}
POST/GET      /api/clients             PATCH/DELETE     /api/clients/{id}
POST/GET      /api/recurring           PATCH/DELETE     /api/recurring/{id}
GET           /api/reports/month/{yyyy-mm}
GET           /api/reports/year/{yyyy}
GET           /api/reports/tax/{yyyy}
GET           /api/pending-payments
GET/PUT       /api/settings            POST /api/settings/password
POST          /api/login               POST /api/logout
GET           /api/backup              GET  /api/export/{expenses|incomes}.csv
```

`PATCH` or `DELETE` on a Base category returns 409. Static assets are served unauthenticated (the SPA shell renders the login screen); every `/api/` route except `/api/login` requires a valid session.

### Auth

One shared password, bcrypt-hashed in `setting`, no username. Set on first run. A signed session cookie lasting 30 days; `HttpOnly`, `SameSite=Lax`. Plain HTTP, reached over Tailscale, which supplies the transport encryption. See ADR-0007.

### Backup and export

`GET /api/backup` runs SQLite's `VACUUM INTO` to a temporary file, streams it as a download, and removes it — a consistent copy taken without stopping writes. CSV export is a flat dump per entity with Items on their own rows referencing their Expense.

### Build and deploy

`build.sh` runs the frontend build (Vite `outDir: '../static'`), then cross-compiles `CGO_ENABLED=0 GOOS=linux GOARCH=arm GOARM=6`. One file is copied to the Pi. `static/` is gitignored except a committed `static/.gitkeep`, because `go:embed static/*` fails to compile against an empty directory.

## Testing Decisions

**One seam: the HTTP boundary.** Every test builds the app exactly as production does — `newApp(db, clock)` — against a real SQLite database in a temp directory, created by the same migration code that runs on the Pi, then drives it through `httptest` and asserts on JSON responses and subsequent reads.

**What makes a good test here.** It asserts on behaviour visible through the API: status codes, response bodies, and what a later request sees. It does not reach into unexported functions, assert on SQL, or check that a particular helper was called. A test should survive any refactor that leaves the API's behaviour unchanged — that is the entire point of putting the seam this high.

**No mocks and no repository interfaces.** SQLite is a file; a real database per test costs microseconds. Introducing a storage interface purely so it can be faked would add a layer that exists only for tests, and would let the SQL — where the interesting bugs live — go untested.

**Time is controlled, never real.** Tests pass a fixed clock. Anything asserting on "the current month" that reads the wall clock is a flake waiting for the 1st of the month.

**Modules under test** — all of them, through that one seam. The highest-value cases:

- Recurring generation: idempotent on repeated reads; respects the start/end window; never generates beyond the current month; honours a skip row after a generated Expense is deleted; a report over an old month generates that month's rows; reactivation creates a new Recurring rather than backfilling.
- Item arithmetic: the remainder falls to the Expense's Category; a partial breakdown leaves the Expense total untouched; Items exceeding the total are rejected.
- Income payment state: unpaid Incomes appear in pending payments and in no total; adding a payment date moves them into totals.
- Tax summary: attribution by Tax year rather than payment date; freelance income only; unpaid freelance income excluded.
- Base categories: rename and delete rejected, hide accepted, historical entries still resolve after hiding.
- Auth: unauthenticated `/api/` calls rejected; login issues a working session; static assets remain reachable.
- Migrations: a fresh database reaches the current `user_version`; running startup twice is a no-op.

**Prior art: none.** This is greenfield, so the first test file sets the pattern for every one after it — build it as the reference. A `newTestApp(t)` helper returning a ready server and a controllable clock is the thing to get right first.

**Frontend tests are deliberately omitted.** The logic worth protecting is in Go. A React test setup for a personal tracker is machinery that will not pay for itself, and this is a choice rather than an oversight.

## Out of Scope

- **Settling up between people.** No balances, no shares, no "who owes whom" (ADR-0001).
- **Multiple households, user accounts, per-person logins** (ADR-0001, ADR-0007).
- **Multi-currency.** EUR only, no currency column.
- **Recurring incomes.** Freelance income varies monthly; the flag can be added later if employment income ever arrives.
- **Receipt photos.** Storage and SD-card wear on a Pi Zero W, for a feature a tracker does not need.
- **Two-level categories.** Deferred deliberately; a flat list plus a later `ALTER TABLE` is cheaper than recursive rollups in every report today.
- **Per-tax-type breakdown** (IVA / INPS / IRPEF as separate lines). One total is what was asked for; the split is what two-level categories would eventually buy.
- **Gross/net and tax-percentage fields on Income.** Tax is an Expense when paid; netting the Income too would deduct it twice (ADR-0004).
- **Notifications of any kind** — email, push, scheduled reminders. Everything surfaces as an in-app notice, because nothing may depend on a Pi being awake.
- **Off-device or scheduled backups.** A download button; the copying is manual.
- **TLS and login rate-limiting.** Both become mandatory the moment the app is reachable from the public internet (ADR-0007).
- **Arbitrary date-range reports.** Calendar month and calendar year only. A date picker changes no part of the model and can be added whenever.
- **LLM / tm3 forecasting.** Explicitly a someday-idea on larger hardware. Nothing in this spec should be shaped to accommodate it, and no pluggable forecasting interface should be built in anticipation — it would read this database from outside, purely additively.
- **Invoicing and accounting features.** The freelancer's existing invoicing software keeps that job; this app records what actually happened.

## Further Notes

- The domain glossary is `CONTEXT.md`; use its vocabulary in code, tests, and issue titles. It is opinionated about words to avoid — notably "receivable" (use Pending payment), "line item" (use Item), and "type of payment" (that concept is the Category).
- Eight ADRs in `docs/adr/` cover the decisions most likely to be reversed by accident. ADR-0003 carries the standing hazard worth repeating: **every total must filter on payment-date-not-empty**, or the app counts money that has not arrived.
- `sonic-finance-frontend` folds into `web/` and can then be deleted. The Next.js template that was the third directory has already been removed.
- The repository currently compiles: `main.go` is a stub with an embedded `static/` and a placeholder handler. It is the skeleton to build into, not code to preserve.
