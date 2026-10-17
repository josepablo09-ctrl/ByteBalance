package http

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"bytebalance/internal/domain"
	"bytebalance/internal/domain/finance"
	"bytebalance/internal/domain/habit"
	"bytebalance/internal/domain/user"
	"bytebalance/internal/usecase"
)

// UserService son los casos de uso de usuarios que expone la API.
type UserService interface {
	Register(ctx context.Context, in usecase.RegisterInput) (*user.User, error)
	Profile(ctx context.Context, userID domain.ID) (*user.User, error)
	UpdatePreferences(ctx context.Context, userID domain.ID, in usecase.PreferencesInput) (*user.User, error)
}

// AuthService es el caso de uso de inicio de sesión.
type AuthService interface {
	Login(ctx context.Context, email, password string) (usecase.AccessToken, error)
}

// FinanceService son los casos de uso de finanzas que expone la API.
type FinanceService interface {
	CreateCategory(ctx context.Context, userID domain.ID, name string, t finance.EntryType) (*finance.Category, error)
	ListCategories(ctx context.Context, userID domain.ID) ([]finance.Category, error)
	RecordTransaction(ctx context.Context, userID domain.ID, in usecase.RecordTransactionInput) (*finance.Transaction, error)
	ListTransactions(ctx context.Context, userID domain.ID, f finance.TransactionFilter) ([]finance.Transaction, error)
	UpdateTransaction(ctx context.Context, userID, transactionID domain.ID, in usecase.UpdateTransactionInput) (*finance.Transaction, error)
	DeleteTransaction(ctx context.Context, userID, transactionID domain.ID) error
	Balance(ctx context.Context, userID domain.ID, from, to time.Time) ([]finance.Balance, error)
}

// HabitService son los casos de uso de hábitos que expone la API.
type HabitService interface {
	CreateHabit(ctx context.Context, userID domain.ID, in usecase.CreateHabitInput) (*habit.Habit, error)
	ListHabits(ctx context.Context, userID domain.ID, activeOnly bool) ([]habit.Habit, error)
	ListLogs(ctx context.Context, userID, habitID domain.ID, from, to time.Time) ([]habit.Log, error)
	UpdateHabit(ctx context.Context, userID, habitID domain.ID, in usecase.UpdateHabitInput) (*habit.Habit, error)
	DeleteHabit(ctx context.Context, userID, habitID domain.ID) error
	DeleteLog(ctx context.Context, userID, habitID domain.ID, date time.Time) error
	LogCompletion(ctx context.Context, userID, habitID domain.ID, in usecase.LogCompletionInput) (*habit.Log, error)
}

// Services agrupa las dependencias de los handlers HTTP.
type Services struct {
	Users   UserService
	Auth    AuthService
	Finance FinanceService
	Habits  HabitService
	// Telegram es nil si el servidor no tiene bot configurado; sus endpoints
	// responden entonces 503.
	Telegram TelegramService
	// Insights es nil si el servidor no tiene llave de Gemini; el análisis
	// con IA responde entonces 503.
	Insights InsightService
	Tokens   TokenVerifier
	DB       Pinger
}

// Options configura el comportamiento transversal del router.
type Options struct {
	// AllowedOrigins son los orígenes (esquema://host[:puerto]) que pueden
	// llamar a la API desde el navegador. Vacío desactiva CORS.
	AllowedOrigins []string
	// TrustedProxyHops es el número de proxies de confianza delante de la API
	// (p. ej. 1 detrás de un ALB). Con 0 se ignora X-Forwarded-For y la IP del
	// cliente es la de la conexión.
	TrustedProxyHops int
}

// api contiene los handlers REST; cada método atiende un endpoint.
type api struct {
	Services
	opts               Options
	loginLimits        *loginThrottle
	telegramLinkLimits *limiter
	aiUserLimits       *limiter
	aiGlobalLimits     *limiter
}

// NewRouter construye el handler HTTP con todas las rutas de la API v1.
// Las rutas bajo /api/v1, salvo registro y login, exigen un token de acceso.
func NewRouter(s Services, opts Options) http.Handler {
	a := &api{
		Services:           s,
		opts:               opts,
		loginLimits:        newLoginThrottle(time.Now),
		telegramLinkLimits: newLimiter(telegramLinkBurst, telegramLinkInterval, time.Now),
		aiUserLimits:       newLimiter(aiUserBurst, aiUserInterval, time.Now),
		aiGlobalLimits:     newLimiter(aiGlobalBurst, aiGlobalInterval, time.Now),
	}
	protected := func(fn http.HandlerFunc) http.Handler { return requireAuth(s.Tokens, fn) }

	mux := http.NewServeMux()
	mux.Handle("GET /healthz", Health(s.DB))

	mux.HandleFunc("POST /api/v1/users", a.register)
	mux.HandleFunc("POST /api/v1/auth/login", a.login)
	mux.Handle("GET /api/v1/users/me", protected(a.profile))
	mux.Handle("PATCH /api/v1/users/me", protected(a.updatePreferences))

	mux.Handle("GET /api/v1/categories", protected(a.listCategories))
	mux.Handle("POST /api/v1/categories", protected(a.createCategory))
	mux.Handle("GET /api/v1/transactions", protected(a.listTransactions))
	mux.Handle("POST /api/v1/transactions", protected(a.recordTransaction))
	mux.Handle("PATCH /api/v1/transactions/{id}", protected(a.updateTransaction))
	mux.Handle("DELETE /api/v1/transactions/{id}", protected(a.deleteTransaction))
	mux.Handle("GET /api/v1/balance", protected(a.balance))

	mux.Handle("GET /api/v1/habits", protected(a.listHabits))
	mux.Handle("POST /api/v1/habits", protected(a.createHabit))
	mux.Handle("PATCH /api/v1/habits/{id}", protected(a.updateHabit))
	mux.Handle("DELETE /api/v1/habits/{id}", protected(a.deleteHabit))
	mux.Handle("GET /api/v1/habits/{id}/logs", protected(a.listLogs))
	mux.Handle("POST /api/v1/habits/{id}/logs", protected(a.logCompletion))
	mux.Handle("DELETE /api/v1/habits/{id}/logs/{date}", protected(a.deleteLog))

	mux.Handle("GET /api/v1/telegram", protected(a.telegramStatus))
	mux.Handle("POST /api/v1/telegram/link", protected(a.createTelegramLink))
	mux.Handle("DELETE /api/v1/telegram/link", protected(a.deleteTelegramLink))
	mux.Handle("POST /api/v1/telegram/test", protected(a.sendTelegramTest))

	mux.Handle("GET /api/v1/ai", protected(a.aiStatus))
	mux.Handle("POST /api/v1/ai/diagnose", protected(a.diagnose))

	return recoverPanics(cors(opts.AllowedOrigins, mux))
}

// recoverPanics convierte un panic en un handler en una respuesta 500 en lugar
// de cortar la conexión, y lo registra para diagnóstico.
func recoverPanics(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			v := recover()
			if v == nil {
				return
			}
			// net/http usa este panic para abortar la respuesta a propósito.
			if v == http.ErrAbortHandler {
				panic(v)
			}
			slog.ErrorContext(r.Context(), "panic al atender la petición", "panic", v, "method", r.Method, "path", r.URL.Path)
			writeError(w, http.StatusInternalServerError, "internal", "error interno del servidor")
		}()
		next.ServeHTTP(w, r)
	})
}
