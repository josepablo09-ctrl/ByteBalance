# CLAUDE.md — ByteBalance

Guía de contexto y convenciones para Claude Code. Léela completa antes de modificar el repositorio.

## 1. Visión General

**ByteBalance** es una aplicación web de hábitos y finanzas personales con IA integrada y notificaciones automáticas.

| Capa | Tecnología |
|---|---|
| Backend | Go (arquitectura limpia / modular) |
| Frontend | Svelte + Tailwind CSS |
| Base de datos | PostgreSQL |
| IA | Google Gemini API (análisis e insights) |
| Notificaciones | Telegram Bot API (alertas y reportes) |
| Infraestructura | AWS |

---

## 2. Arquitectura y Estructura del Repositorio

```
bytebalance/
├── cmd/
│   └── api/                # Entrypoint del backend (main.go). Solo wiring, sin lógica de negocio.
├── internal/               # Código privado de la app (no importable externamente)
│   ├── domain/             # Entidades y reglas de negocio puras (sin dependencias externas)
│   ├── usecase/            # Casos de uso / servicios de aplicación (orquestan el dominio)
│   ├── repository/         # Interfaces + implementaciones de persistencia (PostgreSQL)
│   ├── delivery/
│   │   └── http/           # Handlers, routers, middlewares, DTOs de request/response
│   ├── integration/
│   │   ├── gemini/         # Cliente de Gemini API
│   │   └── telegram/       # Cliente de Telegram Bot API
│   ├── config/             # Carga y validación de configuración/env
│   └── scheduler/          # Jobs programados (reportes, recordatorios)
├── pkg/                    # Utilidades genéricas reutilizables y sin lógica de negocio (logger, validators)
├── migrations/             # Migraciones SQL versionadas (up/down)
├── frontend/               # App Svelte + Tailwind (proyecto independiente con su package.json)
│   ├── src/
│   │   ├── lib/
│   │   │   ├── components/ # Componentes reutilizables
│   │   │   ├── stores/     # Estado global (Svelte stores)
│   │   │   └── api/        # Cliente HTTP hacia el backend
│   │   └── routes/         # Páginas / vistas
│   └── static/
├── deploy/                 # IaC, Dockerfiles, configuración AWS
├── scripts/                # Scripts de desarrollo y utilidades
├── .env.example            # Plantilla de variables (SIN valores reales)
└── CLAUDE.md
```

### Reglas de dependencia (Clean Architecture)

- Flujo permitido: `delivery → usecase → domain` y `repository → domain`.
- `domain` **no importa** nada de `usecase`, `repository`, `delivery` ni de librerías externas (salvo stdlib).
- `usecase` depende de **interfaces** (repositorios, clientes de Gemini/Telegram), nunca de implementaciones concretas.
- Las implementaciones concretas se inyectan en `cmd/api/main.go` (inyección de dependencias manual).
- Los handlers HTTP no contienen lógica de negocio: parsean, validan, llaman al caso de uso y responden.
- Los módulos de dominio (ej. `habits`, `finance`, `insights`, `notifications`) deben mantenerse cohesivos y con bajo acoplamiento entre sí.

---

## 3. Convenciones de Código y Estilo

### Go

- Formatear siempre con `gofmt` (idealmente `goimports`). Verificar con `go vet` y `golangci-lint`.
- **Manejo explícito de errores**: nunca ignorar errores con `_`. Envolver con contexto: `fmt.Errorf("crear hábito: %w", err)`. Usar `errors.Is/As` para comparar.
- Definir errores de dominio como variables/tipos propios (`ErrNotFound`, `ErrInvalidInput`), mapeados a códigos HTTP solo en la capa `delivery`.
- Nombres de paquetes **cortos, en minúscula, en singular y sin guiones bajos** (`habit`, `ledger`, no `habit_utils`).
- Evitar nombres redundantes con el paquete (`habit.Service`, no `habit.HabitService`).
- Todo struct, interfaz y función exportada lleva comentario Godoc que empieza con su nombre.
- Interfaces pequeñas y definidas **donde se consumen**, no donde se implementan.
- Pasar `context.Context` como primer parámetro en operaciones de I/O, DB y llamadas externas.
- Sin variables globales mutables ni `init()` con efectos secundarios.
- Logging estructurado (`log/slog`); nunca registrar secretos ni datos financieros sensibles.
- Tests table-driven, ubicados junto al código (`*_test.go`).

