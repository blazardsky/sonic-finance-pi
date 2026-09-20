import { useCallback, useEffect, useMemo, useState } from "react"
import {
  RiDeleteBinLine,
  RiEditLine,
  RiEyeLine,
  RiEyeOffLine,
  RiMoreLine,
  RiPaletteLine,
} from "@remixicon/react"

import { Button } from "@/components/ui/button"
import { Checkbox } from "@/components/ui/checkbox"
import {
  createDataTableColumnHelper,
  DataTable,
  type DataTableColumnDef,
} from "@/components/data-table"
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog"
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu"
import { Field, FieldLabel } from "@/components/ui/field"
import { Input } from "@/components/ui/input"
import { RadioGroup, RadioGroupItem } from "@/components/ui/radio-group"
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select"
import { api } from "@/lib/api"
import { ColorDot } from "@/components/ColorDot"
import { FormSidebar } from "@/components/form-sidebar"
import { cn } from "@/lib/utils"
import { colorSlots, paletteVar, type ColorSlot } from "@/lib/palette"
import { toast } from "@/lib/toast"
import { t } from "@/lib/strings"
import type { Applies, Category, Subcategory } from "@/types"

const appliesLabels: Record<Applies, string> = {
  expense: t.appliesExpense,
  income: t.appliesIncome,
  both: t.appliesBoth,
}

// Section order both management tables split into (ticket 06) — Spese /
// Entrate / Entrambi, same order the add form's own radio group already uses.
const appliesOrder: Applies[] = ["expense", "income", "both"]

// The default within-section order (colorSlots' fixed order — the whole
// point of a color is grouping, so alphabetical-by-hex would defeat it —
// then name, mirroring the backend's `COLLATE NOCASE` ordering), now given
// to DataTable's Name column as a custom sortFn rather than a
// pre-sorted, pre-partitioned array: DataTable's own groupBy (ticket 03)
// partitions by applies_to itself, sorting first.
function byColorThenName<T extends { color: ColorSlot; name: string }>(
  a: T,
  b: T
): number {
  const byColor = colorSlots.indexOf(a.color) - colorSlots.indexOf(b.color)
  return byColor !== 0 ? byColor : a.name.localeCompare(b.name, "it", { sensitivity: "base" })
}

const colorLabels: Record<ColorSlot, string> = {
  blue: t.colorSlotBlue,
  orange: t.colorSlotOrange,
  aqua: t.colorSlotAqua,
  yellow: t.colorSlotYellow,
  magenta: t.colorSlotMagenta,
  green: t.colorSlotGreen,
  violet: t.colorSlotViolet,
  red: t.colorSlotRed,
  "blue-gray": t.colorSlotBlueGray,
}

// The 9-swatch picker itself, shared by the add form and the row "cambia
// colore" popover: a row of ColorDots, the selected one ringed.
function ColorPicker({
  value,
  onChange,
}: {
  value: ColorSlot
  onChange: (color: ColorSlot) => void
}) {
  return (
    <div className="flex flex-wrap gap-2" role="radiogroup" aria-label={t.color}>
      {colorSlots.map((color) => (
        <button
          key={color}
          type="button"
          role="radio"
          aria-checked={value === color}
          aria-label={colorLabels[color]}
          title={colorLabels[color]}
          onClick={() => onChange(color)}
          className={cn(
            "flex size-7 items-center justify-center rounded-full ring-1 ring-foreground/10",
            value === color && "ring-2 ring-offset-2 ring-offset-background"
          )}
          style={
            value === color
              ? { "--tw-ring-color": paletteVar(color, "primary") } as React.CSSProperties
              : undefined
          }
        >
          <ColorDot color={color} className="size-5" />
        </button>
      ))}
    </div>
  )
}

