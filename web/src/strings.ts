// Every string the UI shows lives here, in Italian. Components import from it
// rather than inlining text, so the wording is in one place and reads as one
// voice. This is not an i18n framework: there is one household and one
// language, so a plain object is the whole mechanism.
//
// Names of things the household typed — Categories, Clients, Stores, notes —
// are data, not UI strings, and stay in whatever language they were typed in.
export const t = {
  appName: "Sonic Finance",
  connecting: "Connessione…",
  serverUnreachable: "Server non raggiungibile",

  // Login
  password: "Password",
  logIn: "Accedi",
  loggingIn: "Accesso in corso…",
  wrongPassword: "Password errata. Riprova.",

  // Categories
  categories: "Categorie",
  categoryName: "Nome categoria",
  addCategory: "Aggiungi categoria",
  appliesTo: "Si applica a",
  appliesExpense: "Spese",
  appliesIncome: "Entrate",
  appliesBoth: "Entrambi",
  rename: "Rinomina",
  hide: "Nascondi",
  unhide: "Mostra",
  hidden: "nascosta",
  delete: "Elimina",
  confirmDeleteCategory: (name: string) => `Eliminare la categoria "${name}"?`,
  categoryProtected:
    "Questa categoria è di base: il riepilogo fiscale la usa. Puoi solo nasconderla.",
} as const
