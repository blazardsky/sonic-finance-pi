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

  // Expenses
  expenses: "Spese",
  amount: "Importo",
  date: "Data",
  category: "Categoria",
  chooseCategory: "Scegli…",
  addExpense: "Aggiungi spesa",
  invalidAmount: "Importo non valido.",
  expenseNotSaved: "Spesa non salvata. Riprova.",
  expenseNotDeleted: "Spesa non eliminata. Riprova.",
  noExpensesYet: "Nessuna spesa registrata.",
  moreDetails: "Altri dettagli",
  store: "Negozio",
  payer: "Pagato da",
  paymentMethod: "Metodo di pagamento",
  note: "Note",
  notSet: "—",
  editExpense: "Modifica spesa",
  save: "Salva",
  cancel: "Annulla",
  confirmDeleteExpense: (amount: string) =>
    `Eliminare la spesa di € ${amount}?`,

  // Items — the part of an Expense that belongs under another Category
  items: "Voci in altre categorie",
  itemName: "Voce",
  addItem: "Aggiungi voce",
  removeItem: "Rimuovi voce",
  invalidItem: "Ogni voce vuole nome, importo e categoria.",
  itemsOverTotal: "Le voci superano il totale della spesa.",
  remainderIn: (category: string, amount: string) =>
    `Resto in ${category}: € ${amount}`,

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
  categoryInUse:
    "Questa categoria è usata da spese registrate. Nascondila invece di eliminarla.",

  // Clients — who money comes from
  clients: "Clienti",
  clientName: "Nome cliente",
  addClient: "Aggiungi cliente",
  noClientsYet: "Nessun cliente registrato.",
  clientNotSaved: "Cliente non salvato. Riprova.",
  hiddenClient: "nascosto",
  confirmDeleteClient: (name: string) => `Eliminare il cliente "${name}"?`,
  clientInUse:
    "Questo cliente è usato da entrate registrate. Nascondilo invece di eliminarlo.",
} as const
