# Scripts de desarrollo

Se ejecutan desde la raíz del repositorio.

## `cleanup-e2e`: borrar usuarios de pruebas end-to-end

Las pruebas end-to-end crean usuarios con emails `e2e+<marca>@bytebalance.test`. Este
script los borra de la base de datos **local** junto con todos sus datos (categorías,
transacciones, hábitos y registros, por `ON DELETE CASCADE`).

```bash
go run ./scripts/cleanup-e2e        # simulación: muestra qué borraría, no borra nada
go run ./scripts/cleanup-e2e -yes   # borra
```

Ejemplo de salida:

```
Simulación: se borrarían 11 usuarios, 6 categorías, 12 transacciones, 9 hábitos, 8 registros de hábitos.
Vuelve a ejecutar con -yes para borrarlos.
```

Protecciones:

- Solo corre con `APP_ENV=development` y si `DATABASE_URL` apunta a `localhost`,
  `127.0.0.1` o `::1`. En cualquier otro caso se niega sin conectarse.
- Solo toca emails que coinciden con `e2e+%@bytebalance.test`. El dominio `.test` está
  reservado (RFC 2606), así que no puede pertenecer a un usuario real.
- Sin `-yes` solo cuenta. Con `-yes` borra dentro de una transacción y la revierte si el
  número de usuarios borrados no coincide con el contado.

Lee la configuración igual que la API (variables de entorno o `.env`), por lo que necesita
las mismas variables obligatorias, y PostgreSQL debe estar levantado (`docker compose up -d`).
