# ADR 0008 — Despliegue en AWS

- **Estado:** aceptado
- **Fecha:** 2026-10-17

## Contexto

ByteBalance pasa a producción en AWS. El backend es un binario Go con procesos de fondo
(bot de Telegram por long polling y recordatorios) y límites de uso en memoria. El
frontend es una SPA estática. Los pasos operativos están en `DEPLOYMENT.md`; aquí se
registra el porqué.

## Decisiones

1. **Imagen multi-stage sobre `distroless/static:nonroot`.**
   - El binario es estático (`CGO_ENABLED=0`, `-trimpath -s -w`) y la base de zonas
     horarias va incrustada.
   - La imagen pesa ~25 MB, no tiene shell ni gestor de paquetes y corre sin
     privilegios.
   - No contiene secretos: la configuración llega por entorno desde Secrets Manager.
2. **Migraciones en una imagen aparte** (`--target migrate`, sobre
   `migrate/migrate:v4.19.1`). Se ejecutan como tarea de un solo uso antes de cada
   versión, sin agregar golang-migrate como dependencia del binario. Lee `DATABASE_URL`
   del entorno para que la contraseña no quede en la línea de comandos.
3. **Una sola instancia de la API.** El long polling de Telegram, el job de
   recordatorios y los límites de uso viven en el proceso; con varias réplicas
   chocarían o se duplicarían. Por eso se recomienda **EC2 detrás de un ALB**. App
   Runner solo es válido sin Telegram, porque deja de asignar CPU sin tráfico y los
   procesos de fondo se detendrían. Escalar horizontalmente requeriría sacar el bot y
   el scheduler a un worker y los límites a un almacén compartido.
4. **Cabeceras de seguridad en la API**, en todas las respuestas, incluidos los errores:
   - `Content-Security-Policy: default-src 'none'; frame-ancestors 'none'` (solo sirve
     JSON).
   - `nosniff`, `X-Frame-Options: DENY`, `Referrer-Policy: no-referrer`.
   - `Cache-Control: no-store`, porque las respuestas tienen datos financieros.
   - HSTS fuera de desarrollo.

   CORS sigue leyendo `CORS_ALLOWED_ORIGINS` y exige https fuera de desarrollo (ADR
   0003).
5. **Frontend en S3 privado + CloudFront con OAC.**
   - El fallback de la SPA se hace con páginas de error 403/404 → `/index.html`.
   - El build de producción falla si `VITE_API_URL` falta o no usa https, porque en una
     SPA el error solo se vería con el sitio ya desplegado.
   - SvelteKit genera una CSP por `<meta>`, con el hash de su script de arranque y
     `connect-src` limitado a la API.
   - Las cabeceras que un `<meta>` no admite (HSTS, `X-Frame-Options`) las añade
     CloudFront con la política gestionada `SecurityHeadersPolicy`.

## Consecuencias

- Los despliegues de la API tienen unos segundos de corte: se detiene la versión
  anterior antes de iniciar la nueva, para no tener dos bots a la vez.
- La CSP del frontend admite `style-src 'unsafe-inline'`, porque Svelte aplica estilos en
  línea (barras de progreso). Los scripts siguen restringidos a los propios y al hash del
  arranque.
- Al cambiar el dominio de la API hay que recompilar el frontend: la URL y la CSP se
  fijan en el build.
