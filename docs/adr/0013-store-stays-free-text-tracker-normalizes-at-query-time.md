# Store stays free text; the Tracker normalizes at query time only

The Item price Tracker groups purchases by Store to compare prices across shops — exactly the kind of exact-match grouping that made Holding a fixed list instead of free text. Store stays free text anyway: the Tracker normalizes (trims, lowercases) at query time for its own grouping, and near-duplicate spellings of the same shop may simply show up as separate rows rather than being merged into one canonical entity.

## Considered Options

Promoting Store to a managed entity like Category or Client, so the Tracker's grouping is always exact. Rejected because it reverses a deliberate, already-documented decision (free text, chosen for quick entry) for the benefit of one new view, and would force every existing Expense form to switch from a text field to a picker.

## Consequences

The Tracker's "where is it cheaper" comparison can under- or over-count store variety when entries aren't spelled consistently. If that turns out to matter in practice, normalizing harder later (e.g. fuzzy-merging Store values) is easier than un-doing a managed entity would be.
