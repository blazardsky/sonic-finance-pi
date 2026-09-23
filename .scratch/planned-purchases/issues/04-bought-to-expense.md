# 04: "Comprato": from plan to Expense

**What to build:** A "Comprato" row action on each Planned purchase that takes the household to the Expense form prefilled with the amount, the Category (if set) and the label. Once that Expense is saved, the Planned purchase leaves the list; cancelling leaves it where it was.

**Blocked by:** 01

**Status:** ready-for-agent

- [x] "Comprato" opens the Expense form with amount, Category and label prefilled, and today's date
- [x] Saving the Expense removes the Planned purchase
- [x] Leaving the form without saving keeps the Planned purchase
- [x] A Planned purchase with no Category opens the form with no Category chosen