// The Category management screen: add, rename, hide, delete. Hidden Categories
// stay on this list — it is the only place they can be brought back from — and
// are what every picker filters out.
export function Categories() {
  const [categories, setCategories] = useState<Category[] | null>(null)
  const [subcategories, setSubcategories] = useState<Subcategory[] | null>(null)
  const [error, setError] = useState("")
  const [name, setName] = useState("")
  const [appliesTo, setAppliesTo] = useState<Applies>("expense")
  // The add form's color pick — omitting one is never a required chore
  // (spec, user story 9), so "blue-gray" is a real choice here too, not a
  // placeholder waiting to be replaced.
  const [color, setColor] = useState<ColorSlot>("blue-gray")
  // The row a "cambia colore" dialog is open for, Category or Subcategory —
  // null closes it, the same shape replacing/replacingSub already use below.
  const [coloringCategory, setColoringCategory] = useState<Category | null>(null)
  const [coloringSubcategory, setColoringSubcategory] = useState<Subcategory | null>(null)
  // The one toggle that decides which table the add form's POST targets —
  // nothing about which Category was on screen at the time is stored: a
  // Subcategory freely pairs with any Category, on any Expense, later.
  const [isSubcategory, setIsSubcategory] = useState(false)
  const [editing, setEditing] = useState<number | null>(null)
  const [editingSub, setEditingSub] = useState<number | null>(null)
  const [sidebarOpen, setSidebarOpen] = useState(true)
  // Ticket 01 (safe delete): the Category an in-use delete is offered a
  // replacement for, once the bare DELETE above comes back 409. Null closes
  // the dialog — there is no other state to track, since re-issuing the
  // DELETE with replace_with set is itself the confirmation.
  const [replacing, setReplacing] = useState<Category | null>(null)
  const [replaceWithId, setReplaceWithId] = useState("")
  // Ticket 02 (safe delete, Subcategory): same idea as replacing/replaceWithId
  // above, plus a second way forward Category doesn't have — replaceWithId
  // left empty means "remove the tag instead" is what confirmSubDelete sends.
  const [replacingSub, setReplacingSub] = useState<Subcategory | null>(null)
  const [replaceSubWithId, setReplaceSubWithId] = useState("")

  // Returns its promise so a write can wait for the reload it triggers, and
  // sets state from a callback rather than an awaited line, which is what
  // keeps the effect below off React's cascading-render path.
  const load = useCallback(
    () =>
      Promise.all([
        api("/api/categories").then((res) => res.json()),
        api("/api/subcategories").then((res) => res.json()),
      ])
        .then(([c, s]) => {
          setCategories(c)
          setSubcategories(s)
        })
        .catch(() => setError(t.serverUnreachable)),
    []
  )

  useEffect(() => {
    void load()
  }, [load])

  // Every write goes through here so that the reload and the error wording
  // are in one place: a 409 is the only failure worth its own sentence, and it
  // now has two readings — the tax summary resolves this Category, or entries
  // are still pointing at it. Which one it is depends on what was attempted,
  // so the caller supplies the sentence.
  async function write(
    path: string,
    init: RequestInit,
    onConflict: string = t.categoryProtected
  ) {
    setError("")
    try {
      await api(path, init)
      await load()
      return true
    } catch (res) {
      setError(
        res instanceof Response && res.status === 409
          ? onConflict
          : t.serverUnreachable
      )
      return false
    }
  }

  async function add(event: React.FormEvent) {
    event.preventDefault()
    const created = await write(
      isSubcategory ? "/api/subcategories" : "/api/categories",
      {
        method: "POST",
        body: JSON.stringify({ name, applies_to: appliesTo, color }),
      }
    )
    if (created) {
      setName("")
      setColor("blue-gray")
      toast(t.added)
    }
  }

  async function changeColor(to: ColorSlot) {
    if (!coloringCategory) return
    const id = coloringCategory.id
    setColoringCategory(null)
    if (to === coloringCategory.color) return
    await write(`/api/categories/${id}`, {
      method: "PATCH",
      body: JSON.stringify({ color: to }),
    })
  }

  async function changeSubColor(to: ColorSlot) {
    if (!coloringSubcategory) return
    const id = coloringSubcategory.id
    setColoringSubcategory(null)
    if (to === coloringSubcategory.color) return
    await write(`/api/subcategories/${id}`, {
      method: "PATCH",
      body: JSON.stringify({ color: to }),
    })
  }

  async function rename(category: Category, to: string) {
    setEditing(null)
    if (to.trim() === "" || to === category.name) return
    await write(`/api/categories/${category.id}`, {
      method: "PATCH",
      body: JSON.stringify({ name: to }),
    })
  }

  async function renameSub(sub: Subcategory, to: string) {
    setEditingSub(null)
    if (to.trim() === "" || to === sub.name) return
    await write(`/api/subcategories/${sub.id}`, {
      method: "PATCH",
      body: JSON.stringify({ name: to }),
    })
  }

  // The common case (an unused Category) is unchanged: confirm, DELETE, done.
  // A 409 on a non-Base Category means something still points at it — that
  // opens the replacement picker instead of just showing an error, since
  // there is now a way to actually finish the delete. A Base Category still
  // just shows the protected-category sentence: no replacement fixes that.
  async function deleteCategory(category: Category) {
    setError("")
    try {
      await api(`/api/categories/${category.id}`, { method: "DELETE" })
      await load()
    } catch (res) {
      if (res instanceof Response && res.status === 409 && !category.base) {
        setReplaceWithId("")
        setReplacing(category)
      } else {
        setError(t.categoryProtected)
      }
    }
  }

  // Re-issues the same DELETE with replace_with set, once a same-side target
  // is chosen.
  async function confirmReplaceAndDelete() {
    if (!replacing || !replaceWithId) return
    const ok = await write(
      `/api/categories/${replacing.id}?replace_with=${replaceWithId}`,
      { method: "DELETE" }
    )
    if (ok) setReplacing(null)
  }

  // What the replacement picker offers: the same rule the server enforces —
  // accepts the same side as the Category being deleted — minus the Category
  // itself and anything already hidden, the same as every other Category
  // picker in this app.
  const replacementOptions = (categories ?? []).filter(
    (c) =>
      replacing !== null &&
      c.id !== replacing.id &&
      !c.hidden &&
      (c.applies_to === replacing.applies_to || c.applies_to === "both")
  )

  // Same idea as deleteCategory, minus the Base check — nothing about a
  // Subcategory is ever Base-protected — so any 409 here means it's in use
  // and opens the replace-or-clear dialog.
  async function deleteSubcategory(sub: Subcategory) {
    setError("")
    try {
      await api(`/api/subcategories/${sub.id}`, { method: "DELETE" })
      await load()
    } catch (res) {
      if (res instanceof Response && res.status === 409) {
        setReplaceSubWithId("")
        setReplacingSub(sub)
      } else {
        setError(t.subcategoryInUse)
      }
    }
  }

  async function confirmReplaceSubAndDelete() {
    if (!replacingSub || !replaceSubWithId) return
    const ok = await write(
      `/api/subcategories/${replacingSub.id}?replace_with=${replaceSubWithId}`,
      { method: "DELETE" },
      t.subcategoryInUse
    )
    if (ok) setReplacingSub(null)
  }

  // The other way forward, unique to Subcategory: clear the tag from every
  // referencing row instead of picking a replacement.
  async function confirmClearSubAndDelete() {
    if (!replacingSub) return
    const ok = await write(
      `/api/subcategories/${replacingSub.id}?clear=true`,
      { method: "DELETE" },
      t.subcategoryInUse
    )
    if (ok) setReplacingSub(null)
  }

  // v1.3.0: DataTable's own groupBy (ticket 03) replaces bySection — same
  // Spese/Entrate/Entrambi partitioning, same within-section order
  // (byColorThenName, given to the Name column as a sortingFn and as the
  // default via initialSorting) — Applies To no longer needs its own column
  // filter since it's now the grouping key itself.
  const categoryColumns = useMemo<DataTableColumnDef<Category>[]>(() => {
    const helper = createDataTableColumnHelper<Category>()
    return [
      helper.accessor("name", {
        header: t.categoryName,
        meta: { filterVariant: "text" },
        sortFn: (rowA, rowB) => byColorThenName(rowA.original, rowB.original),
        cell: ({ row }) => {
          const c = row.original
          return editing === c.id ? (
            <Input
              autoFocus
              defaultValue={c.name}
              className="h-9"
              onBlur={(e) => void rename(c, e.target.value)}
              onKeyDown={(e) => {
                if (e.key === "Enter") e.currentTarget.blur()
                if (e.key === "Escape") setEditing(null)
              }}
            />
          ) : (
            <span
              className={cn(
                "inline-flex items-center gap-2",
                c.hidden && "text-muted-foreground"
              )}
            >
              <ColorDot color={c.color} />
              {c.name}
              {c.hidden && ` · ${t.hidden}`}
            </span>
          )
        },
      }),
      helper.accessor("applies_to", {
        header: t.appliesTo,
        enableSorting: false,
        cell: (info) => (
          <span className="text-xs text-muted-foreground">
            {appliesLabels[info.getValue() as Applies]}
          </span>
        ),
      }),
      helper.display({
        id: "actions",
        header: t.actions,
        enableSorting: false,
        cell: ({ row }) => {
          const c = row.original
          return (
            <DropdownMenu>
              <DropdownMenuTrigger asChild>
                <Button
                  type="button"
                  variant="ghost"
                  size="icon-sm"
                  aria-label={t.actions}
                >
                  <RiMoreLine />
                </Button>
              </DropdownMenuTrigger>
              <DropdownMenuContent
                align="end"
                onCloseAutoFocus={(e) => e.preventDefault()}
              >
                <DropdownMenuItem
                  disabled={c.base}
                  onClick={() => setTimeout(() => setEditing(c.id), 0)}
                >
                  <RiEditLine /> {t.rename}
                </DropdownMenuItem>
                <DropdownMenuItem
                  onClick={() =>
                    void write(`/api/categories/${c.id}`, {
                      method: "PATCH",
                      body: JSON.stringify({ hidden: !c.hidden }),
                    })
                  }
                >
                  {c.hidden ? <RiEyeLine /> : <RiEyeOffLine />}{" "}
                  {c.hidden ? t.unhide : t.hide}
                </DropdownMenuItem>
                <DropdownMenuItem
                  onClick={() => setTimeout(() => setColoringCategory(c), 0)}
                >
                  <RiPaletteLine /> {t.changeColor}
                </DropdownMenuItem>
                <DropdownMenuItem
                  variant="destructive"
                  disabled={c.base}
                  onClick={() => {
                    if (confirm(t.confirmDeleteCategory(c.name)))
                      void deleteCategory(c)
                  }}
                >
                  <RiDeleteBinLine /> {t.delete}
                </DropdownMenuItem>
              </DropdownMenuContent>
            </DropdownMenu>
          )
        },
      }),
    ]
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [editing])

  const subcategoryColumns = useMemo<DataTableColumnDef<Subcategory>[]>(() => {
    const helper = createDataTableColumnHelper<Subcategory>()
    return [
      helper.accessor("name", {
        header: t.categoryName,
        meta: { filterVariant: "text" },
        sortFn: (rowA, rowB) => byColorThenName(rowA.original, rowB.original),
        cell: ({ row }) => {
          const s = row.original
          return editingSub === s.id ? (
            <Input
              autoFocus
              defaultValue={s.name}
              className="h-9"
              onBlur={(e) => void renameSub(s, e.target.value)}
              onKeyDown={(e) => {
                if (e.key === "Enter") e.currentTarget.blur()
                if (e.key === "Escape") setEditingSub(null)
              }}
            />
          ) : (
            <span
              className={cn(
                "inline-flex items-center gap-2",
                s.hidden && "text-muted-foreground"
              )}
            >
              <ColorDot color={s.color} />
              {s.name}
              {s.hidden && ` · ${t.hidden}`}
            </span>
          )
        },
      }),
      helper.accessor("applies_to", {
        header: t.appliesTo,
        enableSorting: false,
        cell: (info) => (
          <span className="text-xs text-muted-foreground">
            {appliesLabels[info.getValue() as Applies]}
          </span>
        ),
      }),
      helper.display({
        id: "actions",
        header: t.actions,
        enableSorting: false,
        cell: ({ row }) => {
          const s = row.original
          return (
            <DropdownMenu>
              <DropdownMenuTrigger asChild>
                <Button
                  type="button"
                  variant="ghost"
                  size="icon-sm"
                  aria-label={t.actions}
                >
                  <RiMoreLine />
                </Button>
              </DropdownMenuTrigger>
              <DropdownMenuContent
                align="end"
                onCloseAutoFocus={(e) => e.preventDefault()}
              >
                <DropdownMenuItem
                  onClick={() => setTimeout(() => setEditingSub(s.id), 0)}
                >
                  <RiEditLine /> {t.rename}
                </DropdownMenuItem>
                <DropdownMenuItem
                  onClick={() =>
                    void write(`/api/subcategories/${s.id}`, {
                      method: "PATCH",
                      body: JSON.stringify({ hidden: !s.hidden }),
                    })
                  }
                >
                  {s.hidden ? <RiEyeLine /> : <RiEyeOffLine />}{" "}
                  {s.hidden ? t.unhide : t.hide}
                </DropdownMenuItem>
                <DropdownMenuItem
                  onClick={() => setTimeout(() => setColoringSubcategory(s), 0)}
                >
                  <RiPaletteLine /> {t.changeColor}
                </DropdownMenuItem>
                <DropdownMenuItem
                  variant="destructive"
                  onClick={() => {
                    if (confirm(t.confirmDeleteSubcategory(s.name)))
                      void deleteSubcategory(s)
                  }}
                >
                  <RiDeleteBinLine /> {t.delete}
                </DropdownMenuItem>
              </DropdownMenuContent>
            </DropdownMenu>
          )
        },
      }),
    ]
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [editingSub])

  const subReplacementOptions = (subcategories ?? []).filter(
    (s) =>
      replacingSub !== null &&
      s.id !== replacingSub.id &&
      !s.hidden &&
      (s.applies_to === replacingSub.applies_to || s.applies_to === "both")
  )

  return (
    <div className="mx-auto flex w-full max-w-(--content-max-width) flex-col gap-6 p-6 md:min-h-full">
      <h1 className="font-medium">{t.categories}</h1>

      <div className="flex flex-1 flex-wrap gap-6">
        <div className="flex min-w-0 flex-1 flex-col gap-4">
          {error && (
            <p role="alert" className="text-sm text-destructive">
              {error}
            </p>
          )}

          <DataTable
            columns={categoryColumns}
            data={categories ?? []}
            getRowId={(c) => String(c.id)}
            initialSorting={[{ id: "name", desc: false }]}
            groupBy={{
              key: "applies_to",
              order: appliesOrder,
              label: (value) => appliesLabels[value as Applies],
            }}
          />

          <h2 className="font-medium">{t.subcategories}</h2>
          <DataTable
            columns={subcategoryColumns}
            data={subcategories ?? []}
            getRowId={(s) => String(s.id)}
            initialSorting={[{ id: "name", desc: false }]}
            groupBy={{
              key: "applies_to",
              order: appliesOrder,
              label: (value) => appliesLabels[value as Applies],
            }}
          />
        </div>

        <FormSidebar
          title={isSubcategory ? t.addSubcategory : t.addCategory}
          open={sidebarOpen}
          onOpenChange={setSidebarOpen}
          footer={
            <Button
              type="submit"
              form="category-form"
              size="lg"
              className="h-12 text-base"
            >
              {isSubcategory ? t.addSubcategory : t.addCategory}
            </Button>
          }
        >
          <form
            id="category-form"
            onSubmit={add}
            className="flex flex-col gap-3 md:pb-24"
          >
            <Field>
              <FieldLabel htmlFor="name">{t.categoryName}</FieldLabel>
              <Input
                id="name"
                value={name}
                onChange={(e) => setName(e.target.value)}
                placeholder={t.categoryName}
                required
                className="h-10"
              />
            </Field>

            <FieldLabel htmlFor="is-subcategory" className="font-normal">
              <Checkbox
                id="is-subcategory"
                checked={isSubcategory}
                onCheckedChange={(v) => setIsSubcategory(v === true)}
              />
              {t.isSubcategory}
            </FieldLabel>

            <Field>
              <FieldLabel>{t.appliesTo}</FieldLabel>
              <RadioGroup
                value={appliesTo}
                onValueChange={(v) => setAppliesTo(v as Applies)}
              >
                {appliesOrder.map((value) => (
                  <FieldLabel
                    key={value}
                    htmlFor={`applies-${value}`}
                    className="font-normal"
                  >
                    <RadioGroupItem value={value} id={`applies-${value}`} />
                    {appliesLabels[value]}
                  </FieldLabel>
                ))}
              </RadioGroup>
            </Field>

            <Field>
              <FieldLabel>{t.color}</FieldLabel>
              <ColorPicker value={color} onChange={setColor} />
            </Field>

          </form>
        </FormSidebar>
      </div>

      <Dialog
        open={replacing !== null}
        onOpenChange={(open) => !open && setReplacing(null)}
      >
        <DialogContent>
          {replacing && (
            <>
              <DialogHeader>
                <DialogTitle>{t.replaceCategoryTitle(replacing.name)}</DialogTitle>
                <DialogDescription>{t.replaceCategoryHint}</DialogDescription>
              </DialogHeader>

              <Field>
                <FieldLabel htmlFor="replace-with">{t.replaceWith}</FieldLabel>
                <Select value={replaceWithId} onValueChange={setReplaceWithId}>
                  <SelectTrigger id="replace-with" className="h-10 w-full">
                    <SelectValue placeholder={t.chooseReplacementCategory} />
                  </SelectTrigger>
                  <SelectContent>
                    {replacementOptions.map((c) => (
                      <SelectItem key={c.id} value={String(c.id)}>
                        {c.name}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </Field>

              <DialogFooter>
                <Button
                  type="button"
                  variant="outline"
                  onClick={() => setReplacing(null)}
                >
                  {t.cancel}
                </Button>
                <Button
                  type="button"
                  disabled={!replaceWithId}
                  onClick={() => void confirmReplaceAndDelete()}
                >
                  {t.confirmReplaceAndDelete}
                </Button>
              </DialogFooter>
            </>
          )}
        </DialogContent>
      </Dialog>

      <Dialog
        open={replacingSub !== null}
        onOpenChange={(open) => !open && setReplacingSub(null)}
      >
        <DialogContent>
          {replacingSub && (
            <>
              <DialogHeader>
                <DialogTitle>
                  {t.replaceSubcategoryTitle(replacingSub.name)}
                </DialogTitle>
                <DialogDescription>{t.replaceSubcategoryHint}</DialogDescription>
              </DialogHeader>

              <Field>
                <FieldLabel htmlFor="replace-sub-with">{t.replaceWith}</FieldLabel>
                <Select
                  value={replaceSubWithId}
                  onValueChange={setReplaceSubWithId}
                >
                  <SelectTrigger id="replace-sub-with" className="h-10 w-full">
                    <SelectValue placeholder={t.chooseReplacementSubcategory} />
                  </SelectTrigger>
                  <SelectContent>
                    {subReplacementOptions.map((s) => (
                      <SelectItem key={s.id} value={String(s.id)}>
                        {s.name}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </Field>

              {/* Three buttons, two of them long Italian phrases — one row
                  past sm: overflows the dialog's own sm:max-w-sm, so this
                  footer (unlike every other 2-button one in this file) wraps
                  onto a second line instead of forcing them all onto one. */}
              <DialogFooter className="sm:flex-wrap">
                <Button
                  type="button"
                  variant="outline"
                  onClick={() => setReplacingSub(null)}
                >
                  {t.cancel}
                </Button>
                <Button
                  type="button"
                  variant="outline"
                  onClick={() => void confirmClearSubAndDelete()}
                >
                  {t.removeSubcategoryTag}
                </Button>
                <Button
                  type="button"
                  disabled={!replaceSubWithId}
                  onClick={() => void confirmReplaceSubAndDelete()}
                >
                  {t.confirmReplaceAndDelete}
                </Button>
              </DialogFooter>
            </>
          )}
        </DialogContent>
      </Dialog>

      <Dialog
        open={coloringCategory !== null}
        onOpenChange={(open) => !open && setColoringCategory(null)}
      >
        <DialogContent>
          {coloringCategory && (
            <>
              <DialogHeader>
                <DialogTitle>{t.changeColor}</DialogTitle>
                <DialogDescription>{coloringCategory.name}</DialogDescription>
              </DialogHeader>
              <ColorPicker value={coloringCategory.color} onChange={changeColor} />
            </>
          )}
        </DialogContent>
      </Dialog>

      <Dialog
        open={coloringSubcategory !== null}
        onOpenChange={(open) => !open && setColoringSubcategory(null)}
      >
        <DialogContent>
          {coloringSubcategory && (
            <>
              <DialogHeader>
                <DialogTitle>{t.changeColor}</DialogTitle>
                <DialogDescription>{coloringSubcategory.name}</DialogDescription>
              </DialogHeader>
              <ColorPicker value={coloringSubcategory.color} onChange={changeSubColor} />
            </>
          )}
        </DialogContent>
      </Dialog>
    </div>
  )
}
