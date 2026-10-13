# ADR 0003 — Política CORS

- **Estado:** aceptado
- **Fecha:** 2026-10-13

## Contexto

El frontend Svelte se servirá desde un origen distinto al de la API (en local,
el servidor de Vite en `http://localhost:5173`), así que el navegador necesita
cabeceras CORS para permitirle leer las respuestas.

## Decisiones

1. **Lista blanca explícita** en `CORS_ALLOWED_ORIGINS` (separada por comas).
   El origen recibido se compara literalmente y se refleja solo si está en la
   lista; **nunca** se usa `*`. Vacía desactiva CORS (solo mismo origen).
2. La configuración se valida al arrancar: cada entrada debe ser
   `esquema://host[:puerto]` sin ruta, y fuera de `development` solo `https`.
3. **Sin `Access-Control-Allow-Credentials`:** la API se autentica con
   `Authorization: Bearer`, no con cookies. Por la misma razón no hay riesgo
   CSRF y las peticiones de orígenes no permitidos se atienden sin cabeceras
   CORS (el navegador impide leerlas). Un preflight no permitido recibe 403.
4. Métodos `GET, POST, PATCH, DELETE`; cabeceras `Authorization, Content-Type`;
   se expone `Retry-After` para que el frontend muestre la espera tras un 429.
   El preflight se cachea 10 minutos.
5. Se responde `Vary: Origin` para que ninguna caché comparta respuestas entre
   orígenes.

## Consecuencias

- Si en el futuro se usan cookies de sesión, habrá que añadir
  `Allow-Credentials` y protección CSRF, y revisar el punto 3.
