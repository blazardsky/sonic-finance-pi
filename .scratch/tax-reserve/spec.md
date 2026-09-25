Status: ready-for-agent

# Tax reserve in the Headroom forecast, and the "Dati lavoratori" settings

## Problem Statement

On Acquisti programmati the Headroom forecast treats freelance Income as if all of it were spendable. It isn't: tax on a year's freelance Income is paid during the next year (ADR-0008), in a few large payments. Until those payments land, the forecast shows money that already belongs to the tax office, and in the months they do land the spending figures jump and the Goal buffer absorbs a shock nobody planned for. A salaried member's pay is already net of tax, so nothing should be set aside for it — but the app cannot tell the two apart unless the household says whether anyone is self-employed. The household also knows when taxes are paid and when extra paychecks (tredicesima, quattordicesima) arrive, and wants the forecast card to remind it of those months without distorting the figures.

## Solution

- **Tax reserve** (CONTEXT.md): when the household says it has at least one self-employed worker, every forecast month sets aside a share of its freelance Income for taxes, and so does last month's actual leftover, the start of the running total. The share is the household's own historical rate — the median, across completed tax years, of tax paid for the year ÷ freelance Income received in it — or, with no completed tax year yet, a fallback percentage the household sets (33% by default). Tax payments themselves then leave the spending figures, so they are never counted twice.
- **"Dati lavoratori"**: a new card in Impostazioni, with its own save button, holding the self-employed switch, the fallback percentage, the months taxes are paid, the extra-paycheck switch and the months those arrive.
- **Reminders**: the Headroom forecast card flags each horizon month in which taxes are paid ("already set aside month by month") or an extra paycheck arrives ("not included in the forecast"). Reminders only; no figure changes.
- **Switch off** means exactly today's forecast: no reserve, tax payments stay in spending.

## User Stories

**Saying who works how**

1. As a household member, I want a "Dati lavoratori" card in Impostazioni with its own save button, so worker details are edited apart from the main settings form, like the password card.
2. As a household member, I want a switch "C'è almeno un lavoratore autonomo in famiglia", off by default, so a salaried-only household never sees a tax reserve.
3. As a household member with a self-employed worker, I want to set the fallback percentage set aside for taxes, 33 by default, so the forecast is sensible before the app has a full tax year of my own records.
4. As a household member with a self-employed worker, I want to mark the months in which taxes are paid ("mesi in cui si pagano le tasse", for example June and November) as twelve month toggles, so the forecast card can remind me of them.
5. As a household member, I want the fallback percentage and the tax months shown only while the self-employed switch is on, so the card stays short otherwise.
6. As a household member, I want a switch "Mensilità aggiuntive", off by default, for employees who receive extra paychecks.
7. As a household member with extra paychecks, I want to mark the months they arrive ("mesi con mensilità aggiuntiva", for example June and December) as twelve month toggles, because the months differ from person to person.
8. As a household member, I want those month toggles shown only while their switch is on.
9. As a household member, I want saving this card to leave every other setting (lists, Target, Goal, starting balance, Net worth target, Spending intent) untouched, and saving the other settings to leave this card untouched.
10. As a household member, I want a switch turned off to keep the percentage and months I had entered, so turning it back on doesn't make me retype them.

**The Tax reserve**

