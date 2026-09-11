# Item price is what was paid; "discounted" is a label, not a correction

An Item's amount is always what was actually paid for it, never a list price, because it has to reconcile with the Expense's own total. A discount is therefore a real price the Tracker should show, not noise to filter out: a temporary markdown is itself useful information about where an Item was cheaper on a given day. An Item may flag itself as discounted so the Tracker can annotate or exclude these points, but the flag changes nothing about the stored amount or its derived price per unit.

## Considered Options

Letting the household enter a separate, "usual" price alongside the paid one, or letting them hand-override the computed price per unit. Rejected both — either would mean two competing numbers for the same purchase, and the paid price is the only one that has to be true.

## Consequences

Price per unit (amount ÷ quantity) is always derived, never stored, so it cannot drift out of sync with what was actually paid.
