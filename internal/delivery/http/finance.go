package http

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"bytebalance/internal/domain"
	"bytebalance/internal/domain/finance"
	"bytebalance/internal/usecase"
)

type createCategoryRequest struct {
	Name string `json:"name"`
	Type string `json:"type"`
}

type categoryResponse struct {
	ID        domain.ID         `json:"id"`
	Name      string            `json:"name"`
	Type      finance.EntryType `json:"type"`
	CreatedAt time.Time         `json:"created_at"`
}

func newCategoryResponse(c finance.Category) categoryResponse {
	return categoryResponse{ID: c.ID, Name: c.Name, Type: c.Type, CreatedAt: c.CreatedAt}
}

// recordTransactionRequest usa punteros en los campos opcionales u obligatorios
// cuyo valor cero sería válido en JSON, para poder detectar su ausencia.
type recordTransactionRequest struct {
	Type        string     `json:"type"`
	AmountCents *int64     `json:"amount_cents"`
	Currency    string     `json:"currency"`
	CategoryID  *domain.ID `json:"category_id"`
	Description string     `json:"description"`
	OccurredAt  *time.Time `json:"occurred_at"`
}

// transactionResponse expresa los montos en centavos para evitar decimales flotantes.
type transactionResponse struct {
	ID          domain.ID         `json:"id"`
	Type        finance.EntryType `json:"type"`
	AmountCents int64             `json:"amount_cents"`
	Currency    domain.Currency   `json:"currency"`
	CategoryID  *domain.ID        `json:"category_id"`
	Description string            `json:"description"`
	OccurredAt  time.Time         `json:"occurred_at"`
	CreatedAt   time.Time         `json:"created_at"`
}

func newTransactionResponse(t finance.Transaction) transactionResponse {
	return transactionResponse{
		ID:          t.ID,
		Type:        t.Type,
		AmountCents: t.Amount.Cents,
		Currency:    t.Amount.Currency,
		CategoryID:  t.CategoryID,
		Description: t.Description,
		OccurredAt:  t.OccurredAt,
		CreatedAt:   t.CreatedAt,
	}
}

// balanceResponse resume una moneda. net_cents es el balance total (ingresos -
// gastos); savings_cents, lo apartado menos lo retirado; available_cents, el
// disponible real para gastar (net_cents - savings_cents).
type balanceResponse struct {
	Currency       domain.Currency `json:"currency"`
	IncomeCents    int64           `json:"income_cents"`
	ExpenseCents   int64           `json:"expense_cents"`
	NetCents       int64           `json:"net_cents"`
	SavedCents     int64           `json:"saved_cents"`
	WithdrawnCents int64           `json:"withdrawn_cents"`
	SavingsCents   int64           `json:"savings_cents"`
	AvailableCents int64           `json:"available_cents"`
}

