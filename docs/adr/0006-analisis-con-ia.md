# ADR 0006 — Análisis inteligente con Gemini

- **Estado:** aceptado
- **Fecha:** 2026-10-16

## Contexto

Los usuarios deben poder pedir un diagnóstico personalizado que analice juntas sus
finanzas y la constancia de sus hábitos, con sugerencias concretas. El análisis lo
genera Google Gemini, un servicio externo con costo y cuota, que recibe datos personales.

## Decisiones

1. **Configuración opcional.** `GEMINI_API_KEY` vacía desactiva el análisis: `GET
   /api/v1/ai` responde `{"enabled": false}`, `POST /api/v1/ai/diagnose` responde 503
   `ai_disabled` y el panel lo indica. Se validan el formato de la llave (sin mostrarla
   en el error) y el de `GEMINI_MODEL`, que forma parte de la ruta de la URL.
2. **Modelo por defecto: `gemini-3.5-flash`** con `thinkingLevel: low`. En pruebas
   respondió en 2–4 s. `gemini-3.8-flash` tardó entre 57 y 79 s, demasiado para una
   petición interactiva, y `gemini-2.5-flash` ya no está disponible para llaves
   nuevas. `GEMINI_MODEL` permite cambiarlo sin tocar código; solo admite modelos 3.x,
   que son los que aceptan `thinkingLevel`.
3. **Minimización de datos.** `internal/domain/insight` arma un `Snapshot` agregado de
   los últimos 30 días (más los 30 anteriores para comparar):
   - Finanzas: totales por moneda y las 5 categorías con más gasto.
   - Hábitos activos: periodos cumplidos, racha y progreso.

   **No se envían** nombre, email, IDs ni descripciones de transacciones o notas. Los
   nombres de categorías y hábitos sí se envían, recortados a 40 caracteres, porque sin
   ellos las sugerencias no serían útiles. El panel lo explica al usuario.
4. **Precisión.** Los montos se envían ya formateados ("95 000"). La tasa de ahorro y la
   variación de gastos se calculan en Go. El prompt explica cada campo y pide copiar las
   cifras tal cual, con `temperature` 0,2. Sin estas medidas, el modelo transcribía mal
   montos (95 000 → 950 000) y confundía un hábito nuevo con uno abandonado.
5. **Salida estructurada y saneada.** Se pide JSON con `responseSchema` (titular,
   resumen, observaciones de finanzas y de hábitos, sugerencias). La respuesta pasa por
   `Diagnosis.Sanitized`:
   - Convierte todo a texto plano: quita caracteres de control y marcas Markdown.
   - Recorta longitudes y cantidades.
   - Si no queda contenido, la respuesta se trata como fallo.

   El frontend lo muestra como texto, nunca como HTML. El diagnóstico no se guarda en
   la base: se conserva en memoria durante la sesión del navegador.
6. **Resiliencia.**
   - Cada intento tiene un timeout de 15 s.
   - Hay hasta 3 intentos con espera exponencial (1 s, 2 s) ante 429, 5xx o errores de
     red. En pruebas, Gemini devolvió 503 "high demand" con frecuencia.
   - Con todos los intentos, el peor caso queda en ~48 s, por debajo del idle timeout de
     60 s de un ALB. Solo esta ruta amplía su write deadline, a 60 s, con
     `http.ResponseController`.
   - Un fallo se responde 503 `ai_unavailable`, sin detalles. El detalle de Google
     (código, cuota) va al log y nunca incluye datos del usuario.
   - Sin datos que analizar se responde 422 `not_enough_data`, sin llamar a la IA.
7. **Límites de uso.** Por usuario se permiten 3 análisis seguidos y luego 1 cada 20
   minutos. En todo el servidor, 10 seguidos y luego 1 cada 6 s (~10/min), para no
   agotar la cuota de la llave. Los límites viven en memoria, igual que los del login
   (ADR 0002).
8. **La llave no se filtra.** Viaja en la cabecera `x-goog-api-key`, nunca en la URL, así
   que los errores de transporte no la contienen. Solo existe en el backend.
9. `GEMINI_API_URL` (opcional) permite usar un servidor simulado; http solo se admite
   hacia localhost en development.

## Consecuencias

- La IA puede equivocarse. El panel lo advierte ("puede contener errores, no es asesoría
  financiera profesional") y el prompt prohíbe recomendar productos financieros o dar
  consejos médicos.
- Con varias réplicas, los límites se multiplican por el número de instancias; si se
  escala, conviene moverlos a un almacén compartido.
- Si Google retira el modelo por defecto, basta con cambiar `GEMINI_MODEL`.
