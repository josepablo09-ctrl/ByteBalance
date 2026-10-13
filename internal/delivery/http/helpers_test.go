package http

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"bytebalance/internal/domain"
	"bytebalance/internal/domain/finance"
	"bytebalance/internal/domain/habit"
	"bytebalance/internal/domain/user"
	"bytebalance/internal/usecase"
)

const validToken = "token-valido"

var (
	testUserID    = domain.MustParseID("0199d3a4-5b6c-7d8e-9f01-23456789abcd")
	errNotStubbed = errors.New("método no configurado en el stub")
)

// stubTokens acepta únicamente validToken.
type stubTokens struct{}

func (stubTokens) Verify(token string) (domain.ID, error) {
	if token == validToken {
		return testUserID, nil
	}
	return domain.ID{}, errors.New("token inválido")
}

// Los stubs de servicios delegan en funciones configurables por test. Un
// método no configurado devuelve errNotStubbed, que se traduce en un 500.

type stubUsers struct {
	register          func(usecase.RegisterInput) (*user.User, error)
	updatePreferences func(domain.ID, usecase.PreferencesInput) (*user.User, error)
}

func (s stubUsers) Register(_ context.Context, in usecase.RegisterInput) (*user.User, error) {
	if s.register == nil {
		return nil, errNotStubbed
	}
	return s.register(in)
}

func (s stubUsers) UpdatePreferences(_ context.Context, id domain.ID, in usecase.PreferencesInput) (*user.User, error) {
	if s.updatePreferences == nil {
		return nil, errNotStubbed
	}
	return s.updatePreferences(id, in)
}

type stubAuth struct {
	login func(email, password string) (usecase.AccessToken, error)
}

func (s stubAuth) Login(_ context.Context, email, password string) (usecase.AccessToken, error) {
	if s.login == nil {
		return usecase.AccessToken{}, errNotStubbed
	}
	return s.login(email, password)
}

type stubFinance struct {
	createCategory    func(domain.ID, string, finance.EntryType) (*finance.Category, error)
	listCategories    func(domain.ID) ([]finance.Category, error)
	recordTransaction func(domain.ID, usecase.RecordTransactionInput) (*finance.Transaction, error)
	listTransactions  func(domain.ID, finance.TransactionFilter) ([]finance.Transaction, error)
	balance           func(domain.ID, time.Time, time.Time) ([]finance.Balance, error)
}

func (s stubFinance) CreateCategory(_ context.Context, id domain.ID, name string, t finance.EntryType) (*finance.Category, error) {
	if s.createCategory == nil {
		return nil, errNotStubbed
	}
	return s.createCategory(id, name, t)
}

func (s stubFinance) ListCategories(_ context.Context, id domain.ID) ([]finance.Category, error) {
	if s.listCategories == nil {
		return nil, errNotStubbed
	}
	return s.listCategories(id)
}

func (s stubFinance) RecordTransaction(_ context.Context, id domain.ID, in usecase.RecordTransactionInput) (*finance.Transaction, error) {
	if s.recordTransaction == nil {
		return nil, errNotStubbed
	}
	return s.recordTransaction(id, in)
}

func (s stubFinance) ListTransactions(_ context.Context, id domain.ID, f finance.TransactionFilter) ([]finance.Transaction, error) {
	if s.listTransactions == nil {
		return nil, errNotStubbed
	}
	return s.listTransactions(id, f)
}

func (s stubFinance) Balance(_ context.Context, id domain.ID, from, to time.Time) ([]finance.Balance, error) {
	if s.balance == nil {
		return nil, errNotStubbed
	}
	return s.balance(id, from, to)
}

type stubHabits struct {
	createHabit   func(domain.ID, usecase.CreateHabitInput) (*habit.Habit, error)
	logCompletion func(domain.ID, domain.ID, usecase.LogCompletionInput) (*habit.Log, error)
}

func (s stubHabits) CreateHabit(_ context.Context, id domain.ID, in usecase.CreateHabitInput) (*habit.Habit, error) {
	if s.createHabit == nil {
		return nil, errNotStubbed
	}
	return s.createHabit(id, in)
}

func (s stubHabits) LogCompletion(_ context.Context, userID, habitID domain.ID, in usecase.LogCompletionInput) (*habit.Log, error) {
	if s.logCompletion == nil {
		return nil, errNotStubbed
	}
	return s.logCompletion(userID, habitID, in)
}

// newTestRouter construye el router con stubs y opciones por defecto.
func newTestRouter(s Services) http.Handler {
	return NewRouter(withStubs(s), Options{})
}

// withStubs completa con stubs los servicios no indicados, que quedan sin configurar.
func withStubs(s Services) Services {
	if s.Users == nil {
		s.Users = stubUsers{}
	}
	if s.Auth == nil {
		s.Auth = stubAuth{}
	}
	if s.Finance == nil {
		s.Finance = stubFinance{}
	}
	if s.Habits == nil {
		s.Habits = stubHabits{}
	}
	if s.Tokens == nil {
		s.Tokens = stubTokens{}
	}
	if s.DB == nil {
		s.DB = pingerFunc(func(context.Context) error { return nil })
	}
	return s
}

// do envía una petición al handler; con authenticated añade un token válido.
func do(h http.Handler, method, target, body string, authenticated bool) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, target, strings.NewReader(body))
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	if authenticated {
		req.Header.Set("Authorization", "Bearer "+validToken)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

// decodeJSONBody interpreta el cuerpo de la respuesta en un mapa genérico.
func decodeJSONBody(t *testing.T, rec *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("respuesta no es JSON (%v): %s", err, rec.Body.String())
	}
	return body
}

// errorCode devuelve error.code de una respuesta de error.
func errorCode(t *testing.T, rec *httptest.ResponseRecorder) string {
	t.Helper()
	detail, ok := decodeJSONBody(t, rec)["error"].(map[string]any)
	if !ok {
		t.Fatalf("respuesta sin objeto error: %s", rec.Body.String())
	}
	code, ok := detail["code"].(string)
	if !ok {
		t.Fatalf("error sin code: %s", rec.Body.String())
	}
	return code
}

// expectStatus falla el test si el código de estado no es el esperado.
func expectStatus(t *testing.T, rec *httptest.ResponseRecorder, want int) {
	t.Helper()
	if rec.Code != want {
		t.Fatalf("status = %d, se esperaba %d; body: %s", rec.Code, want, rec.Body.String())
	}
}
