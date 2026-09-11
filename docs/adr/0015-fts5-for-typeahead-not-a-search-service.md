# Typeahead for Item and Store names uses SQLite FTS5, not an external search service

Suggesting existing Item and Store names while typing — so "melanzana" and "melanzane" converge, and repeat visits to "esselunga" don't fragment into new spellings — needs typo-tolerant matching over what is, for one household, at most a few thousand rows. `modernc.org/sqlite`, already the app's driver, compiles in FTS5 by default, so a virtual table over these names gives that matching with no new dependency and no new process.

## Considered Options

Typesense (a separate search server) and Typo.js (a spellchecker) were both considered and rejected: this app targets a Raspberry Pi Zero W, where running a second service for typeahead over a personal grocery list is disproportionate, and FTS5 already ships with what's running.
