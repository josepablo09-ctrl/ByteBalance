# ADR 0004 — Frontend como SPA y manejo de la sesión

- **Estado:** aceptado
- **Fecha:** 2026-10-13

## Contexto

El backend es una API REST en Go que autentica con `Authorization: Bearer <JWT>`
(24 h de vigencia). El frontend se construye con Svelte, Vite y Tailwind, y se
desplegará en AWS por separado del backend.

## Decisiones

1. **SvelteKit 3 como SPA estática** (`adapter-static` con `fallback: index.html`,
   `ssr = false`). Encaja con la estructura `src/routes` + `static/` del repo y
   permite servir el frontend desde S3 + CloudFront sin un servidor Node.
2. **JavaScript con JSDoc** (convención del repo) verificado con `svelte-check`
   en modo estricto.
3. **Un único cliente HTTP** (`src/lib/api/client.js`): adjunta el Bearer token,
   serializa JSON y convierte las respuestas de error en `ApiError` con el `code`
   estable de la API y `Retry-After`. Un 401 cierra la sesión **solo si se envió
   un token**; el 401 de credenciales incorrectas no la toca.
4. **Sesión en `localStorage`** (`token` + `expiresAt`), descartada al expirar o
   si su contenido es inválido. Se elige frente a `sessionStorage` o memoria para
   que la sesión sobreviva a recargas y pestañas nuevas.
5. **Guardas por grupo de rutas:** `(app)` exige sesión y `(auth)` redirige al
   inicio si ya la hay. Son solo de UX: la autorización real la aplica la API.
6. Las validaciones del cliente replican las del backend (contraseña de 8
   caracteres a 72 bytes, nombre ≤ 150, email ≤ 255) solo para dar feedback
   inmediato; el servidor vuelve a validar todo.

## Consecuencias

- Un token en `localStorage` es legible por cualquier script que logre
  ejecutarse en la página (XSS). Mitigaciones: Svelte escapa todo el contenido
  por defecto, no se usa `{@html}`, y en producción se servirá con una CSP
  estricta. Si el riesgo aumenta (p. ej. scripts de terceros), migrar a una
  cookie `HttpOnly` + `SameSite=Strict`, lo que exige protección CSRF y
  `Access-Control-Allow-Credentials` (ver ADR 0003).
- No hay refresh token: al expirar a las 24 h el usuario vuelve a iniciar sesión.
