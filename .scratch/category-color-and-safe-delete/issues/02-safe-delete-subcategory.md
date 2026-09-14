# 02: Safe delete: Subcategory (replacement or clear)

**What to build:** Deleting a Subcategory still in use offers the same replacement choice ticket 01 gave Category, plus a second option Category doesn't have: since Subcategory is optional everywhere it's used, a household member can instead just remove the tag from every entry using it, leaving each entry's own Category untouched.

**Blocked by:** 01 (reuses and extends the same delete-dialog pattern and API shape)

**Status:** ready-for-agent

- [ ] `DELETE /api/subcategories/{id}` accepts the same `replace_with` query parameter as Category (validated against Subcategory, same-side only) — reassigns `expense.subcategory_id` and `recurring_expense.subcategory_id`, then deletes
- [ ] `DELETE /api/subcategories/{id}` also accepts `clear=true`: sets `expense.subcategory_id` and `recurring_expense.subcategory_id` to `NULL` for every row referencing it, then deletes — the Category on those rows is untouched
- [ ] Neither parameter present: unchanged from today (409 if in use)
- [ ] Frontend: the in-use delete dialog offers both "replace with" (a same-side Subcategory picker) and "remove the tag instead"
- [ ] Backend tests cover: `replace_with` reassignment, `clear=true` nulling both columns without touching `category_id`, and the unparameterized-delete-still-refuses case
