package http

import (
	"log/slog"
	"math"
	"net"
	"net/http"
	"net/netip"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Límites de intentos de login. Por IP se admite una ráfaga de 10 intentos y
// luego uno cada 6 s (~10/min). Por email, más estricto para frenar ataques a
// una cuenta concreta desde muchas IPs: 5 intentos y luego uno por minuto.
const (
	loginIPBurst       = 10
	loginIPInterval    = 6 * time.Second
	loginEmailBurst    = 5
	loginEmailInterval = time.Minute
)

// limiter es un limitador token bucket por clave, en memoria y seguro para uso
// concurrente. Cada clave dispone de burst intentos que se recuperan a razón
// de uno cada interval.
//
// El estado es local a la instancia: con varias réplicas detrás de un balanceador
// el límite efectivo se multiplica por el número de réplicas.
type limiter struct {
	burst    int
	interval time.Duration
	now      func() time.Time

	mu        sync.Mutex
	buckets   map[string]bucket
	nextSweep time.Time
}

type bucket struct {
	tokens float64
	last   time.Time
}

func newLimiter(burst int, interval time.Duration, now func() time.Time) *limiter {
	return &limiter{
		burst:    burst,
		interval: interval,
		now:      clockOrDefault(now),
		buckets:  map[string]bucket{},
	}
}

// allow consume un intento de key. Si no le quedan intentos devuelve false y
// el tiempo que falta para el siguiente.
func (l *limiter) allow(key string) (bool, time.Duration) {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := l.now()
	l.sweep(now)

	b, ok := l.buckets[key]
	if !ok {
		b = bucket{tokens: float64(l.burst), last: now}
	}
	elapsed := max(now.Sub(b.last), 0)
	b.tokens = min(float64(l.burst), b.tokens+float64(elapsed)/float64(l.interval))
	b.last = now

	if b.tokens < 1 {
		l.buckets[key] = b
		return false, time.Duration((1 - b.tokens) * float64(l.interval))
	}
	b.tokens--
	l.buckets[key] = b
	return true, 0
}

// sweep descarta, como mucho una vez por ciclo de recarga completa, las claves
// que ya recuperaron todos sus intentos: equivalen a una clave nueva. Así la
// memoria no crece indefinidamente con cada IP o email visto.
func (l *limiter) sweep(now time.Time) {
	if now.Before(l.nextSweep) {
		return
	}
	fullRefill := l.interval * time.Duration(l.burst)
	for key, b := range l.buckets {
		if now.Sub(b.last) >= fullRefill {
			delete(l.buckets, key)
		}
	}
	l.nextSweep = now.Add(fullRefill)
}

// loginThrottle limita los intentos de login por IP de origen y por email.
type loginThrottle struct {
	byIP    *limiter
	byEmail *limiter
}

func newLoginThrottle(now func() time.Time) *loginThrottle {
	return &loginThrottle{
		byIP:    newLimiter(loginIPBurst, loginIPInterval, now),
		byEmail: newLimiter(loginEmailBurst, loginEmailInterval, now),
	}
}

// allow registra un intento de login. Se comprueba primero la IP para que un
// cliente ya limitado no pueda seguir consumiendo los intentos de otras cuentas.
func (t *loginThrottle) allow(ip, email string) (bool, time.Duration) {
	if ok, wait := t.byIP.allow(ipKey(ip)); !ok {
		return false, wait
	}
	return t.byEmail.allow(strings.ToLower(strings.TrimSpace(email)))
}

// ipKey agrupa las direcciones IPv6 por su prefijo /64, que suele asignarse
// completo a un mismo cliente; si no, rotar direcciones evadiría el límite.
func ipKey(ip string) string {
	addr, err := netip.ParseAddr(ip)
	if err != nil {
		return ip
	}
	addr = addr.Unmap()
	if addr.Is4() {
		return addr.String()
	}
	prefix, err := addr.Prefix(64)
	if err != nil {
		return addr.String()
	}
	return prefix.String()
}

// clientIP devuelve la IP del cliente. Con trustedHops > 0 la API corre detrás
// de ese número de proxies de confianza (p. ej. un ALB de AWS), que añaden al
// final de X-Forwarded-For la IP que ven. Se toma la entrada que añadió el
// proxy más externo: las anteriores las controla el cliente y pueden ser falsas.
// Si la cabecera no tiene suficientes entradas se usa la dirección de la conexión.
func clientIP(r *http.Request, trustedHops int) string {
	if trustedHops > 0 {
		var hops []string
		for _, header := range r.Header.Values("X-Forwarded-For") {
			for _, hop := range strings.Split(header, ",") {
				if hop = strings.TrimSpace(hop); hop != "" {
					hops = append(hops, hop)
				}
			}
		}
		if len(hops) >= trustedHops {
			if addr, err := netip.ParseAddr(hops[len(hops)-trustedHops]); err == nil {
				return addr.String()
			}
		}
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

// tooManyRequests responde 429 con message, indicando en Retry-After los segundos de espera.
func tooManyRequests(w http.ResponseWriter, r *http.Request, wait time.Duration, message string) {
	// No se registran la IP, el email ni el usuario: son datos personales.
	slog.WarnContext(r.Context(), "petición limitada", "path", r.URL.Path)
	seconds := max(int(math.Ceil(wait.Seconds())), 1)
	w.Header().Set("Retry-After", strconv.Itoa(seconds))
	writeError(w, http.StatusTooManyRequests, "too_many_requests", message)
}

// clockOrDefault devuelve now, o time.Now si es nil.
func clockOrDefault(now func() time.Time) func() time.Time {
	if now == nil {
		return time.Now
	}
	return now
}
