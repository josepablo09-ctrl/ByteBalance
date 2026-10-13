# ADR 0002 — Límite de intentos de login

- **Estado:** aceptado
- **Fecha:** 2026-10-13

## Contexto

`POST /api/v1/auth/login` es el objetivo natural de ataques de fuerza bruta y
de *credential stuffing*. Necesitamos frenarlos sin agregar infraestructura
nueva en esta etapa.

## Decisiones

1. **Token bucket en memoria** (`internal/delivery/http/ratelimit.go`), solo
   con la stdlib. Es una preocupación de transporte, por eso vive en `delivery`.
2. **Doble clave:**
   - Por IP: ráfaga de 10 intentos, luego 1 cada 6 s (~10/min).
   - Por email (normalizado a minúsculas): ráfaga de 5, luego 1 por minuto.
     Frena ataques a una cuenta concreta desde muchas IPs.
   Se comprueba primero la IP, para que un cliente ya limitado no consuma los
   intentos de otras cuentas. Las IPv6 se agrupan por prefijo `/64`.
3. Al superar el límite se responde **429** con `Retry-After` y el intento no
   llega al caso de uso (no se ejecuta bcrypt). Se cuentan todos los intentos,
   no solo los fallidos.
4. **IP real detrás de proxies:** `TRUSTED_PROXY_HOPS` (por defecto 0) indica
   cuántos proxies de confianza hay delante (1 con un ALB). Se toma la entrada
   de `X-Forwarded-For` añadida por el proxy más externo; con 0 la cabecera se
   ignora, porque el cliente puede falsificarla.

## Consecuencias

- El límite es **por instancia**: con N réplicas el límite efectivo es N veces
  mayor y se reinicia al desplegar. Si escalamos horizontalmente, migrar el
  estado a Redis/ElastiCache o complementar con AWS WAF.
- El límite por email permite a un atacante bloquear temporalmente (minutos) el
  login de una cuenta ajena. Se acepta: el bloqueo es corto y se recupera solo.
- Configurar `TRUSTED_PROXY_HOPS` mayor que el número real de proxies permite
  falsificar la IP; menor hace que todos los clientes compartan la IP del proxy.