11. As a household member with a self-employed worker, I want each forecast month to set aside a share of that month's freelance Income for taxes, so the Headroom I see is money I can actually spend.
12. As a household member, I want the reserve to apply to what active Contracts still owe in that month and to the forecast of freelance Incomes without a Contract (Extra), because both are freelance money that will be taxed.
13. As a household member, I want salary and every other non-freelance Income never reserved against, because salary arrives already net of tax and gifts are not taxable income.
14. As a household member, I want last month's actual leftover to set aside the same share of the freelance Incomes actually received that month, so the start of the running total is treated the same way as every forecast month.
15. As a household member, I want the share to come from my own history — the median, across completed tax years, of tax paid for the year ÷ freelance Income received in it — so the reserve matches what I really pay.
16. As a household member, I want a median rather than an average, so one unusual year doesn't swing the rate.
17. As a household member, I want the fallback percentage used only while there is no completed tax year on record, and my history to replace it from then on.
18. As a household member, I want the reserve subtracted before the Goal buffer, so Goal can absorb a shortfall caused by taxes the same way it absorbs any other.
19. As a household member, I want tax payments (Expenses in the Taxes base category) left out of the spending forecast and out of last month's actual leftover while the reserve applies, so taxes are not counted once as a reserve and again as spending.
20. As a household member, I want a Recurring expense in the Taxes base category (an instalment plan) likewise left out of the Recurring due while the reserve applies, for the same reason.
21. As a household member without a self-employed worker, I want the forecast exactly as it is today, tax payments included in spending, so nothing changes for me.
22. As a household member, I want to see "accantonato per tasse: € X" under "Margine mensile previsto", X being next month's reserve, so I know how much of the coming month's freelance money is spoken for.
23. As a household member without a self-employed worker, I want that line not shown at all.
24. As a household member, I want the Headroom report to expose the reserve for every month and for the start, the rate used and whether it came from my history or the fallback, so every figure can still be checked by hand.

**Reminders**

25. As a household member with a self-employed worker, I want each horizon month marked as a tax month to show "A <mese> si pagano le tasse: sono già accantonate mese per mese.", so a large payment doesn't surprise me and I know it is covered.
26. As a household member with extra paychecks, I want each horizon month marked as a bonus month to show "A <mese> arriva la mensilità aggiuntiva: non è inclusa nella previsione.", so I know the forecast is deliberately cautious there.
27. As a household member, I want no reminder when no marked month falls within the horizon, or when its switch is off.
28. As a household member, I want reminders never to change any figure, because they are notes, not forecasts.

**One forecast for every household**

29. As a salaried-only household member (no Contracts, switch off), I want Headroom to be median Income minus median spending minus Recurring due, through the Goal buffer, from last month's start and by season — nothing extra to configure.
30. As a member of a mixed household, I want salary forecast through the median and freelance through Contracts and Extra with the reserve, without choosing a "mode".

## Implementation Decisions

- **Settings** (no new table, no schema change): stored as settings keys and carried on the existing settings payload, which already leaves omitted fields untouched on a partial update:
  - `self_employed` (switch, default off), `tax_reserve_fallback_percent` (whole number 0–100, default 33), `tax_months` (a set of months 1–12, default empty);
  - `bonus_paychecks` (switch, default off), `bonus_months` (a set of months 1–12, default empty).
  - Month sets are stored the way the label lists are and returned sorted and without duplicates; a month outside 1–12 or a percentage outside 0–100 is refused.
