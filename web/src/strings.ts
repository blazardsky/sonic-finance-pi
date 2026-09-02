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

  // Incomes — money owed to or received by the household
  incomes: "Entrate",
  addIncome: "Aggiungi entrata",
  editIncome: "Modifica entrata",
  incomeReason: "Motivo",
  client: "Cliente",
  // The same Payer field an Expense carries, from the same configured list,
  // worded for the direction money moved — "Pagato da" is wrong for an Income.
  // One concept, two wordings, exactly as Category is "Motivo" here.
  incomePayer: "Ricevuto da",
  paymentDate: "Data incasso",
  invoiceSentDate: "Fattura inviata",
  notPaidYet: "Da incassare",
  waitingSince: (date: string) => `in attesa dal ${date}`,
  incomeNotSaved: "Entrata non salvata. Riprova.",
  incomeNotDeleted: "Entrata non eliminata. Riprova.",
  noIncomesYet: "Nessuna entrata registrata.",
  confirmDeleteIncome: (amount: string) =>
    `Eliminare l'entrata di € ${amount}?`,

  // Month — where the month stands
  month: "Mese",
  difference: "Differenza",
  previousMonth: "Mese precedente",
  nextMonth: "Mese successivo",
  byCategory: "Dove sono andati i soldi",
  nothingSpent: "Nessuna spesa questo mese.",
  // "movimenti" is what a bank calls these; this app has Expenses and Incomes,
  // and the word it already uses for having recorded one is "registrata".
  recentEntries: "Ultime registrazioni",
  noEntriesYet: "Nessuna registrazione ancora.",

  // Recurring expenses — the rent, defined once
  recurring: "Ricorrenti",
  addRecurring: "Aggiungi spesa ricorrente",
  editRecurring: "Modifica spesa ricorrente",
  dayOfMonth: "Giorno del mese",
  startMonth: "Dal mese",
  endMonth: "Fino al mese",
  ongoing: "in corso",
  endedIn: (month: string) => `terminata a ${month}`,
  // Ending is all that deactivating is: the window is the record, so the
  // months it did cover stay exactly as they were.
  end: "Termina",
  confirmEndRecurring: (month: string) =>
    `Terminare questa spesa ricorrente a ${month}? Le spese già generate restano.`,
  // The amount changed, so this one ends and a new one starts — history keeps
  // what was true at the time.
  newAmount: "Nuovo importo",
  confirmDeleteRecurring:
    "Eliminare questa spesa ricorrente? Per fermarla senza perdere lo storico, terminala.",
  noReactivating:
    "Una spesa ricorrente terminata si riattiva creandone una nuova, non riaprendola.",
  recurringNotSaved: "Spesa ricorrente non salvata. Riprova.",
  noRecurringYet: "Nessuna spesa ricorrente.",
  recurringInUse:
    "Questa spesa ricorrente ha già generato delle spese. Terminala invece di eliminarla.",
  invalidDayOfMonth: "Il giorno del mese deve essere tra 1 e 31.",
  endBeforeStart: "Il mese di fine è prima di quello di inizio.",
  // The 31st does not exist in April: say so once, here, rather than let the
  // household wonder why the date moved.
  dayClamped: "Nei mesi più corti la spesa cade nell'ultimo giorno.",

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
