# 16: Settings screen

**What to build:** Adding a payment method stops requiring an SSH session. The configured lists and the password become editable from the app itself.

**Blocked by:** 06, 09

**Status:** ready-for-agent

- [ ] The Payer list is editable in the app
- [ ] The Payment method list is editable in the app
- [ ] Editing a list never rewrites existing Expenses: they keep the label text they were saved with
- [ ] The password can be changed without a redeploy
- [ ] Tests confirm that renaming a Payer leaves historical entries reading exactly as before
