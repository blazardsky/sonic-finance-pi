# 02: Tracker groups by category before product

**What to build:** `Tracker.tsx`'s item table and item picker currently list
Items in whatever order `GET /api/tracker` returns them — product name only,
with Category shown as a column but not used to group or sort. Sort by
Category name first, then Item name within it, so items sharing a Category
sit together instead of interleaved alphabetically by product.

**Blocked by:** None

**Status:** resolved

- [x] `Tracker.tsx`'s `items` list is sorted by `(categoryName, item.name)`
      before both the item-picker `<Select>` and the table render it.
- [x] No backend change — this is a client-side sort over data already
      fetched.

## Comments

Verified live: Alimentari items (Funghetti all'uovo, Speck di Norcia) now
group together ahead of the single Regali item, alphabetically within each
Category.
