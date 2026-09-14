# 04: Categories and Subcategories can be colored

**What to build:** A Category or Subcategory can carry a color — one of 9 fixed, WCAG AA-validated palette entries (computed in the spec, not to be re-derived). A household member can pick one when adding or editing a Category/Subcategory, sees it as a swatch next to the name in the management tables, and existing Categories keep a sensible migrated color rather than all resetting to the same default.

**Blocked by:** None (can start immediately; independent of 01–03)

**Status:** ready-for-agent

- [ ] Schema (schemaVersion 16 → 17): `category` and `subcategory` each gain `color TEXT NOT NULL DEFAULT 'blue-gray' CHECK (color IN ('blue','orange','aqua','yellow','magenta','green','violet','red','blue-gray'))`
- [ ] Migration backfills every existing `category` row's `color` from its id, using the retired `hue = (id * 137.508) % 360` formula snapped to the nearest of the 8 vivid slots by hue angle (table in the spec) — not left at the column default
- [ ] Existing `subcategory` rows take the plain column default (`blue-gray`) — no snapping, since Subcategory never had a color before
- [ ] `POST`/`PATCH` on both Category and Subcategory accept an optional `color`; invalid values are rejected the same way an invalid `applies_to` is; a Base Category accepts a `color` change same as any other (color is not a Base-protected attribute)
- [ ] The palette itself (9 keys → primary/foreground/background, light and dark) lives as one shared frontend constant, values exactly as computed in the spec
- [ ] Add-Category and Add-Subcategory forms gain a 9-swatch color picker; omitting a choice yields `blue-gray`
- [ ] The Categories table and the Subcategories table each show a small swatch (the slot's `primary`) next to the row's name
- [ ] Backend tests cover: default-on-create, round-trip on patch, rejection of an out-of-set value, and the migration backfill landing on the expected slot for a couple of known ids
