# ADR 0007 — Ahorro separado del disponible para gastar

- **Estado:** aceptado
- **Fecha:** 2026-10-16

## Contexto

El balance total (ingresos − gastos) no dice cuánto se puede gastar sin tocar el
dinero reservado para ahorrar, y eso invita a gastar de más. Los usuarios deben poder
apartar dinero al ahorro, retirarlo cuando lo necesiten y fijar una meta mensual.

## Decisiones

1. **Dos tipos de transacción nuevos** (migración `000003_add_savings`):
   - `saving` aparta dinero del disponible al ahorro.
   - `withdrawal` lo devuelve del ahorro al disponible.

   Se eligieron tipos de transacción, y no una categoría especial ni una tabla aparte,
   para reutilizar el registro, la edición, el borrado, los filtros y el historial que
   ya existen.
2. **Fórmulas** (en `finance.Balance`):
   - Balance total = ingresos − gastos. El ahorro **no** lo cambia: el dinero apartado
     sigue siendo del usuario.
   - Ahorro = apartado − retirado.
   - Disponible real = balance total − ahorro.

   Se cumple siempre balance total = disponible + ahorro. `GET /api/v1/balance` devuelve
   `saved_cents`, `withdrawn_cents`, `savings_cents` y `available_cents` junto a los
   campos anteriores, sin romper a los clientes existentes.
3. **El ahorro nunca queda negativo** (por moneda). El caso de uso rechaza con 422 y un
   mensaje claro:
   - un retiro mayor que el ahorro;
   - una edición que reduzca un aporte por debajo de lo ya retirado, o que convierta un
     aporte en otro tipo;
   - borrar un aporte ya retirado en parte.

   Solo se consulta el ahorro cuando interviene un movimiento de ahorro. Los aportes no
   se limitan por el disponible: si el usuario no registró sus ingresos, el panel
   muestra el disponible en negativo y lo explica, en lugar de bloquearlo.
4. **El ahorro no lleva categoría.** Las categorías siguen siendo solo de ingresos o
   gastos. El dominio lo valida y la base lo refuerza con
   `transactions_saving_without_category`. Al convertir un gasto en ahorro, su
   categoría se quita.
5. **Meta mensual fija** en `users.monthly_savings_goal` (`NUMERIC(14,2)`, nulo = sin
   meta), en la moneda principal del usuario. Se fija o se quita con `PATCH
   /api/v1/users/me` (`monthly_savings_goal_cents`; `null` la quita). El progreso del
   mes es (apartado − retirado en el mes) / meta.
6. **UI.**
   - El resumen muestra una cuarta tarjeta, "Ahorro del mes", con la barra de progreso
     hacia la meta y el ahorro acumulado.
   - Debajo, "Disponible para gastar" desglosa balance total − ahorro = disponible.
   - En la lista, cada movimiento muestra su efecto en el disponible: un aporte resta y
     un retiro suma.
7. **Análisis con IA.** El resumen para Gemini incluye lo apartado en el periodo y la
   meta, y el prompt distingue el excedente (ingresos − gastos) del ahorro explícito.

## Consecuencias

- Revertir la migración borra los movimientos de ahorro, porque el esquema anterior no
  los admite.
- La meta es un monto en la moneda principal. Si el usuario cambia de moneda, la meta
  conserva el número y conviene revisarla.
- La comprobación del ahorro lee todas las transacciones del usuario. A la escala de una
  app personal es suficiente; si creciera, se calcularía con un `SUM` en SQL.
