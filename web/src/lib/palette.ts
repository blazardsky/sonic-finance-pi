// The 9-slot Category/Subcategory color palette (spec's "The 9-slot palette",
// ADR-0016): 8 vivid slots reusing the dataviz standard's own CVD-validated
// categorical hues as `primary`, plus one neutral `blue-gray` default for a
// genuinely uncolored Category. `background`/`foreground` are this feature's
// own WCAG AA-validated addition (>=4.5:1) — every hex below is copied
// verbatim from the spec, nothing here is computed at render time.
//
// Colors are a shared grouping signal, not a unique identity (ADR-0016): two
// unrelated Categories can and do render identically.
export type ColorSlot =
  | "blue"
  | "orange"
  | "aqua"
  | "yellow"
  | "magenta"
  | "green"
  | "violet"
  | "red"
  | "blue-gray"

type PaletteEntry = {
  primary: string
  background: string
  foreground: string
}

// The fixed render order every swatch picker and legend uses — the same 9
// keys the backend's CHECK constraint lists, blue-gray last since it reads as
// "no color chosen" rather than a genuine pick.
export const colorSlots: ColorSlot[] = [
  "blue",
  "orange",
  "aqua",
  "yellow",
  "magenta",
  "green",
  "violet",
  "red",
  "blue-gray",
]

export const palette: Record<"light" | "dark", Record<ColorSlot, PaletteEntry>> = {
  light: {
    blue: { primary: "#2a78d6", background: "#e7eff8", foreground: "#195fb3" },
    orange: { primary: "#eb6834", background: "#f8ece7", foreground: "#b34519" },
    aqua: { primary: "#1baf7a", background: "#e7f8f2", foreground: "#127d57" },
    yellow: { primary: "#eda100", background: "#f8f3e7", foreground: "#8f6814" },
    magenta: { primary: "#e87ba4", background: "#f8e7ee", foreground: "#b31953" },
    green: { primary: "#008300", background: "#e7f8e7", foreground: "#128112" },
    violet: { primary: "#4a3aa7", background: "#eae7f8", foreground: "#3019b3" },
    red: { primary: "#e34948", background: "#f8e7e7", foreground: "#b31a19" },
    "blue-gray": { primary: "#7b8b9d", background: "#eef0f1", foreground: "#52647a" },
  },
  dark: {
    blue: { primary: "#3987e5", background: "#1e2c3e", foreground: "#6596d2" },
    orange: { primary: "#d95926", background: "#3e271e", foreground: "#d18161" },
    aqua: { primary: "#199e70", background: "#1e3e33", foreground: "#61d1aa" },
    yellow: { primary: "#c98500", background: "#3e331e", foreground: "#d1ab61" },
    magenta: { primary: "#d55181", background: "#3e1e2a", foreground: "#d56d92" },
    green: { primary: "#008300", background: "#1e3e1e", foreground: "#61d161" },
    violet: { primary: "#9085e9", background: "#211e3e", foreground: "#877dd9" },
    red: { primary: "#e66767", background: "#3e1e1e", foreground: "#d67171" },
    "blue-gray": { primary: "#9da5af", background: "#2a2e32", foreground: "#8d98a5" },
  },
}

// A CSS color that resolves to the right variant on its own, via the
// `light-dark()` CSS function — index.css already sets `color-scheme` on
// `:root`/`.dark` to match the app's own theme toggle (not just the OS
// preference), so this needs no React theme lookup to stay correct.
export function paletteVar(color: ColorSlot, role: keyof PaletteEntry): string {
  return `light-dark(${palette.light[color][role]}, ${palette.dark[color][role]})`
}
