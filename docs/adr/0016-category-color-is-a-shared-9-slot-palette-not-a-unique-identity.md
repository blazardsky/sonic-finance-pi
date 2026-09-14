# Category color is a shared 9-slot palette, not a unique identity

A Category (and Subcategory) can carry a color, used in the management tables, every category picker, and the three chart views (CategoryTrendChart, Dashboard's upcoming-list dot, YearlyReport's pie/bar). Each color is a key into a fixed 9-slot palette — the dataviz standard's 8 CVD-validated categorical hues plus one neutral "blue-gray" default — not an arbitrary value. Categories are expected to share slots: with ~18 categories and 16 subcategories already, and no ceiling on that count, uniqueness was never on the table, so sharing is treated as the feature working as intended — a loose grouping signal for categories the household considers related — rather than a limitation to work around. This retires the earlier `categoryColor(id)`, a deterministic unlimited-hue generator keyed on id, which gave every category a technically-unique but never accessibility-validated color.

## Considered Options

Keeping the unlimited-hue generator and formalizing it into a stored, editable field (arbitrary hue per category, always unique). Rejected: the dataviz standard is explicit that categorical color stops being reliably distinguishable past 8 hues in view at once — a full year's breakdown chart routinely shows more categories than that — so "always unique" was already a broken promise in practice, just not a visible one.

## Consequences

A category's color is not addressable by identity — two categories can render identically, so any code path needing to tell categories apart must do it by id or name, never by color. Adding a category never requires inventing a new hue: it either takes an explicit palette pick or the blue-gray default. Existing categories keep visual continuity via a one-time migration that snaps each one's old generated hue to the nearest of the 9 slots, not an exact carry-over.
