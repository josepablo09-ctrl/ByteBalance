package http

import (
	"context"
	"net/http"
	"strings"

	"bytebalance/internal/domain"
)

// TokenVerifier valida un token de acceso y devuelve el ID del usuario.
type TokenVerifier interface {
	Verify(token string) (domain.ID, error)
}

type userIDKey struct{}

// requireAuth exige un token "Authorization: Bearer <token>" válido y deja el
// ID del usuario en el contexto de la petición. Si falta o no es válido
// responde 401 sin indicar el motivo.
func requireAuth(tokens TokenVerifier, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		scheme, token, ok := strings.Cut(r.Header.Get("Authorization"), " ")
		if !ok || !strings.EqualFold(scheme, "Bearer") || strings.TrimSpace(token) == "" {
			unauthorized(w)
			return
		}
		userID, err := tokens.Verify(strings.TrimSpace(token))
		if err != nil {
			unauthorized(w)
			return
		}
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), userIDKey{}, userID)))
	})
}

func unauthorized(w http.ResponseWriter) {
	w.Header().Set("WWW-Authenticate", `Bearer realm="bytebalance"`)
	writeError(w, http.StatusUnauthorized, "unauthorized", "token de acceso inválido o ausente")
}

// userIDFrom devuelve el ID del usuario autenticado. Solo debe usarse en
// handlers protegidos por requireAuth; fuera de ellos devuelve el ID cero.
func userIDFrom(r *http.Request) domain.ID {
	if id, ok := r.Context().Value(userIDKey{}).(domain.ID); ok {
		return id
	}
	return domain.ID{}
}
