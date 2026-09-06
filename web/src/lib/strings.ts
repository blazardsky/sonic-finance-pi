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
  // The Pi has no real-time clock, so until NTP answers its idea of "now" is
  // wrong and it refuses to generate anything. Nothing is lost by waiting: the
  // next look at a month, once the clock is right, fills in every month owed.
  clockUnset:
    "L'orologio del server non è ancora sincronizzato: le spese ricorrenti non vengono generate. Riprova tra qualche minuto.",

  // Table swap — column headers shared across the record-management screens
  // (Expenses, Incomes, Categories, Clients, Recurring). Everything else a
  // column shows already has its own string above/below, screen by screen.
  details: "Dettagli",
  status: "Stato",
  actions: "Azioni",

  // Date picker (web/src/components/date-picker.tsx) — the one placeholder
  // shown before a date is chosen; the chosen date renders itself.
  chooseDate: "Scegli una data",

  // Form sidebar (web/src/components/form-sidebar.tsx) — the trigger's
  // accessible label, shared by every "add new" form it holds.
  showForm: "Mostra modulo",
  hideForm: "Nascondi modulo",

  // Toast (web/src/components/toast.tsx) — shown after a successful add,
  // shared by every FormSidebar-based page.
  added: "Aggiunto.",

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
  invalidPayer: "Scegli chi ha pagato.",
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
  invalidIncomePayer: "Scegli da chi è arrivato.",
  invalidIncomeClient: "Scegli un cliente.",
  paymentDate: "Data incasso",
  invoiceSentDate: "Fattura inviata",
  notPaidYet: "Da incassare",
  waitingSince: (date: string) => `in attesa dal ${date}`,
  // The row action that sets payment_date to today, for an invoiced Income
  // still waiting on its money — one tap rather than opening the form to
  // pick today off the DatePicker.
  markPaid: "Segna come incassata",
  // The Client Combobox's empty-search-results state.
  noClientsFound: "Nessun cliente trovato.",
  incomeNotSaved: "Entrata non salvata. Riprova.",
  incomeNotDeleted: "Entrata non eliminata. Riprova.",
  noIncomesYet: "Nessuna entrata registrata.",
  confirmDeleteIncome: (amount: string) =>
    `Eliminare l'entrata di € ${amount}?`,

  // Sidebar groups — the same altitude the screens are already ordered at:
  // where things stand, what gets typed, what gets managed.
  navOverview: "Riepilogo",
  navEntries: "Registrazioni",
  navManagement: "Gestione",

  // Dashboard — the home screen's recap of where the year stands
  dashboard: "Panoramica",
  yearIncome: "Entrate dell'anno",
  yearExpenses: "Spese dell'anno",
  // The fixed target each of the three cards' progress bar is read against.
  ofEstimate: (amount: string) => `di € ${amount} stimati`,
  ofWhichTax: "di cui tasse",
  ofWhichExtra: "di cui extra",
  // The daily/weekly trend charts, by Category — the Dashboard reads the
  // rolling window, the Month page buckets the same daily rows into weeks
  // itself (no separate weekly endpoint).
  dailyTrend: "Andamento giornaliero",
  noDailyTrend: "Nessuna spesa negli ultimi 30 giorni.",
  weeklyTrend: "Andamento settimanale",
  latestExpenses: "Ultime spese",
  noRecentExpenses: "Nessuna spesa recente.",
  recentIncomes: "Ultime entrate",
  noRecentIncomes: "Nessuna entrata recente.",
  upcomingRecurring: "Prossime scadenze",
  noUpcomingRecurring: "Nessuna scadenza in arrivo.",
  // "tra" reads naturally before a day count in Italian; "oggi"/"domani" are
  // said the way a person would say them rather than as "tra 0/1 giorni".
  dueIn: (days: number) =>
    days === 0 ? "oggi" : days === 1 ? "domani" : `tra ${days} giorni`,

  // The two alerts: unpaid Incomes from named Clients, and regular Clients
  // who look unbilled. Names as badges — the yes/no copy is only the empty
  // state. See cmd/pending.go.
  clientsThisMonth: "Clienti questo mese",
  invoicesSentTitle: "Fatture",
  invoicesAllSent: "Tutti i clienti abituali sono stati fatturati questo mese.",
  noPendingClients: "Nessun pagamento in attesa.",
  markPaidLabel: (name: string) => `Segna come incassato: ${name}`,

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

  // Budget / Target / Goal (ticket 04) — shown on the Month page only for the
  // current real month (CONTEXT.md: Budget/Target are about Expenses, Goal is
  // about savings — three distinct words, kept distinct here too). The same
  // two labels are reused on the Settings screen, where Target and Goal are
  // actually edited.
  budgetTargetGoal: "Budget, target e obiettivo",
  budget: "Budget",
  target: "Target",
  savingsGoal: "Obiettivo di risparmio",
  budgetUnavailable:
    "Servono almeno 3 mesi di storico per calcolare il budget.",
  targetHint: "Parte uguale al budget calcolato, poi resta quello che scegli.",

  // Year — the shape of a whole year, and the year in tax terms
  year: "Anno",
  previousYear: "Anno precedente",
  nextYear: "Anno successivo",
  byMonth: "Mese per mese",
  // Short month names, January first, as the year's twelve rows label
  // themselves. Lower case: they are labels in a table, not sentences.
  monthsShort: [
    "gen",
    "feb",
    "mar",
    "apr",
    "mag",
    "giu",
    "lug",
    "ago",
    "set",
    "ott",
    "nov",
    "dic",
  ],
  taxSummary: "Riepilogo fiscale",
  // "Incassato" and not "fatturato": the whole point of this figure is that it
  // is the money that actually arrived, not the invoicing software's forecast.
  received: "Incassato (freelance)",
  taxPaid: "Tasse pagate",
  net: "Netto",
  netPercent: (percent: string) => `${percent}% di quanto incassato`,
  // Why the two halves are counted differently, in one line: the household
  // will otherwise compare this against a bank statement and find it wrong.
  taxSummaryHint:
    "Le tasse sono attribuite all'anno di competenza, non alla data in cui sono state pagate.",
  taxYear: "Anno di competenza",
  taxYearHint: "L'anno del reddito a cui si riferisce questo pagamento.",
  // Named rather than "quest'anno", which would be a lie on every year but
  // the current one — this screen steps back through them.
  nothingReceived: (year: string) =>
    `Nessuna entrata freelance incassata nel ${year}.`,

  // Full yearly report (ticket 07) — its own page rather than a Year.tsx
  // extension (ADR-0011): expenses by Category for every month, a running
  // total, a tax-excluded total, Savings at the start of the year, the
  // year's cumulative path, and medians scoped to that year alone.
  yearReport: "Report annuale",
  fullYearBreakdown: "Spese per categoria, mese per mese",
  runningTotal: "Progressivo",
  total: "Totale",
  expenseExcludingTax: "Spese, esclusa Tasse",
  taxes: "Tasse",
  savingsAtStartOfYear: (year: string) => `Risparmi a inizio ${year}`,
  financesPath: "Andamento dell'anno",
  medianExpense: "Spesa mediana",
  medianIncome: "Entrata mediana",
  medianNet: "Netto mediano",
  // Named to tell it apart from Budget's own median (CONTEXT.md): this one
  // is scoped to the reported year's completed months, not a rolling window.
  medianHint: (year: string) =>
    `Sui mesi già conclusi del ${year}.`,
  noSpendThisYear: "Nessuna spesa in questo anno.",

  // Pending payments — what is owed, and what looks unbilled
  pendingPayments: "In attesa di pagamento",
  // Days rather than a date, because chasing is based on a number: an invoice
  // sent today has been waiting no days, which is "da oggi" and not "da 0".
  waitingDays: (days: number) =>
    days === 0
      ? "in attesa da oggi"
      : days === 1
        ? "in attesa da 1 giorno"
        : `in attesa da ${days} giorni`,
  notYetInvoiced: "Forse da fatturare",
  notYetInvoicedHint:
    "Clienti fatturati negli ultimi mesi, ma non ancora questo mese.",

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
  editClient: "Modifica cliente",
  noClientsYet: "Nessun cliente registrato.",
  clientNotSaved: "Cliente non salvato. Riprova.",
  hiddenClient: "nascosto",
  confirmDeleteClient: (name: string) => `Eliminare il cliente "${name}"?`,
  clientInUse:
    "Questo cliente è usato da entrate registrate. Nascondilo invece di eliminarlo.",
  defaultCategory: "Categoria predefinita",
  totalEarned: "Totale incassato",

  // Contracts — an agreed total from a Client over a date range (ticket 05).
  // startMonth/endMonth/save/notSet above are reused as-is, same concept.
  contracts: "Contratti",
  addContract: "Aggiungi contratto",
  noContractsYet: "Nessun contratto per questo cliente.",
  contractTotal: "Totale contratto",
  contractNotSaved:
    "Contratto non salvato: controlla le date o sovrappone un altro contratto di questo cliente.",
  expectedSoFar: "Dovuto finora",
  contractReceived: "Incassato",
  invoiceTarget: "Da fatturare questo mese",
  contractOverdue: "scaduto",
  // The Income form's Contract picker, scoped to the selected Client's own
  // Contracts. Unlinked is the default and stays a real, common choice — an
  // Income from a Client with a Contract is not automatically that
  // Contract's money (spec, Out of Scope: no auto-guessing).
  contract: "Contratto",
  extraIncome: "Extra (nessun contratto)",

  // Titoli — the fixed list of stocks/ETFs/crypto/bonds the household
  // invests in (ticket 03 is what records buying and selling one)
  holdings: "Titoli",
  holdingName: "Nome",
  holdingType: "Tipo",
  addHolding: "Aggiungi investimento",
  editHolding: "Modifica investimento",
  noHoldingsYet: "Nessun investimento registrato.",
  holdingNotSaved: "Investimento non salvato. Riprova.",
  holdingTypeETF: "ETF",
  holdingTypeCrypto: "Crypto",
  holdingTypeStock: "Azione",
  holdingTypeBond: "Obbligazione",
  holdingTypeOther: "Altro",

  // Investimenti — the dedicated Buy/Sell form (ticket 03): a buy posts to
  // /api/expenses and a sell to /api/incomes, both under the Investments
  // category, so this screen's own vocabulary is deliberately thin.
  investments: "Investimenti",
  buy: "Acquisto",
  sell: "Vendita",
  holding: "Investimento",
  chooseHolding: "Scegli un investimento",
  noHoldingsToBuySell:
    "Nessun investimento in elenco. Aggiungine uno nella pagina Titoli.",
  buySellNotSaved: "Operazione non salvata. Riprova.",
  recordBuy: "Registra acquisto",
  recordSell: "Registra vendita",
  noBuysSellsYet: "Nessun acquisto o vendita registrato.",

  // Risparmi — read-only recap (ticket 07): total Savings, starting balance,
  // and the portfolio breakdown from GET /api/reports/savings. No entries are
  // added here except the starting balance, saved through the same
  // /api/settings payload Target and Goal use.
  savings: "Risparmi",
  startingBalance: "Saldo di partenza",
  startingBalanceHint: "Il risparmio accumulato prima di usare l'app.",
  startingBalanceNotSaved: "Saldo di partenza non salvato. Riprova.",
  portfolio: "Portafoglio",
  noHoldingsInPortfolio: "Nessun investimento in portafoglio.",

  // Settings — the two configured lists, and the password
  settings: "Impostazioni",
  // The Payer list is one list read in both directions, so it is named after
  // the people on it rather than after either wording the pickers use.
  payersList: "Persone",
  paymentMethodsList: "Metodi di pagamento",
  onePerLine: "Una per riga. L'ordine è quello dei menù.",
  // The bargain worth stating out loud on the screen that makes it: a Payer is
  // label text on the entry, so renaming one here is a change to what the next
  // entry can say and to nothing already recorded.
  listsHistorySafe:
    "Le spese e le entrate già registrate non cambiano: conservano il testo con cui sono state salvate.",
  listsSaved: "Elenchi salvati.",
  listsNotSaved: "Elenchi non salvati. Riprova.",
  emptyList: "Ogni elenco vuole almeno una voce.",

  changePassword: "Cambia password",
  currentPassword: "Password attuale",
  newPassword: "Nuova password",
  // Changing it signs every other device out, because the session signature is
  // derived from the password. Said before the fact, not discovered after it.
  changePasswordHint:
    "Cambiando la password gli altri dispositivi dovranno accedere di nuovo.",
  passwordChanged: "Password aggiornata.",
  passwordNotChanged: "Password non aggiornata. Riprova.",
  wrongCurrentPassword: "Password attuale errata.",
  passwordTooShort: "La password deve avere almeno 8 caratteri.",

  // Reminders — labeled on/off toggles for a manual action the app doesn't
  // automate (ticket 06). CONTEXT.md is explicit these are not alerts,
  // notifications or tasks: a Reminder never fires anything on its own.
  reminders: "Promemoria pagamenti di questo mese",
  addReminder: "Aggiungi promemoria",
  reminderLabel: "Es. Bonifico affitto",
  noRemindersYet: "Nessun promemoria.",
  reminderNotSaved: "Promemoria non salvato. Riprova.",
  confirmDeleteReminder: (label: string) =>
    `Eliminare il promemoria "${label}"?`,
  // The per-row Toggle's accessible name (ticket 14) — the visible label
  // beside it already names the reminder, so this names the action instead.
  reminderToggleLabel: (label: string, enabled: boolean) =>
    enabled ? `Disattiva ${label}` : `Attiva ${label}`,

  // Backup — a button, then the copying is yours. The spreadsheet had no
  // way out; a .db and two CSVs are the way out of this app.
  backupAndExport: "Backup ed esportazione",
  downloadBackup: "Scarica il database",
  exportExpenses: "Esporta spese (CSV)",
  exportIncomes: "Esporta entrate (CSV)",
  backupHint:
    "Una copia da spostare altrove a mano — Proton Drive, un altro disco. L'app non lo fa da sola.",
} as const
