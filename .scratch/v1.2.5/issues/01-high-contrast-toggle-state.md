# 01: High-contrast toggle: state, persistence, header control

**What to build:** A `highContrast` boolean joins `ThemeProvider`'s existing `theme` state — its own `localStorage` key, its own additive `"high-contrast"` class on `<html>` (alongside whatever `light`/`dark` class is already there), and a second icon button in the header next to the existing theme toggle that flips it. No new CSS tokens yet — that's ticket 02; this ticket makes the class toggle observable in the DOM and survive a reload, nothing more.

**Blocked by:** None (can start immediately)

**Status:** ready-for-agent

- [ ] `web/src/components/theme-provider.tsx`: `ThemeProviderState` gains `highContrast: boolean` and `setHighContrast: (next: boolean) => void`; `useTheme()` exposes both
- [ ] Persisted under its own `localStorage` key (sibling to the existing `theme` key, following the same `storageKey`-prop pattern rather than a hardcoded literal), as `"true"`/`"false"`; missing or unparseable value defaults to `false`
- [ ] An effect adds/removes a `"high-contrast"` class on `document.documentElement` whenever `highContrast` changes, independent of the `theme` effect — toggling one never touches the other's class
- [ ] The existing cross-tab `storage` event listener (currently only watching the `theme` key) also watches this new key and updates state to match
- [ ] The existing `"d"` keyboard shortcut is untouched — no new shortcut is added for high-contrast
- [ ] `web/src/components/site-header.tsx`: a second icon button sits next to `ThemeToggle`, same `Button variant="ghost" size="icon-sm"` + `headerButton` styling, toggling `highContrast` on click; its icon and `aria-label` reflect current on/off state (pick a clear on/off icon pair from `@remixicon/react`, consistent with how `ThemeToggle` already varies its icon per state)
- [ ] Manual check: toggle on, reload the page, confirm both the header icon and the `"high-contrast"` class on `<html>` still reflect the choice; toggle off and confirm the same in reverse; confirm toggling high-contrast never changes the `light`/`dark` class and vice versa
