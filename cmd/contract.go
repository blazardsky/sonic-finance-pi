package main

import (
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"
)

// migrateGiftContractsReminders is schema step 10: the "Regali" category
// promoted in place to the new protected Gift base category, a Client's
// optional default Income category, the new Contract table an Income can
// point at, and the new Reminder table — bundled into one step because a
// database with some of these and not the others is not a state any of the
// four feature tickets built on this one can build on. No user-facing
// behaviour yet: that is every one of those tickets, not this one.
func migrateGiftContractsReminders(tx *sql.Tx) error {
	// Updated in place, never inserted again — same reasoning and the same
	// `code IS NULL` defense-in-depth guard as migrateInvestments's promotion
	// of "Investimenti".
	if _, err := tx.Exec(`UPDATE category SET applies_to = ?, code = ? WHERE name = ? AND code IS NULL`,
		appliesBoth, codeGift, seedRegaliName); err != nil {
		return err
	}

	// Only ever a prefill for the Income form's category picker, never
	// enforced — an Income from this Client can still pick any Income
	// category. A plain reference, so hiding or deleting the category is
	// unaffected; nothing here forces it to stay an Income-appliable one.
	if _, err := tx.Exec(`ALTER TABLE client ADD COLUMN default_category_id INTEGER REFERENCES category(id)`); err != nil {
		return err
	}

	// start_month/end_month are both required, unlike Recurring's open-ended
	// end_month — a Contract always names the range it covers. Plain string
	// comparison on YYYY-MM, same as Recurring's own range.
	if _, err := tx.Exec(`CREATE TABLE contract (
		id           INTEGER PRIMARY KEY,
		client_id    INTEGER NOT NULL REFERENCES client(id),
		start_month  TEXT NOT NULL,
		end_month    TEXT NOT NULL CHECK (end_month >= start_month),
		total_cents  INTEGER NOT NULL CHECK (total_cents > 0)
	) STRICT`); err != nil {
		return err
	}

	// A plain reference, like income.client_id: a Contract with Incomes linked
	// to it refuses deletion rather than silently orphaning the money.
	if _, err := tx.Exec(`ALTER TABLE income ADD COLUMN contract_id INTEGER REFERENCES contract(id)`); err != nil {
		return err
	}

	// set_for_month CHECKs against '' for the reason every other optional month
	// column here does: NULL is the only spelling of "not currently set".
	_, err := tx.Exec(`CREATE TABLE reminder (
		id             INTEGER PRIMARY KEY,
		label          TEXT NOT NULL,
		enabled        INTEGER NOT NULL DEFAULT 0 CHECK (enabled IN (0, 1)),
		set_for_month  TEXT CHECK (set_for_month <> '')
	) STRICT`)
	return err
}

// A Contract is a total the household expects from a Client over a date
// range — a start and end month, both required (ticket 05) — not a payment
// schedule. Everything below TotalCents is computed at read time from the
// range and whatever Incomes ended up linked to it via income.contract_id,
// and none of it is stored: ADR-0012 is explicit that there is no fixed
// schedule to drift out of sync with, only the running total and the months
// left.
//
// A Client can have more than one Contract over time (a yearly renewal), so
// this is not a client column: two Contracts for the same Client are refused
// at write time if their ranges overlap, so "which Contract does this month
// belong to" is never ambiguous (contractOverlaps).
type contract struct {
	ID         int64  `json:"id"`
	ClientID   int64  `json:"client_id"`
	StartMonth string `json:"start_month"`
	EndMonth   string `json:"end_month"`
	TotalCents int64  `json:"total_cents"`

	// The three read-time figures. ExpectedSoFarCents and InvoiceTargetCents
	// are set by computeContractFigures; ReceivedCents and AccountedCents are
	// scanned straight off contractSelect's subqueries.
	ExpectedSoFarCents int64 `json:"expected_so_far_cents"`
	ReceivedCents      int64 `json:"received_cents"`
	AccountedCents     int64 `json:"accounted_cents"`
	InvoiceTargetCents int64 `json:"invoice_target_this_month_cents"`

	// True once the current month is past end_month: invoice_target_this_month
	// then holds the whole remaining shortfall rather than a shortfall divided
	// by zero or negative months remaining — the ticket's explicit edge case.
	Overdue bool `json:"overdue"`
}

