# ADR 0005 — Integración con Telegram

- **Estado:** aceptado
- **Fecha:** 2026-10-16

## Contexto

Los usuarios deben poder recibir recordatorios de hábitos en Telegram. Cada usuario
tiene su propio chat (`users.telegram_chat_id`), así que hace falta un flujo para
vincular la cuenta con el chat de forma segura, recibir los mensajes del bot y enviar
recordatorios a la hora local de cada usuario.

## Decisiones

1. **Configuración opcional.** `TELEGRAM_BOT_TOKEN` vacía desactiva Telegram: los
   endpoints responden 503 `telegram_disabled` y el frontend lo muestra. Si se define,
   debe tener el formato de BotFather y la API valida el bot con `getMe` al arrancar
   (falla rápido si el token no sirve). `TELEGRAM_CHAT_ID` se elimina: no hay chat
   global.
2. **Vinculación con código de un solo uso.**
   - `POST /api/v1/telegram/link` genera un código de 192 bits aleatorios, vigente
     10 minutos, y devuelve `https://t.me/<bot>?start=<código>`.
   - Solo se guarda su **SHA-256** (`telegram_link_codes`); un hash rápido basta porque
     el código es aleatorio y de vida corta. Generar uno nuevo invalida el anterior.
   - El bot recibe `/start <código>`, lo consume con un `UPDATE` condicional (atómico:
     un código solo sirve una vez) y guarda el chat. Solo se aceptan **chats privados**.
   - Un chat ya vinculado a otra cuenta no se reasigna (índice único). `/stop` desde el
     chat o `DELETE /api/v1/telegram/link` desde la app desvinculan.
   - Generar enlaces está limitado a 5 seguidos y 1 cada 30 s por usuario.
3. **Recepción por long polling** (`getUpdates`) en `internal/delivery/telegram`. No
   necesita URL pública, por lo que funciona igual en local y en una instancia de AWS.
4. **Recordatorios** (`internal/scheduler`). Un job procesa cada minuto exactamente una
   vez. La consulta compara `reminder_time` con la hora local de cada usuario en SQL
   (`AT TIME ZONE`), y el caso de uso omite los hábitos que ya cumplieron la meta del
   periodo (día, semana ISO o mes). El mensaje solo lleva el nombre del hábito y el
   progreso: ningún dato financiero.
5. **El token no se filtra.** La URL de la Bot API contiene el token; el cliente quita
   la URL de todo error de transporte y nunca se registra. Los mensajes se envían como
   texto plano (sin `parse_mode`) para que ningún contenido se interprete como formato.
6. `TELEGRAM_API_URL` (opcional) permite usar un servidor propio de la Bot API o uno
   simulado en pruebas; http solo se admite hacia localhost en `development`.

## Consecuencias

- **Una sola instancia.** Con varias réplicas, Telegram responde 409 si dos procesos
  hacen long polling, y cada réplica enviaría los mismos recordatorios. Antes de escalar
  horizontalmente: pasar a **webhook** (`setWebhook` con `secret_token`, validando la
  cabecera `X-Telegram-Bot-Api-Secret-Token`) y elegir un líder para el job (p. ej. un
  advisory lock de PostgreSQL) o moverlo a EventBridge.
- Los minutos en que el proceso estuvo detenido no se recuperan: un recordatorio de esa
  hora no se envía. Es aceptable para un recordatorio diario.
- Si el proceso se reinicia dentro del mismo minuto, un recordatorio podría enviarse dos
  veces. Para evitarlo del todo habría que registrar los envíos en la base.