// createCategory atiende POST /api/v1/categories.
func (a *api) createCategory(w http.ResponseWriter, r *http.Request) {
	var req createCategoryRequest
	if err := decodeJSON(w, r, &req); err != nil {
		respondError(w, r, err)
		return
	}
	if strings.TrimSpace(req.Name) == "" || req.Type == "" {
		respondError(w, r, badRequest("name y type son obligatorios"))
		return
	}

	c, err := a.Finance.CreateCategory(r.Context(), userIDFrom(r), req.Name, finance.EntryType(req.Type))
	if err != nil {
		respondError(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, newCategoryResponse(*c))
}

// listCategories atiende GET /api/v1/categories.
func (a *api) listCategories(w http.ResponseWriter, r *http.Request) {
	categories, err := a.Finance.ListCategories(r.Context(), userIDFrom(r))
	if err != nil {
		respondError(w, r, err)
		return
	}
	resp := make([]categoryResponse, 0, len(categories))
	for _, c := range categories {
		resp = append(resp, newCategoryResponse(c))
	}
	writeJSON(w, http.StatusOK, map[string]any{"categories": resp})
}

// recordTransaction atiende POST /api/v1/transactions.
func (a *api) recordTransaction(w http.ResponseWriter, r *http.Request) {
	var req recordTransactionRequest
	if err := decodeJSON(w, r, &req); err != nil {
		respondError(w, r, err)
		return
	}
	if req.Type == "" || req.AmountCents == nil {
		respondError(w, r, badRequest("type y amount_cents son obligatorios"))
		return
	}

	in := usecase.RecordTransactionInput{
		Type:        finance.EntryType(req.Type),
		AmountCents: *req.AmountCents,
		Currency:    req.Currency,
		CategoryID:  req.CategoryID,
		Description: req.Description,
	}
	if req.OccurredAt != nil {
		in.OccurredAt = *req.OccurredAt
	}
	t, err := a.Finance.RecordTransaction(r.Context(), userIDFrom(r), in)
	if err != nil {
		respondError(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, newTransactionResponse(*t))
}

// listTransactions atiende GET /api/v1/transactions?from&to&type&category_id&limit&offset.
func (a *api) listTransactions(w http.ResponseWriter, r *http.Request) {
	f, err := parseTransactionFilter(r)
	if err != nil {
		respondError(w, r, err)
		return
	}
	transactions, err := a.Finance.ListTransactions(r.Context(), userIDFrom(r), f)
	if err != nil {
		respondError(w, r, err)
		return
	}
	resp := make([]transactionResponse, 0, len(transactions))
	for _, t := range transactions {
		resp = append(resp, newTransactionResponse(t))
	}
	writeJSON(w, http.StatusOK, map[string]any{"transactions": resp})
}

func parseTransactionFilter(r *http.Request) (finance.TransactionFilter, error) {
	q := r.URL.Query()
	var (
		f   finance.TransactionFilter
		err error
	)
	if f.From, err = queryTime(q, "from"); err != nil {
		return f, err
	}
	if f.To, err = queryTime(q, "to"); err != nil {
		return f, err
	}
	if f.CategoryID, err = queryID(q, "category_id"); err != nil {
		return f, err
	}
	if f.Limit, err = queryInt(q, "limit"); err != nil {
		return f, err
	}
	if f.Offset, err = queryInt(q, "offset"); err != nil {
		return f, err
	}
	f.Type = finance.EntryType(q.Get("type"))
	return f, nil
}

// balance atiende GET /api/v1/balance?from&to.
func (a *api) balance(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	from, err := queryTime(q, "from")
	if err != nil {
		respondError(w, r, err)
		return
	}
	to, err := queryTime(q, "to")
	if err != nil {
		respondError(w, r, err)
		return
	}

	balances, err := a.Finance.Balance(r.Context(), userIDFrom(r), from, to)
	if err != nil {
		respondError(w, r, err)
		return
	}
	resp := make([]balanceResponse, 0, len(balances))
	for _, b := range balances {
		resp = append(resp, balanceResponse{
			Currency:       b.Currency,
			IncomeCents:    b.IncomeCents,
			ExpenseCents:   b.ExpenseCents,
			NetCents:       b.NetCents(),
			SavedCents:     b.SavedCents,
			WithdrawnCents: b.WithdrawnCents,
			SavingsCents:   b.SavingsCents(),
			AvailableCents: b.AvailableCents(),
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{"balances": resp})
}

// updateTransactionRequest usa punteros para distinguir "no enviado" de "vacío".
// CategoryID se lee en crudo para distinguir además "no enviado" (no cambia) de
// null (quita la categoría).
type updateTransactionRequest struct {
	Type        *string         `json:"type"`
	AmountCents *int64          `json:"amount_cents"`
	Currency    *string         `json:"currency"`
	CategoryID  json.RawMessage `json:"category_id"`
	Description *string         `json:"description"`
	OccurredAt  *time.Time      `json:"occurred_at"`
}

// updateTransaction atiende PATCH /api/v1/transactions/{id}. Solo modifica los
// campos enviados.
func (a *api) updateTransaction(w http.ResponseWriter, r *http.Request) {
	transactionID, err := pathID(r, "id")
	if err != nil {
		respondError(w, r, err)
		return
	}
	var req updateTransactionRequest
	if err := decodeJSON(w, r, &req); err != nil {
		respondError(w, r, err)
		return
	}

	in := usecase.UpdateTransactionInput{
		AmountCents: req.AmountCents,
		Currency:    req.Currency,
		Description: req.Description,
		OccurredAt:  req.OccurredAt,
	}
	if req.Type != nil {
		t := finance.EntryType(*req.Type)
		in.Type = &t
	}
	if len(req.CategoryID) > 0 {
		in.SetCategory = true
		if string(req.CategoryID) != "null" {
			var id domain.ID
			if err := json.Unmarshal(req.CategoryID, &id); err != nil {
				respondError(w, r, badRequest("category_id debe ser un UUID o null"))
				return
			}
			in.CategoryID = &id
		}
	}
	if in == (usecase.UpdateTransactionInput{}) {
		respondError(w, r, badRequest("indica al menos uno de: type, amount_cents, currency, category_id, description, occurred_at"))
		return
	}

	t, err := a.Finance.UpdateTransaction(r.Context(), userIDFrom(r), transactionID, in)
	if err != nil {
		respondError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, newTransactionResponse(*t))
}

// deleteTransaction atiende DELETE /api/v1/transactions/{id}. Responde 204 sin cuerpo.
func (a *api) deleteTransaction(w http.ResponseWriter, r *http.Request) {
	transactionID, err := pathID(r, "id")
	if err != nil {
		respondError(w, r, err)
		return
	}
	if err := a.Finance.DeleteTransaction(r.Context(), userIDFrom(r), transactionID); err != nil {
		respondError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