### Svelte / JavaScript

- Componentes **pequeños y modulares**, con una sola responsabilidad.
- Componentes en `PascalCase` (`HabitCard.svelte`); variables, funciones y stores en `camelCase`; constantes en `UPPER_SNAKE_CASE`.
- Nombres descriptivos: `isLoading`, `monthlyBalance`, `handleSubmit`; evitar abreviaturas crípticas.
- Estado local en el componente; estado compartido en stores (`src/lib/stores`). Evitar prop-drilling profundo.
- Toda llamada al backend pasa por `src/lib/api/`, nunca `fetch` directo en componentes.
- Estilos con clases utilitarias de Tailwind; extraer a componente (no a CSS ad-hoc) cuando se repitan patrones.
- Manejar siempre estados de carga, error y vacío en la UI.

### SQL / PostgreSQL

- Tablas en **plural y snake_case**: `users`, `habit_logs`, `transactions`.
- Columnas en `snake_case`; PK `id` (UUID o BIGINT identity); FK como `<singular>_id` (`user_id`).
- Toda tabla incluye `created_at` y `updated_at` (`TIMESTAMPTZ`); usar `deleted_at` si aplica soft delete.
- Montos monetarios con `NUMERIC(14,2)` (**nunca** `FLOAT`); guardar la moneda explícitamente.
- Definir constraints (`NOT NULL`, `UNIQUE`, `CHECK`, `FOREIGN KEY`) e índices para consultas frecuentes.
- **Migraciones**: un cambio lógico por migración, con `up` y `down`, nombradas `NNNNNN_descripcion.up.sql` / `.down.sql`. **Nunca editar una migración ya aplicada**; crear una nueva.
- Consultas siempre parametrizadas (`$1, $2`); jamás concatenar strings en SQL.

---

## 4. Principios de Desarrollo y Workflow

- **DRY**: extraer lógica repetida, pero sin abstraer prematuramente (regla de tres).
- **KISS**: preferir la solución más simple y legible; evitar sobreingeniería.
- **SOLID**: responsabilidad única por paquete/struct, dependencias por interfaz, composición sobre herencia.
- Cambios pequeños y enfocados; no mezclar refactors con features en el mismo commit.
- Commits con formato Conventional Commits (`feat:`, `fix:`, `refactor:`, `chore:`, `test:`, `docs:`).
- Todo código nuevo con lógica de negocio debe incluir tests. No se mergea con tests rotos ni con lint fallando.
- Antes de dar una tarea por terminada: `gofmt`, `go vet`, `go test ./...` y, si toca frontend, lint + tests del frontend.

### Variables de entorno

- La configuración se carga **exclusivamente** desde variables de entorno, centralizada en `internal/config`.
- En local se usa un archivo `.env` (cargado en desarrollo); en AWS se inyectan desde el servicio de secretos.
- `.env` **debe estar en `.gitignore`**. Solo se versiona `.env.example` con claves y valores ficticios.
- Al agregar una nueva variable: actualizar `.env.example`, `internal/config` (con validación al arrancar) y esta documentación.
- La app debe **fallar al iniciar** si falta una variable requerida.

Variables esperadas (ver `.env.example`):

```
APP_ENV=development
APP_PORT=8080
TRUSTED_PROXY_HOPS=0             # Proxies de confianza delante de la API (1 detrás de un ALB)
CORS_ALLOWED_ORIGINS=http://localhost:5173  # Orígenes del frontend, separados por comas (https fuera de local)
POSTGRES_USER=user               # Solo para docker-compose (PostgreSQL local)
POSTGRES_PASSWORD=password
POSTGRES_DB=bytebalance
POSTGRES_PORT=5432
DATABASE_URL=postgres://user:password@localhost:5432/bytebalance?sslmode=disable
JWT_SECRET=                      # Obligatoria, mín. 32 bytes (openssl rand -base64 48)
GEMINI_API_KEY=
TELEGRAM_BOT_TOKEN=
TELEGRAM_CHAT_ID=
```

Frontend (`frontend/.env`, ver `frontend/.env.example`). Toda variable `VITE_*` se incrusta en el bundle público del navegador: **nunca** contiene secretos.

