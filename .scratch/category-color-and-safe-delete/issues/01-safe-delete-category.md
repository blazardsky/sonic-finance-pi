# 01: Safe delete: Category (mandatory replacement)

**What to build:** Deleting a Category still in use no longer just refuses — it offers a replacement Category on the same side (Expense/Income/both) to reassign everything to first. A household member deleting an in-use Category is offered this choice and, once they confirm a replacement, every Expense, Item, Income, Recurring expense, and Client default pointing at the old Category now points at the replacement, and the old Category is gone. Deleting an unused Category, or a Base Category, behaves exactly as it does today.

**Blocked by:** None (can start immediately)

**Status:** ready-for-agent

- [ ] `DELETE /api/categories/{id}` accepts an optional `replace_with` query parameter (a Category id)
- [ ] Absent `replace_with`: behavior unchanged from today (204 if unused, 409 if in use, 409 if Base regardless)
- [ ] Present `replace_with`: 400 if it doesn't exist or doesn't accept the same side as the Category being deleted; a Base Category still refuses (409) even with `replace_with` set
- [ ] On success: `expense.category_id`, `item.category_id`, `income.category_id`, `recurring_expense.category_id`, and `client.default_category_id` are all reassigned from the old id to `replace_with` in one transaction, then the old Category row is deleted
- [ ] Frontend: deleting an unused Category is unchanged (confirm, delete, done). Deleting an in-use Category surfaces the 409 as a same-side replacement picker instead of a bare error; confirming re-issues the delete with `replace_with` set
- [ ] Backend tests (black-box HTTP via `testApp`) cover: successful reassignment across every listed column, wrong-side replacement refused, nonexistent replacement refused, Base category still refused with `replace_with` set
