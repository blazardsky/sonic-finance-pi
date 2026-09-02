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
  connected: "Connesso al server",
  serverUnreachable: "Server non raggiungibile",
} as const
