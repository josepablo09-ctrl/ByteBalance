package http

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"bytebalance/internal/usecase"
)

// fakeClock es un reloj manual para los tests del limitador.
type fakeClock struct {
	mu sync.Mutex
	t  time.Time
}

func (c *fakeClock) now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.t
}

func (c *fakeClock) advance(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.t = c.t.Add(d)
}

func TestLimiter(t *testing.T) {
	clock := &fakeClock{t: time.Date(2026, 10, 13, 12, 0, 0, 0, time.UTC)}
	l := newLimiter(3, 10*time.Second, clock.now)

	for i := range 3 {
		if ok, _ := l.allow("a"); !ok {
			t.Fatalf("intento %d rechazado dentro de la ráfaga", i+1)
		}
	}
	ok, wait := l.allow("a")
	if ok || wait != 10*time.Second {
		t.Fatalf("allow() tras la ráfaga = %v, %v; se esperaba false, 10s", ok, wait)
	}
	if ok, _ := l.allow("b"); !ok {
		t.Error("las claves deben ser independientes")
	}

	clock.advance(4 * time.Second)
	if ok, wait := l.allow("a"); ok || wait != 6*time.Second {
		t.Errorf("allow() a mitad de recarga = %v, %v; se esperaba false, 6s", ok, wait)
	}
	clock.advance(6 * time.Second)
	if ok, _ := l.allow("a"); !ok {
		t.Error("se esperaba recuperar un intento tras el intervalo")
	}
	if ok, _ := l.allow("a"); ok {
		t.Error("solo debía recuperarse un intento")
	}
}

func TestLimiterSweepsRefilledKeys(t *testing.T) {
	clock := &fakeClock{t: time.Date(2026, 10, 13, 12, 0, 0, 0, time.UTC)}
	l := newLimiter(2, time.Second, clock.now)
	for i := range 100 {
		l.allow(fmt.Sprintf("ip-%d", i))
	}
	clock.advance(2 * time.Second)
	l.allow("nueva")
	if len(l.buckets) != 1 {
		t.Errorf("quedan %d claves; se esperaba descartar las ya recargadas", len(l.buckets))
	}
}

func TestLoginThrottle(t *testing.T) {
	t.Run("por IP con emails distintos", func(t *testing.T) {
		th := newLoginThrottle(nil)
		for i := range loginIPBurst {
			if ok, _ := th.allow("203.0.113.7", fmt.Sprintf("u%d@example.com", i)); !ok {
				t.Fatalf("intento %d rechazado", i+1)
			}
		}
		if ok, _ := th.allow("203.0.113.7", "otro@example.com"); ok {
			t.Error("se esperaba limitar la IP")
		}
		if ok, _ := th.allow("198.51.100.1", "otro@example.com"); !ok {
			t.Error("otra IP no debe verse afectada")
		}
	})
	t.Run("por email desde IPs distintas, sin distinguir mayúsculas", func(t *testing.T) {
		th := newLoginThrottle(nil)
		for i := range loginEmailBurst {
			if ok, _ := th.allow(fmt.Sprintf("203.0.113.%d", i), "ana@example.com"); !ok {
				t.Fatalf("intento %d rechazado", i+1)
			}
		}
		if ok, _ := th.allow("198.51.100.1", " ANA@example.com "); ok {
			t.Error("se esperaba limitar el email")
		}
	})
}

func TestIPKey(t *testing.T) {
	tests := map[string]string{
		"203.0.113.7":             "203.0.113.7",
		"::ffff:203.0.113.7":      "203.0.113.7",
		"2001:db8:1:2:aaaa::1":    "2001:db8:1:2::/64",
		"2001:db8:1:2:bbbb::9":    "2001:db8:1:2::/64",
		"no-es-una-ip":            "no-es-una-ip",
		"2001:db8:1:3:aaaa:0:0:1": "2001:db8:1:3::/64",
	}
	for in, want := range tests {
		if got := ipKey(in); got != want {
			t.Errorf("ipKey(%q) = %q, se esperaba %q", in, got, want)
		}
	}
}

func TestClientIP(t *testing.T) {
	tests := []struct {
		name string
		xff  []string
		hops int
		want string
	}{
		{name: "sin proxy ignora X-Forwarded-For", xff: []string{"1.1.1.1"}, want: "10.0.0.5"},
		{name: "un proxy toma la última entrada", xff: []string{"6.6.6.6, 203.0.113.7"}, hops: 1, want: "203.0.113.7"},
		{name: "dos proxies", xff: []string{"6.6.6.6, 203.0.113.7, 10.0.1.1"}, hops: 2, want: "203.0.113.7"},
		{name: "varias cabeceras", xff: []string{"6.6.6.6", "203.0.113.7"}, hops: 1, want: "203.0.113.7"},
		{name: "cabecera ausente", hops: 1, want: "10.0.0.5"},
		{name: "menos entradas que proxies", xff: []string{"203.0.113.7"}, hops: 2, want: "10.0.0.5"},
		{name: "entrada inválida", xff: []string{"basura"}, hops: 1, want: "10.0.0.5"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/", nil)
			req.RemoteAddr = "10.0.0.5:51234"
			for _, v := range tt.xff {
				req.Header.Add("X-Forwarded-For", v)
			}
			if got := clientIP(req, tt.hops); got != tt.want {
				t.Errorf("clientIP() = %q, se esperaba %q", got, tt.want)
			}
		})
	}
}

func TestLoginRateLimited(t *testing.T) {
	var calls int
	h := newTestRouter(Services{Auth: stubAuth{login: func(string, string) (usecase.AccessToken, error) {
		calls++
		return usecase.AccessToken{}, usecase.ErrInvalidCredentials
	}}})
	login := func(email string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login",
			strings.NewReader(`{"email":"`+email+`","password":"incorrecta"}`))
		req.RemoteAddr = "203.0.113.7:40000"
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		return rec
	}

	for i := range loginEmailBurst {
		expectStatus(t, login("ana@example.com"), http.StatusUnauthorized)
		if calls != i+1 {
			t.Fatalf("Login llamado %d veces, se esperaban %d", calls, i+1)
		}
	}
	rec := login("ana@example.com")
	expectStatus(t, rec, http.StatusTooManyRequests)
	if code := errorCode(t, rec); code != "too_many_requests" {
		t.Errorf("code = %q", code)
	}
	if got := rec.Header().Get("Retry-After"); got != "60" {
		t.Errorf("Retry-After = %q, se esperaba 60", got)
	}
	if calls != loginEmailBurst {
		t.Errorf("un intento limitado no debe llegar al caso de uso (%d llamadas)", calls)
	}
	// Otra cuenta desde la misma IP sigue pudiendo intentar.
	expectStatus(t, login("otra@example.com"), http.StatusUnauthorized)
}
