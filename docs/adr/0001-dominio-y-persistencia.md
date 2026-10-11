# ADR 0001 — Modelo de dominio y persistencia en PostgreSQL

- **Estado:** aceptado
- **Fecha:** 2026-10-11

## Contexto

La capa de dominio (`internal/domain`) solo puede depender de la stdlib, y la
persistencia se hace en PostgreSQL con el esquema de `migrations/000001_init_schema`.

## Decisiones

1. **Identificadores:** `domain.ID` es un tipo definido sobre `uuid.UUID`
   (paquete `uuid` de la stdlib, Go 1.27). Los genera la base de datos
   (`gen_random_uuid()`) y los repositorios los devuelven tras `Create`.
   Por esto el módulo exige `go 1.27`.
2. **Montos:** `finance.Money` guarda centavos en `int64`. La conversión a
   `NUMERIC(14,2)` se hace en SQL (`$n::numeric / 100` y `(amount * 100)::bigint`),
   así los montos nunca pasan por `float`.
3. **Interfaces de repositorio en el dominio:** cada módulo (`user`, `finance`,
   `habit`) declara sus interfaces en `repository.go`. Las implementaciones viven
   en `internal/repository/postgres` y usan `pgx/v5`.
4. **Aislamiento por usuario:** las operaciones sobre entidades de un usuario
   reciben su `userID` y lo filtran en SQL; si la entidad es de otro usuario se
   responde `domain.ErrNotFound`, sin revelar que existe.
5. **Errores:** los repositorios traducen errores de PostgreSQL a errores de
   dominio: sin filas → `ErrNotFound`, `23505` → `ErrConflict`,
   `23503` → `ErrNotFound`, `23514`/`22001` → `ErrInvalidInput`.
6. **Transacciones:** los repositorios reciben la interfaz `postgres.DBTX`, que
   cumplen `*pgxpool.Pool`, `*pgx.Conn` y `pgx.Tx`, para poder componer varios
   repositorios en una misma transacción.

## Consecuencias

- Los tests de integración de `internal/repository/postgres` necesitan
  `DATABASE_URL`; sin ella se omiten. Cada test corre en una transacción que se
  revierte, así que no deja datos.
- `habit.LogRepository` no comprueba la propiedad del hábito: el caso de uso
  debe validarla antes con `habit.Repository.GetByID`.