- **"Dati lavoratori" card**: its own card in Impostazioni with its own save button, sending only its own fields, like the password card. shadcn Switch for the two switches, Input for the percentage, twelve shadcn Toggle buttons (short Italian month names) per month set. Dependent fields are hidden, not cleared, while their switch is off.
- **Completed tax year**: tax year Y is completed once all its taxes are due — once year Y+1 is over, because tax on Y's Income is paid during Y+1 (ADR-0008). So, by the clock, Y ≤ current year − 2 (in 2026: 2024 and earlier). A completed year counts only if it has both freelance Income received and tax paid on record — a year with no tax Expenses is far more likely unrecorded than tax-free.
- **Rate**: for each counted completed year, the tax summary's two figures for that year — tax paid (Taxes Expenses by Tax year, including the summary's default of the payment year when none is recorded) ÷ freelance Income received (Freelance Incomes by payment date). The rate is the median of those ratios (the mean of the middle two for an even count). No counted year → the fallback percentage. The same rate is used for every horizon month and for the start.
- **Freelance forecast** for month M: freelance Incomes without a Contract are forecast exactly like Income is today — the same seasonal window (last year's M−1, M, M+1), the same blend with this year's completed months and the same fallbacks — but counting only Incomes in the Freelance base category without a Contract. The overall Income forecast is unchanged and still includes them; the freelance forecast only sizes the reserve.
- **Reserve** for month M = rate × (Contract share(M) + freelance forecast(M)), rounded to the cent. For the start: rate × freelance Incomes received in the last completed month (Contract Incomes included).
- **Leftover** for month M = forecast Income + Contract share − forecast spending − Recurring due − reserve; then the Goal buffer. The start likewise: actual Income − actual Expenses − reserve; then the Goal buffer.
- **No double counting**, only while the reserve applies: Taxes-category Expenses are left out of the monthly spending figures that feed the seasonal and yearly medians, and out of last month's actual Expenses; Taxes-category Recurring expenses are left out of the Recurring due.
- **Switch off**: rate and reserve are not computed; every figure is identical to the current forecast.
- **Headroom report** gains:
  - per month: `tax_reserve_cents` and `freelance_forecast_cents`;
  - top level: `start_tax_reserve_cents`, `tax_reserve_percent` (the rate as a percentage) and `tax_reserve_source` (`history` or `fallback`); with the switch off the reserves and the percent are 0 and the source is empty.
  - Everything else, placements included, keeps its shape and meaning over the new running total.
- **Page**: under "Margine mensile previsto", the line "accantonato per tasse: € X" with next month's reserve, only when the reserve applies.
- **Reminders**: a shadcn Alert inside the Headroom forecast area, one line per horizon month that is a tax month (self-employed switch on) or a bonus month (extra-paycheck switch on), with the texts from the User Stories and the full Italian month name. The page reads the worker settings from the settings payload; the report doesn't carry them. No Alert when there is nothing to say.
- **Glossary**: CONTEXT.md gains the **Tax reserve** entry.

## Testing Decisions

- Test external behaviour only, through the existing seams — no new ones:
  - **The settings API** (prior art: the settings tests): the new fields' defaults, round-trip, partial updates leaving everything else untouched, and refusal of out-of-range values.
  - **The Headroom report over HTTP** with the test app's fixed clock (prior art: the Headroom and Savings tests), seeding Incomes, Expenses, Contracts, Recurring expenses, Goal and the worker settings through the API.
- Cases:
  - Fallback 33% applied to a month's Contract share plus its freelance Extra forecast, and to last month's freelance Incomes for the start.
  - A completed tax year's ratio replaces the fallback; a tax year not yet completed (last year, this year) is ignored; the median across several years.
  - Salary is never reserved against.
  - Taxes Expenses left out of the medians and of the start while the reserve applies.
  - The Goal buffer applies after the reserve.
  - Switch off → every figure identical to the current forecast.
- UI (the card, the line, the reminders) verified by type-check and a manual look.

## Out of Scope

- Changing how Contracts count — what they still owe, spread over their remaining months. A future hybrid that also expects renewals is possible but not decided.
- Different tax rates per person or per Contract.
- Adding extra paychecks to the forecast.
- Reminder kinds other than tax and extra-paycheck months.
- Any Dashboard card.

## Further Notes

- **Experimental**: this is a first attempt at taxes in the forecast. The rate, the "completed tax year" rule and the reminders may be revisited after real use.
- **No modes**: there is deliberately no "employee mode" vs "freelancer mode". Each part activates only with data or a switch: an employee with no Contracts and the switch off gets median Income − median spending − Recurring due, through Goal, from the start and by season; a mixed household gets salary via the median and freelance via Contracts and Extra with the reserve.
- **Extra paychecks are not forecast on purpose**: the seasonal median already filters one-off peaks, and a bonus month is still one of twelve months, just a higher one. The reminder only flags it, keeping the forecast on the cautious side.
- Builds on `.scratch/headroom-forecast/spec.md`; everything it specifies stays as is when the switch is off.
