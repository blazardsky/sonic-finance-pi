import { useCallback, useEffect, useState } from "react"
import { RiDeleteBinLine, RiEditLine, RiEyeLine, RiEyeOffLine, RiMoreLine } from "@remixicon/react"

import { Button } from "@/components/ui/button"
import { Checkbox } from "@/components/ui/checkbox"
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu"
import { Field, FieldLabel } from "@/components/ui/field"
import { Input } from "@/components/ui/input"
import { RadioGroup, RadioGroupItem } from "@/components/ui/radio-group"
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table"
import { api } from "@/lib/api"
import { FormSidebar } from "@/components/form-sidebar"
import { toast } from "@/lib/toast"
import { t } from "@/lib/strings"
import type { Applies, Category, Subcategory } from "@/types"

const appliesLabels: Record<Applies, string> = {
  expense: t.appliesExpense,
  income: t.appliesIncome,
  both: t.appliesBoth,
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
  // The one toggle that decides which table the add form's POST targets —
  // nothing about which Category was on screen at the time is stored: a
  // Subcategory freely pairs with any Category, on any Expense, later.
  const [isSubcategory, setIsSubcategory] = useState(false)
  const [editing, setEditing] = useState<number | null>(null)
  const [editingSub, setEditingSub] = useState<number | null>(null)
  const [sidebarOpen, setSidebarOpen] = useState(true)

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
      { method: "POST", body: JSON.stringify({ name, applies_to: appliesTo }) }
    )
    if (created) {
      setName("")
      toast(t.added)
    }
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

          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>{t.categoryName}</TableHead>
                <TableHead>{t.appliesTo}</TableHead>
                <TableHead className="w-10">{t.actions}</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {categories?.map((c) => (
                <TableRow key={c.id}>
                  <TableCell className="whitespace-normal">
                    {editing === c.id ? (
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
                      <span className={c.hidden ? "text-muted-foreground" : ""}>
                        {c.name}
                        {c.hidden && ` · ${t.hidden}`}
                      </span>
                    )}
                  </TableCell>
                  <TableCell className="text-xs text-muted-foreground">
                    {appliesLabels[c.applies_to]}
                  </TableCell>
                  <TableCell>
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
                        // Without this, Radix returns focus to the "..."
                        // trigger once the menu closes — after Rinomina has
                        // already rendered the name cell's autoFocus input,
                        // stealing focus right back off it.
                        onCloseAutoFocus={(e) => e.preventDefault()}
                      >
                        {/* The server refuses these two on a Base category;
                            disabling them here is so the household is not
                            offered the refusal. */}
                        <DropdownMenuItem
                          disabled={c.base}
                          // Deferred a tick past the click: Radix is still
                          // tearing down the menu's own focus handling at
                          // this point, and mounting the autoFocus input
                          // immediately loses the race against it.
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
                          {c.hidden ? (
                            <RiEyeLine />
                          ) : (
                            <RiEyeOffLine />
                          )}{" "}
                          {c.hidden ? t.unhide : t.hide}
                        </DropdownMenuItem>
                        <DropdownMenuItem
                          variant="destructive"
                          disabled={c.base}
                          onClick={() => {
                            if (confirm(t.confirmDeleteCategory(c.name)))
                              void write(
                                `/api/categories/${c.id}`,
                                { method: "DELETE" },
                                t.categoryInUse
                              )
                          }}
                        >
                          <RiDeleteBinLine /> {t.delete}
                        </DropdownMenuItem>
                      </DropdownMenuContent>
                    </DropdownMenu>
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>

          <h2 className="font-medium">{t.subcategories}</h2>
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>{t.categoryName}</TableHead>
                <TableHead>{t.appliesTo}</TableHead>
                <TableHead className="w-10">{t.actions}</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {subcategories?.map((s) => (
                <TableRow key={s.id}>
                  <TableCell className="whitespace-normal">
                    {editingSub === s.id ? (
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
                      <span className={s.hidden ? "text-muted-foreground" : ""}>
                        {s.name}
                        {s.hidden && ` · ${t.hidden}`}
                      </span>
                    )}
                  </TableCell>
                  <TableCell className="text-xs text-muted-foreground">
                    {appliesLabels[s.applies_to]}
                  </TableCell>
                  <TableCell>
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
                          variant="destructive"
                          onClick={() => {
                            if (confirm(t.confirmDeleteSubcategory(s.name)))
                              void write(
                                `/api/subcategories/${s.id}`,
                                { method: "DELETE" },
                                t.subcategoryInUse
                              )
                          }}
                        >
                          <RiDeleteBinLine /> {t.delete}
                        </DropdownMenuItem>
                      </DropdownMenuContent>
                    </DropdownMenu>
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
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
                {(["expense", "income", "both"] as const).map((value) => (
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

          </form>
        </FormSidebar>
      </div>
    </div>
  )
}
