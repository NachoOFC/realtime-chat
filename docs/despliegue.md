# Desplegar frontend en Netlify y backend en Render

El mismo repositorio contiene las dos partes. Netlify compila `frontend`; Render compila `backend`. Neon conserva la base de datos existente. Los archivos `netlify.toml` y `render.yaml` definen la configuración de construcción de cada servicio.

## Cómo se conectan

```text
Navegador -> https://tu-chat.netlify.app/api/... -> proxy Nuxt -> https://tu-chat-api.onrender.com/api/...
Navegador -> wss://tu-chat-api.onrender.com/ws (ticket de un solo uso)
Backend -> Neon
```

Las peticiones HTTP de autenticación pasan por el dominio del frontend. Por eso las cookies de sesión y de OAuth pertenecen al sitio que visita el usuario y no dependen de cookies de terceros entre `netlify.app` y `onrender.com`.

El acceso de Google comienza y termina en Netlify mediante `/api/auth/google` y `/api/auth/google/callback`; el proxy entrega ambas peticiones al backend. Los secretos de Google y de la base de datos permanecen en Render.

## 1. Crear el sitio en Netlify y obtener su URL

1. Sube los cambios del repositorio a GitHub.
2. En Netlify, elige importar un proyecto existente y conecta este repositorio.
3. Mantén la **Base directory** en la raíz del repo y selecciona **Package directory: `frontend`**. El archivo `netlify.toml` ya indica:
   - Build command: `pnpm --dir frontend build`.
   - Publish directory: `frontend/dist`.
   - Node 24 y preset de Nitro para Netlify.
4. Elige un nombre estable para el sitio y guarda su URL, por ejemplo `https://tu-chat.netlify.app`.

Netlify puede lanzar el primer despliegue al crear el sitio. La pantalla de acceso todavía no podrá conectarse al backend hasta completar el paso 3; ese primer despliegue sirve para obtener la URL del frontend.

## 2. Crear el backend en Render

En Render, crea un **Blueprint** conectado al mismo repositorio. Selecciona el archivo `render.yaml`. Creará un único Web Service con:

| Campo | Valor |
| --- | --- |
| Runtime | Go |
| Root directory | `backend` |
| Build command | `go build -trimpath -ldflags="-s -w" -o server ./cmd/server` |
| Start command | `./server` |
| Health check | `/healthz` |

También puedes crear un Web Service manual con esos valores. El servicio utiliza el puerto que Render proporciona en `PORT`; no es necesario copiar `HTTP_ADDR=:8080` del entorno local. Render ejecuta y supervisa el binario directamente.

Completa estas variables cuando Render las solicite:

| Variable | Qué poner |
| --- | --- |
| `DATABASE_URL` | La conexión a Neon, conservando sus parámetros TLS. |
| `CORS_ORIGIN` | La URL del frontend, por ejemplo `https://tu-chat.netlify.app`, sin `/` final. |
| `FRONTEND_URL` | La misma URL del frontend. |
| `GOOGLE_CLIENT_ID` | El Client ID de tu cliente OAuth. |
| `GOOGLE_CLIENT_SECRET` | El secreto del cliente OAuth. |
| `GOOGLE_REDIRECT_URL` | `https://tu-chat.netlify.app/api/auth/google/callback`. |

El Blueprint genera un `JWT_SECRET` nuevo y configura `APP_ENV=production`, `COOKIE_SECURE=true`, `COOKIE_SAME_SITE=lax` y `COOKIE_DOMAIN` vacío. `TLS_CERT_FILE` y `TLS_KEY_FILE` también quedan vacíos: Render gestiona el certificado público HTTPS. Los certificados locales de `.certs/` se usan solo en tu PC.

Si creas el servicio manualmente, usa [.env.production.example](../.env.production.example) como referencia de las variables. Los valores de ejemplo deben reemplazarse en el panel del hosting.