// handleListClientContracts answers every Contract a Client has ever had,
// past and current alike — a past Contract is exactly how "renewed each
// year" is told apart from "still running" (spec, Client contracts).
func handleListClientContracts(db *sql.DB, now func() time.Time) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cl, ok := findClient(w, db, r.PathValue("id"))
		if !ok {
			return
		}

		rows, err := db.Query(contractSelect+` WHERE client_id = ? ORDER BY start_month`, cl.ID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		defer rows.Close()

		current := now().Format(monthLayout)
		out := []contract{}
		for rows.Next() {
			c, err := scanContract(rows)
			if err != nil {
				writeError(w, http.StatusInternalServerError, err)
				return
			}
			if err := computeContractFigures(&c, current); err != nil {
				writeError(w, http.StatusInternalServerError, err)
				return
			}
			out = append(out, c)
		}
		if err := rows.Err(); err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		writeJSON(w, http.StatusOK, out)
	}
}

// handleCreateClientContract defines a Contract for the Client the path
// names. client_id, like every other create in this codebase, comes from the
// path rather than the body — there is no way to POST a Contract onto a
// different Client than the one the request addressed.
func handleCreateClientContract(db *sql.DB, now func() time.Time) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cl, ok := findClient(w, db, r.PathValue("id"))
		if !ok {
			return
		}

		var body struct {
			StartMonth string `json:"start_month"`
			EndMonth   string `json:"end_month"`
			TotalCents int64  `json:"total_cents"`
		}
		if err := decodeJSON(w, r, &body); err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		c := contract{
			ClientID:   cl.ID,
			StartMonth: body.StartMonth,
			EndMonth:   body.EndMonth,
			TotalCents: body.TotalCents,
		}
		if err := c.validate(); err != nil {
			writeInvalid(w, err)
			return
		}
		switch overlaps, err := contractOverlaps(db, c.ClientID, c.StartMonth, c.EndMonth, 0); {
		case err != nil:
			writeError(w, http.StatusInternalServerError, err)
			return
		case overlaps:
			writeInvalid(w, errors.New("this contract's dates overlap another contract already on file for this client"))
			return
		}

		res, err := db.Exec(`INSERT INTO contract (client_id, start_month, end_month, total_cents)
			VALUES (?, ?, ?, ?)`, c.ClientID, c.StartMonth, c.EndMonth, c.TotalCents)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		if c.ID, err = res.LastInsertId(); err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		// A brand new Contract has nothing linked to it yet, so the figures
		// are computed straight off zero received/accounted rather than a
		// second round trip through contractSelect.
		if err := computeContractFigures(&c, now().Format(monthLayout)); err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		writeJSON(w, http.StatusCreated, c)
	}
}

// contractOverlaps reports whether [start, end] would overlap any existing
// Contract already on file for this Client, other than excludeID itself.
// Two inclusive ranges [a,b] and [c,d] overlap iff a<=d && c<=b — the
// off-by-one trap being that "starts the month straight after another ends"
// must NOT count as an overlap, and this formula gets that right where a
// naive `<` / `>` swap would not: with existing = [a,b] and new = [c,d], c
// straight after b means c > b, so c<=b is false and the two are correctly
// read as disjoint.
//
// excludeID is 0 for a create (nothing to exclude — ids start at 1) and the
// Contract's own id for an edit, so that a PATCH which leaves its own dates
// unchanged does not find its own not-yet-updated row and read that as
// overlapping itself.
func contractOverlaps(db *sql.DB, clientID int64, start, end string, excludeID int64) (bool, error) {
	var found int
	err := db.QueryRow(`SELECT 1 FROM contract
		WHERE client_id = ? AND id != ? AND start_month <= ? AND ? <= end_month LIMIT 1`,
		clientID, excludeID, end, start).Scan(&found)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return false, nil
	case err != nil:
		return false, err
	}
	return true, nil
}

