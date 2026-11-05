# IACHAT con Gemini

IACHAT usa una IA real de Google Gemini. No contiene respuestas simuladas. El modelo predeterminado es `gemini-3.5-flash-lite`; se puede cambiar con `GEMINI_MODEL`.

## Activarlo en Render

1. Abre [Google AI Studio: claves de API](https://aistudio.google.com/apikey), crea o elige un proyecto y genera una clave para la Gemini API. Esta clave es distinta del ID y del secreto de OAuth usados para iniciar sesión con Google.
2. En Render, abre el servicio del backend, entra a **Environment** y agrega `GEMINI_API_KEY` con la clave. Pega solo el valor, sin comillas ni el nombre de la variable.
3. Deja `GEMINI_MODEL=gemini-3.5-flash-lite` o escribe un [modelo compatible](https://ai.google.dev/gemini-api/docs/models).
4. Guarda y despliega nuevamente el backend. Vuelve a abrir el chat para consultar la disponibilidad de la IA.

La clave permanece en Render. No la pongas en Netlify, en variables `NUXT_PUBLIC_*`, en el repositorio ni en mensajes del chat. `render.yaml` no solicita una clave obligatoria al crear el Blueprint: sin ella el backend y las conversaciones entre personas siguen funcionando.

Para desarrollo local, agrega las mismas dos variables al `.env` de la raíz y reinicia el backend. Los archivos `.env.example` y `.env.production.example` muestran únicamente los nombres y valores de ejemplo.

Gemini ofrece un nivel gratuito con cuotas para este modelo. La disponibilidad y los límites dependen del proyecto, la región y el plan de Google; consulta [precios](https://ai.google.dev/gemini-api/docs/pricing#gemini-3.5-flash-lite) y [límites](https://ai.google.dev/gemini-api/docs/rate-limits) antes de activar facturación. La integración no configura facturación ni hace llamadas durante la instalación o las pruebas.

## Qué información recibe la IA

El navegador envía al backend el historial reciente de la conversación con IACHAT, y el backend lo entrega a Gemini para producir la siguiente respuesta. No se consultan ni se envían conversaciones con otras personas, correos, cookies de sesión ni credenciales de OAuth.

El historial de IACHAT permanece en la memoria de la sesión del frontend; todavía no se guarda en PostgreSQL. Se limpia al cerrar sesión y al recargar la aplicación. El backend no almacena las conversaciones con IA. Google procesa los mensajes según sus [términos](https://ai.google.dev/gemini-api/terms); su nivel gratuito puede usar contenido para mejorar productos.

## API y límites

Ambas rutas requieren una sesión activa, de Google o de invitado, y responden con `Cache-Control: private, no-store`:

- `GET /api/assistant`: devuelve `{ "enabled": true, "provider": "gemini", "model": "gemini-3.5-flash-lite" }`. `enabled` indica que existe una clave configurada; no garantiza que la clave, la cuota o el proveedor estén disponibles.
- `POST /api/assistant/messages`: recibe `{ "messages": [{ "role": "user", "content": "Hola" }] }` y devuelve `{ "content": "Respuesta de Gemini" }`.

El historial debe alternar `user` y `assistant`, comenzar y terminar con `user`, y contener como máximo 21 mensajes y 16000 caracteres en total. Cada mensaje de usuario admite 4000 caracteres; cada respuesta del asistente, hasta 8000. El cuerpo JSON se limita a 128 KiB.

La generación usa [GenerateContent de Gemini](https://ai.google.dev/api/generate-content) con un límite de 1024 tokens de salida, una sola respuesta candidata y un plazo de 22 segundos. El cliente cancela la solicitud al cancelar la petición original, limita la respuesta del proveedor a 1 MiB, no sigue redirecciones y envía la clave únicamente en la cabecera `x-goog-api-key`.

Hay una solicitud simultánea por usuario y cuatro en total. Los límites son 6 solicitudes por minuto y 60 por día por usuario, y 30 por minuto y 500 por día en todo el backend. Cuentan los intentos admitidos aunque el proveedor falle. El estado se mantiene en memoria, tiene un máximo de 4096 usuarios y elimina entradas inactivas después de 24 horas; reiniciar el backend reinicia estas cuotas. Estos límites se aplican a la instancia actual y no sustituyen las cuotas de Google ni un control externo de gasto.

Si falta la clave, el envío devuelve HTTP 503 con una explicación. Una cuota agotada devuelve HTTP 429; una espera excesiva, HTTP 504. Los errores del proveedor se traducen a mensajes en español y no devuelven claves, cuerpos de error de Google ni razonamiento interno del modelo. No se generan respuestas simuladas cuando Gemini falla.

## Verificación sin llamadas a Google

Desde `backend`, ejecuta:

```sh
go test ./internal/assistant ./internal/config
```

Las pruebas usan un servidor HTTP local que simula el protocolo de Gemini. Verifican la autenticación, el historial, el envío seguro de la clave, las respuestas, los errores, las cuotas y la cancelación sin consumir tokens ni tocar la base de datos.