Guarda la URL pública que Render le asigne al backend, por ejemplo `https://tu-chat-api.onrender.com`. Confirma que `https://tu-chat-api.onrender.com/healthz` devuelve `{"status":"ok"}`.

La base de datos no se crea ni se migra durante este despliegue. Si estás usando la misma Neon que probaste localmente, conserva sus tablas y datos; las migraciones ya aplicadas siguen aplicadas.

## 3. Completar las variables del frontend en Netlify

En las variables de entorno del sitio, agrega los valores siguientes y habilita su alcance para **Builds y Functions**:

```env
NUXT_API_PROXY_TARGET=https://tu-chat-api.onrender.com
NUXT_PUBLIC_WS_BASE=wss://tu-chat-api.onrender.com
NUXT_PUBLIC_API_BASE=
NUXT_DEV_TLS=false
```

`NUXT_API_PROXY_TARGET` es una opción privada del servidor Nuxt; no se incorpora al código del navegador. `NUXT_PUBLIC_API_BASE` queda vacío para que el navegador use `/api` en el mismo origen de Netlify. El WebSocket usa la URL pública de Render y se autentica con el ticket obtenido por el proxy.

Guarda las variables y vuelve a desplegar Netlify para aplicar la configuración. [frontend/.env.production.example](../frontend/.env.production.example) contiene los mismos campos como referencia.

## 4. Registrar el callback público en Google

En Google Cloud, abre **Google Auth Platform -> Clientes -> tu aplicación web** y agrega otra URI de redirección autorizada:

```text
https://tu-chat.netlify.app/api/auth/google/callback
```

Mantén también la URI local si vas a seguir desarrollando en tu PC:

```text
https://localhost:8080/api/auth/google/callback
```

El callback de producción va al dominio de Netlify. Debe coincidir exactamente con `GOOGLE_REDIRECT_URL` en Render. Si cambias el nombre del sitio de Netlify o agregas un dominio propio, actualiza esa URI, `CORS_ORIGIN` y `FRONTEND_URL` en conjunto.

Si el proyecto OAuth continúa en modo de pruebas, agrega las cuentas que lo van a probar en **Público -> Usuarios de prueba**.

## 5. Probar el despliegue

1. Abre la URL de Netlify y entra con Google.
2. Recarga para comprobar que se recupera la sesión.
3. Abre otra ventana de incógnito y entra como invitado.
4. Inicia un chat entre ambos, envía mensajes y comprueba el historial.
5. Sal como invitado. Su perfil y sus mensajes deben desaparecer; la cuenta Google conserva los mensajes que ella envió.

IACHAT aparece siempre en la lista. Para habilitar sus respuestas, agrega `GEMINI_API_KEY` en Render y vuelve a desplegar el backend. La clave se crea en [Google AI Studio](https://aistudio.google.com/apikey); los pasos y el alcance del historial están en [la guía de IACHAT](asistente-ia.md).

El Blueprint elige el plan Free de Render para las pruebas. Ese plan se suspende después de 15 minutos sin tráfico y puede tardar alrededor de un minuto en volver a responder. El primer acceso puede necesitar una espera y recarga. Un chat que deba estar disponible continuamente necesita una instancia que permanezca activa. [Límites del plan Free](https://render.com/docs/free)

El hub de WebSockets actual vive en la memoria de una sola instancia; el Blueprint mantiene una instancia. Para escalar a varias habría que compartir los eventos, por ejemplo con Redis.

## Archivos y secretos

Los archivos `.env` locales y `.certs/` están excluidos de Git. Introduce `DATABASE_URL`, `GOOGLE_CLIENT_SECRET` y el resto de secretos en el panel del servicio correspondiente. No subas esos archivos ni copies secretos al frontend.

Referencias: [monorepos en Netlify](https://docs.netlify.com/build/configure-builds/monorepos/), [Blueprints de Render](https://render.com/docs/blueprint-spec), [puerto de Render](https://render.com/docs/environment-variables) y [WebSockets en Render](https://render.com/docs/websocket).