// findContract loads the Contract the path names, writing the response
// itself when there is none — an unparseable id and a missing row are both a
// 404, same as findClient and findIncome.
func findContract(w http.ResponseWriter, db *sql.DB, rawID string) (contract, bool) {
	id, err := strconv.ParseInt(rawID, 10, 64)
	if err != nil {
		writeError(w, http.StatusNotFound, err)
		return contract{}, false
	}
	c, err := scanContract(db.QueryRow(contractSelect+` WHERE contract.id = ?`, id))
	if errors.Is(err, sql.ErrNoRows) {
		writeError(w, http.StatusNotFound, err)
		return contract{}, false
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return contract{}, false
	}
	return c, true
}

// handlePatchClientContract edits a Contract's own terms — the total the
// household agreed with the Client, or the range it covers — for the day one
// was mistyped or renegotiated. Unlike an Income, every field here is
// required rather than optional, so a PATCH still merges onto the stored
// row (decodeJSON's usual partial-update bargain) purely so that correcting
// just the total does not also require resending dates that were already
// right.
//
// ReceivedCents and AccountedCents are restored after decoding, the same way
// handlePatchClient protects total_earned_cents: both are computed at read
// time from linked Incomes, never a PATCH-able field, and leaving a forged
// value in would corrupt computeContractFigures's own arithmetic below.
func handlePatchClientContract(db *sql.DB, now func() time.Time) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cl, ok := findClient(w, db, r.PathValue("id"))
		if !ok {
			return
		}
		c, ok := findContract(w, db, r.PathValue("contractId"))
		if !ok {
			return
		}
		if c.ClientID != cl.ID {
			writeError(w, http.StatusNotFound, errors.New("that contract does not belong to this client"))
			return
		}

		id, clientID := c.ID, c.ClientID
		received, accounted := c.ReceivedCents, c.AccountedCents
		if err := decodeJSON(w, r, &c); err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		c.ID, c.ClientID = id, clientID
		c.ReceivedCents, c.AccountedCents = received, accounted

		if err := c.validate(); err != nil {
			writeInvalid(w, err)
			return
		}
		switch overlaps, err := contractOverlaps(db, c.ClientID, c.StartMonth, c.EndMonth, c.ID); {
		case err != nil:
			writeError(w, http.StatusInternalServerError, err)
			return
		case overlaps:
			writeInvalid(w, errors.New("this contract's dates overlap another contract already on file for this client"))
			return
		}

		if _, err := db.Exec(`UPDATE contract SET start_month = ?, end_month = ?, total_cents = ? WHERE id = ?`,
			c.StartMonth, c.EndMonth, c.TotalCents, c.ID); err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		if err := computeContractFigures(&c, now().Format(monthLayout)); err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		writeJSON(w, http.StatusOK, c)
	}
}

// contractClientID reports the client_id a Contract belongs to, and whether
// it exists at all — checked explicitly by checkIncome rather than left to
// the foreign key, for the same reason clientExists is: a bad id should read
// as an invalid request, not the 409 writeError turns a constraint failure
// into.
func contractClientID(db *sql.DB, id int64) (int64, bool, error) {
	var clientID int64
	err := db.QueryRow(`SELECT client_id FROM contract WHERE id = ?`, id).Scan(&clientID)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, false, nil
	}
	return clientID, err == nil, err
}