```
VITE_API_URL=http://localhost:8080/api/v1   # URL base de la API REST
```

---

## 5. Comandos Frecuentes

### Backend (Go)

```bash
go run ./cmd/api                 # Ejecutar el servidor
air                              # Ejecutar con hot reload (si está instalado)
go build -o bin/api ./cmd/api    # Compilar binario
gofmt -l -w .                    # Formatear código
go vet ./...                     # Análisis estático
golangci-lint run                # Linter
```

### Tests

```bash
go test ./...                    # Todos los tests
go test -race -cover ./...       # Con detector de race conditions y cobertura
go test ./internal/usecase/...   # Un paquete específico
cd frontend && npm run test      # Tests del frontend
```

### Frontend (Svelte)

```bash
cd frontend
npm install                      # Instalar dependencias
npm run dev                      # Servidor de desarrollo
npm run build                    # Build de producción
npm run lint                     # Lint
npm run check                    # Verificación de tipos / svelte-check
```

### Migraciones (golang-migrate)

```bash
migrate create -ext sql -dir migrations -seq nombre_migracion   # Crear migración
migrate -path migrations -database "$DATABASE_URL" up           # Aplicar todas
migrate -path migrations -database "$DATABASE_URL" down 1       # Revertir la última
migrate -path migrations -database "$DATABASE_URL" version      # Ver versión actual
```

---

## 6. Seguridad y Manejo de Secretos

### Reglas generales

- **Nunca** commitear credenciales, tokens ni llaves API (ni en código, ni en tests, ni en logs, ni en comentarios, ni en historial de git).
- **Nunca** hardcodear secretos; siempre leerlos desde variables de entorno vía `internal/config`.
- Si Claude detecta un secreto expuesto, debe detenerse, avisar y recomendar **rotarlo** inmediatamente.
- Antes de cada commit, verificar que no se incluyan `.env` ni archivos con credenciales.

### Gemini API (`GEMINI_API_KEY`)

- Se usa **solo desde el backend**; jamás exponerla al frontend ni en bundles del cliente.
- Toda llamada a Gemini se encapsula en `internal/integration/gemini` detrás de una interfaz (facilita mocks en tests).
- Aplicar timeouts con `context`, reintentos con backoff y límites de uso/rate limiting.
- **Minimización de datos**: enviar a Gemini únicamente lo necesario, anonimizando o agregando datos financieros personales (sin nombres, correos, ni identificadores directos).
- Validar y sanear la respuesta de la IA antes de persistirla o mostrarla al usuario.

### Telegram Bot (`TELEGRAM_BOT_TOKEN`)

- El token vive solo en el backend; nunca se imprime en logs ni en mensajes de error.
- Encapsular en `internal/integration/telegram` detrás de una interfaz.
- Validar el origen de los webhooks (secret token) y verificar que el `chat_id` pertenezca a un usuario vinculado.
- No enviar por Telegram datos sensibles innecesarios (números de cuenta completos, etc.); preferir resúmenes.
- Si el token se filtra: revocarlo con BotFather y generar uno nuevo.

### Infraestructura (AWS)

- En producción, los secretos se gestionan con **AWS Secrets Manager** o **SSM Parameter Store** (nunca en imágenes Docker ni en el repo).
- Aplicar el principio de mínimo privilegio en roles IAM.
- Conexiones a PostgreSQL con TLS (`sslmode=require` o superior) en entornos no locales.
- Rotación periódica de credenciales y llaves API.

### Seguridad de aplicación

- Validar y sanear toda entrada de usuario en la capa `delivery`.
- Consultas SQL siempre parametrizadas.
- Configurar CORS de forma restrictiva, y usar HTTPS en producción.
- No registrar datos financieros ni PII en logs.

---

## 7. Instrucciones para Claude Code

- Respetar la estructura y las reglas de dependencia de la sección 2; ante la duda, preguntar antes de crear nuevos paquetes de nivel superior.
- Preferir cambios incrementales y verificables: implementar, compilar, ejecutar tests y luego continuar.
- No instalar dependencias nuevas sin justificarlas; priorizar la librería estándar de Go.
- No modificar migraciones ya existentes ni archivos `.env`.
- Documentar decisiones arquitectónicas relevantes en `docs/` (ADRs breves) cuando se tomen.