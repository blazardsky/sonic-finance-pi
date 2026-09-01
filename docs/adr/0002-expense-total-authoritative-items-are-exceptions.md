# An Expense's total is authoritative; Items are category exceptions

An Expense always stores what was actually paid. Items are an optional, partial breakdown that exist only to say "this part of the shop belonged to a different Category" — a book bought during the grocery run. Whatever Items do not cover stays under the Expense's own Category.

The obvious alternative — deriving the total from the Items — was rejected because partial itemisation would then silently shrink a €62 shop into a €14 one. A finance tracker may never lose money out of its own totals. Items exceeding the Expense total are rejected on save, since a negative remainder has no meaning.