// contractSelect scans a Contract plus the two sums computeContractFigures needs:
// received (cash-basis, ADR-0003) and accounted (every linked Income at all,
// paid or not). accounted needs no payment_date filter — an unpaid Income
// already linked to this Contract is an invoice already sent, and spec's
// accounted_cents is deliberately "regardless of payment status" so that
// money already invoiced is never suggested for invoicing twice.
// Every column is qualified with contract., unneeded when contractSelect
// queries the bare table but required once contractsDueThisMonth joins in
// client — id and client_id would otherwise be ambiguous between the two.
// netOfBolloFattura is what an Income actually counts toward a Contract:
// AmountCents itself is never adjusted (it is still the amount that arrived),
// but a qualifying Income's 2€ marca da bollo (bolloThresholdCents,
// bolloFatturaCents) is not real freelance revenue, so it is subtracted here
// rather than at the source.
var netOfBolloFattura = fmt.Sprintf(
	`(amount_cents - CASE WHEN bollo_fattura = 1 AND amount_cents > %d THEN %d ELSE 0 END)`,
	bolloThresholdCents, bolloFatturaCents)

var contractColumns = `contract.id, contract.client_id, contract.start_month, contract.end_month, contract.total_cents,
	(SELECT COALESCE(SUM(` + netOfBolloFattura + `), 0) FROM income
		WHERE income.contract_id = contract.id AND income.payment_date IS NOT NULL),
	(SELECT COALESCE(SUM(` + netOfBolloFattura + `), 0) FROM income WHERE income.contract_id = contract.id)`

var contractSelect = `SELECT ` + contractColumns + ` FROM contract`

func scanContract(row interface{ Scan(...any) error }) (contract, error) {
	var c contract
	err := row.Scan(&c.ID, &c.ClientID, &c.StartMonth, &c.EndMonth, &c.TotalCents,
		&c.ReceivedCents, &c.AccountedCents)
	return c, err
}

// clientContractDue is one Client's share of an active Contract this month —
// the Dashboard's Clienti card, one badge per Client. Client travels as a
// name for the same reason outstandingIncome's does: the card holds no
// Client list of its own to look a hidden one's name up in.
type clientContractDue struct {
	ClientID int64  `json:"client_id"`
	Client   string `json:"client"`
	DueCents int64  `json:"due_cents"`
}

// contractsDueThisMonth answers, per Client, this month's own straight-line
// share (InvoiceTargetCents, ADR-0012) for whichever Contract is currently
// inside its own range — start_month <= current <= end_month. A Contract not
// yet begun, or already past end_month, is left out entirely: neither is "an
// active one" in the sense the card asks about. A hidden Client is left out
// regardless of its Contract's figures, same as readNotYetInvoiced: hiding is
// how a Client is retired, and this card is a prompt to act, not a record of
// history.
//
// A Client already invoiced this calendar month for this Contract — an
// invoice_sent_date within it, whatever its amount or payment status — is
// left out too: the prompt is "send this month's invoice," and one already
// went out. A Client whose Contract is fully accounted for (invoiced at or
// past its total, this month or any other) is left out on the same terms
// InvoiceTargetCents already floors shortfall at zero for.
//
// Two Contracts for the same Client never overlap in range (contractOverlaps
// enforces it at write time), so this is at most one row per Client.
func contractsDueThisMonth(db *sql.DB, now func() time.Time) ([]clientContractDue, error) {
	current := now().Format(monthLayout)
	rows, err := db.Query(`SELECT `+contractColumns+`, client.name,
			EXISTS (
				SELECT 1 FROM income
				WHERE income.contract_id = contract.id
					AND substr(income.invoice_sent_date, 1, 7) = ?
			)
		FROM contract
		JOIN client ON client.id = contract.client_id
		WHERE contract.start_month <= ? AND contract.end_month >= ? AND client.hidden = 0`,
		current, current, current)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []clientContractDue{}
	for rows.Next() {
		var c contract
		var clientName string
		var invoicedThisMonth bool
		if err := rows.Scan(&c.ID, &c.ClientID, &c.StartMonth, &c.EndMonth, &c.TotalCents,
			&c.ReceivedCents, &c.AccountedCents, &clientName, &invoicedThisMonth); err != nil {
			return nil, err
		}
		if invoicedThisMonth {
			continue
		}
		if err := computeContractFigures(&c, current); err != nil {
			return nil, err
		}
		if c.InvoiceTargetCents <= 0 {
			continue
		}
		out = append(out, clientContractDue{
			ClientID: c.ClientID, Client: clientName, DueCents: c.InvoiceTargetCents,
		})
	}
	return out, rows.Err()
}

