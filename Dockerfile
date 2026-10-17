# syntax=docker/dockerfile:1
#
# Imagen de producción del backend de ByteBalance (ver DEPLOYMENT.md).
#
#   docker build -t bytebalance-api .                      # API (stage por defecto)
#   docker build -t bytebalance-migrate --target migrate . # Migraciones de la base
#
# La imagen no contiene secretos: toda la configuración llega por variables de
# entorno al ejecutarla (en AWS, desde Secrets Manager o SSM Parameter Store).

ARG GO_VERSION=1.27

# --- Compilación ---------------------------------------------------------------
FROM golang:${GO_VERSION}-alpine AS build
WORKDIR /src

# Las dependencias van en su propia capa: solo se descargan si cambia go.mod/go.sum.
COPY go.mod go.sum ./
RUN go mod download && go mod verify

COPY cmd ./cmd
COPY internal ./internal
COPY pkg ./pkg

# Binario estático (sin cgo), sin rutas locales ni símbolos de depuración. La base
# de zonas horarias va incrustada en el binario (import time/tzdata en main.go).
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/api ./cmd/api

# --- Migraciones -----------------------------------------------------------------
# Tarea de un solo uso: lee DATABASE_URL del entorno para que la contraseña no
# quede en la línea de comandos. Por defecto aplica todas ("up"); se le puede
# pasar otro comando, p. ej. "version" o "down 1".
FROM migrate/migrate:v4.19.1 AS migrate
COPY migrations /migrations
USER 65532:65532
ENTRYPOINT ["/bin/sh", "-c", "exec migrate -path /migrations -database \"$DATABASE_URL\" \"$@\"", "--"]
CMD ["up"]

# --- API (stage por defecto) ------------------------------------------------------
# distroless/static: sin shell ni gestor de paquetes, con certificados CA para
# las llamadas TLS a Gemini, Telegram y RDS. Corre como usuario sin privilegios.
FROM gcr.io/distroless/static-debian12:nonroot AS api
COPY --from=build /out/api /api
ENV APP_ENV=production \
    APP_PORT=8080
EXPOSE 8080
USER nonroot:nonroot
ENTRYPOINT ["/api"]
