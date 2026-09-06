# 04: Savings page — starting balance as a non-editable block with an edit action

**What to build:** The starting-balance card on the Savings page shows a read-only value by default, with an "edit" action button that reveals the existing editable input/save form. The card is sized normally, not stretched full-length.

**Blocked by:** None (can start immediately)

**Status:** ready-for-agent

- [ ] Starting balance renders as a read-only value by default (no visible input), with an edit affordance (icon button).
- [ ] Clicking edit reveals the existing input + Save control; saving or cancelling returns to the read-only view.
- [ ] The card no longer grows to fill the row (drop/adjust the current `flex-1`) — sized like a normal card alongside the Risparmi total card.
- [ ] Existing save behavior (`PUT /api/settings`, error/success messaging) is unchanged.