// computeContractFigures fills in ExpectedSoFarCents, InvoiceTargetCents and Overdue
// from the range, the total, and whatever contractSelect already scanned into
// ReceivedCents/AccountedCents — the whole of ADR-0012, recomputed fresh on
// every call rather than read off a stored schedule.
func computeContractFigures(c *contract, current string) error {
	totalMonths, err := monthsInclusive(c.StartMonth, c.EndMonth)
	if err != nil {
		return err
	}

	// Elapsed is 0 before the Contract starts (monthsInclusive already floors
	// a negative span at 0) and never counts past the Contract's own length —
	// that is the "capped at total_cents once end_month has passed" half of
	// the ticket, and it falls out of capping the month count rather than the
	// cents: total_cents * totalMonths / totalMonths is exactly total_cents,
	// with no separate clamp needed.
	elapsed, err := monthsInclusive(c.StartMonth, current)
	if err != nil {
		return err
	}
	if elapsed > totalMonths {
		elapsed = totalMonths
	}
	// Integer division truncates toward zero — both operands are
	// non-negative, so this is a plain floor, matching every other cents
	// figure in this codebase that is never asked to round up.
	c.ExpectedSoFarCents = c.TotalCents * int64(elapsed) / int64(totalMonths)

	// The shortfall everything still owed comes out of, floored at zero: a
	// Contract invoiced for more than its total asks for nothing further
	// rather than a negative invoice.
	shortfall := c.TotalCents - c.AccountedCents
	if shortfall < 0 {
		shortfall = 0
	}

	if current > c.EndMonth {
		// Past the end month there is no "months remaining" to divide the
		// shortfall by — the ticket's explicit edge case. The whole shortfall
		// is what's owed, shown as overdue — but only when something actually
		// still is: a Contract invoiced for its full total (or beyond) before
		// its own end month is not overdue just for having ended, and showing
		// it as such would flag a Client who owes nothing further.
		c.InvoiceTargetCents = shortfall
		c.Overdue = shortfall > 0
		return nil
	}

	// Months remaining runs from whichever is later of "now" and the start —
	// a Contract not yet begun still divides across its whole length — through
	// the end, inclusive: this month is one of the months still open to
	// invoice in. Always >= 1 here, since current <= end_month in this branch.
	from := current
	if c.StartMonth > from {
		from = c.StartMonth
	}
	remaining, err := monthsInclusive(from, c.EndMonth)
	if err != nil {
		return err
	}
	c.InvoiceTargetCents = shortfall / int64(remaining)
	return nil
}

// monthsInclusive counts the calendar months from a through b, both ends
// included — Jan through Jan is 1, Jan through Mar is 3 — which is what makes
// a straight-line share of elapsed time a plain count rather than a date
// subtraction to get wrong. b before a answers 0 rather than a negative
// count: "no months of this yet," not an error.
func monthsInclusive(a, b string) (int, error) {
	from, err := time.Parse(monthLayout, a)
	if err != nil {
		return 0, err
	}
	to, err := time.Parse(monthLayout, b)
	if err != nil {
		return 0, err
	}
	months := (to.Year()-from.Year())*12 + int(to.Month()-from.Month()) + 1
	if months < 0 {
		months = 0
	}
	return months, nil
}

// validate rejects at the door what generation would otherwise have to keep
// asking about. It touches no database, so every error it returns is a bad
// request and nothing else — the overlap refusal is a separate check because
// it does touch the database.
func (c *contract) validate() error {
	if c.TotalCents <= 0 {
		return errors.New("a contract needs a total above zero")
	}
	if err := validMonth("start_month", c.StartMonth); err != nil {
		return err
	}
	if err := validMonth("end_month", c.EndMonth); err != nil {
		return err
	}
	if c.EndMonth < c.StartMonth {
		return errors.New("end_month cannot be before start_month")
	}
	return nil
}
