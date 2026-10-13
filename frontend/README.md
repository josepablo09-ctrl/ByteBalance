# ByteBalance — Frontend

SPA en **SvelteKit 3 (Svelte 5 + Vite)** con **Tailwind CSS 4**. Consume la API REST
del backend Go y se compila a archivos estáticos (`adapter-static`).

## Requisitos

- Node.js 24 LTS
- El backend corriendo en `http://localhost:8080` con
  `CORS_ALLOWED_ORIGINS=http://localhost:5173` en su `.env`.

## Puesta en marcha

```bash
cd frontend
cp .env.example .env     # VITE_API_URL=http://localhost:8080/api/v1
npm install
npx playwright install chromium   # solo la primera vez: navegador de los tests de componentes
npm run dev              # http://localhost:5173 (puerto fijo, coincide con CORS)
```

> Las variables `VITE_*` se incrustan en el bundle público. **Nunca** pongas secretos en `frontend/.env`.

## Scripts

| Comando           | Descripción                                                       |
| ----------------- | ----------------------------------------------------------------- |
| `npm run dev`     | Servidor de desarrollo                                            |
| `npm run build`   | Build de producción en `build/`                                   |
| `npm run preview` | Sirve el build localmente                                         |
| `npm run lint`    | Prettier + ESLint                                                 |
| `npm run check`   | svelte-check (tipos JSDoc)                                        |
| `npm run test`    | Vitest: unitarios en Node y de componentes en Chromium (headless) |

## Estructura

```
src/
├── lib/
│   ├── api/          # Cliente HTTP (único punto que habla con el backend)
│   │   ├── client.js # fetch + Bearer token + ApiError
│   │   ├── auth.js   # login, registro, perfil
│   │   ├── errors.js # ApiError → mensaje para el usuario
│   │   └── index.js  # instancia de la app conectada a la sesión
│   ├── components/   # UI reutilizable (Button, TextField, LoginForm…)
│   ├── stores/       # Estado global (session)
│   ├── config.js     # VITE_API_URL validada
│   └── validation.js # Reglas espejo del backend
└── routes/
    ├── (auth)/       # login, register — redirigen a / si hay sesión
    └── (app)/        # pantallas privadas — redirigen a /login sin sesión
```

## Notas de SvelteKit 3

- El alias de `src/lib` es `#lib/…` (no `$lib`), y `$app/environment` ahora es `$app/env`.
- `resolve()` de `$app/paths` recibe IDs de ruta tipados, p. ej. `resolve('/(auth)/login')`;
  así un enlace a una ruta inexistente falla en `npm run check`.
