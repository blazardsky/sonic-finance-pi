# 03: Reminders banner in the Headroom forecast card

**What to build:** A small shadcn Alert in the Headroom forecast area of Acquisti programmati, one line per horizon month that is a tax month (self-employed switch on): "A <mese> si pagano le tasse: sono già accantonate mese per mese."; and one per horizon month that is a bonus month (extra-paycheck switch on): "A <mese> arriva la mensilità aggiuntiva: non è inclusa nella previsione." — <mese> being the full Italian month name. The page reads the worker settings from the settings payload. Reminders only: no figure changes.

**Blocked by:** 01

**Status:** ready-for-agent

- [ ] A tax month within the horizon shows its reminder only with the self-employed switch on
- [ ] A bonus month within the horizon shows its reminder only with the extra-paycheck switch on
- [ ] Marked months outside the horizon show nothing; no Alert at all when there is nothing to say
- [ ] Texts exactly as above
- [ ] Every figure on the page is unchanged by the reminders
- [ ] Type-check passes; manual look at the card
