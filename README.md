# Realtime Chat

Chat 1:1 en tiempo real estilo WhatsApp. Este repositorio contiene la **estructura inicial** del proyecto: frontend, backend, base de datos y configuración local listos para empezar a implementar la lógica.

## Stack

| Capa | Tecnología |
| --- | --- |
| Frontend | Nuxt 3 + Vue 3 + Pinia + Tailwind CSS |
| Backend | Go + Chi |
| Tiempo real | WebSocket (`coder/websocket`) |
| Base de datos | PostgreSQL 16 + pgx |
| Deploy | Render |

No se usa Next.js ni React. La autenticación se basa en **cookies HttpOnly**, nunca en `localStorage`.

## Estructura

```text
realtime-chat/
├── frontend/                     # Nuxt 3
│   ├── assets/css/main.css
│   ├── components/chat/          # Lista, ventana, burbujas, composer
│   ├── composables/
│   │   ├── useApi.ts             # fetch con cookies (credentials: include)
│   │   └── useChatSocket.ts      # WebSocket + reconexión
│   ├── layouts/default.vue
│   ├── middleware/auth.global.ts
│   ├── pages/
│   │   ├── index.vue
│   │   ├── login.vue
│   │   └── chat/index.vue
│   ├── stores/
│   │   ├── auth.ts
│   │   └── chat.ts
│   ├── types/index.ts
│   ├── nuxt.config.ts
│   └── package.json
├── backend/                      # Go
│   ├── cmd/server/main.go        # entrypoint
│   ├── internal/
│   │   ├── auth/                 # JWT, cookies HttpOnly, bcrypt
│   │   ├── chat/                 # conversaciones y mensajes
│   │   ├── config/               # variables de entorno
│   │   ├── db/                   # pool de PostgreSQL
│   │   ├── httpapi/              # router Chi y CORS
│   │   ├── httputil/             # helpers JSON
│   │   ├── users/                # registro, login y perfiles
│   │   └── websocket/            # hub, clientes, tickets y eventos
│   ├── migrations/               # SQL versionado
│   ├── go.mod
│   └── Dockerfile
├── docker-compose.yml            # PostgreSQL (+ Adminer opcional)
├── .env.example
├── .gitignore
├── pnpm-workspace.yaml
└── README.md
```

## Seguridad

- La sesión viaja en una cookie `HttpOnly`, `SameSite` configurable y `Secure` en producción. **No se guarda ningún token en `localStorage` ni `sessionStorage`.**
- El JWT se firma con HS256 usando `JWT_SECRET`. El backend se niega a arrancar en producción con el secreto por defecto.
- El WebSocket no recibe el JWT. El cliente pide un **ticket de un solo uso** a `POST /api/ws/ticket` y lo usa para abrir `/ws?ticket=...`.
- CORS con credenciales limitado a `CORS_ORIGIN`.
- Contraseñas con bcrypt.
- Límite de tamaño de mensajes de WebSocket y de cuerpos JSON.
- Los mensajes se persisten en PostgreSQL antes de emitirse (cuando se implemente el repositorio).

## Requisitos

- Go 1.24+
- Node.js 20+
- pnpm 9+
- Docker

## Puesta en marcha

1. Copiar variables de entorno:

   ```powershell
   Copy-Item .env.example .env
   Copy-Item frontend/.env.example frontend/.env
   ```

2. Levantar PostgreSQL:

   ```powershell
   docker compose up -d
   ```

3. Backend:

   ```powershell
   cd backend
   go mod tidy
   go run ./cmd/server
   ```

   La API queda en `http://localhost:8080` y el healthcheck en `GET /healthz`.

4. Frontend (desde la raíz):

   ```powershell
   pnpm install
   pnpm --dir frontend dev
   ```

   La app queda en `http://localhost:3000`.

5. Opcional, para inspeccionar la base de datos:

   ```powershell
   docker compose --profile tools up -d adminer
   ```

   Adminer: `http://localhost:8081` (sistema `PostgreSQL`, servidor `postgres`).

## Variables de entorno

### Backend (`.env` en la raíz)

| Variable | Descripción |
| --- | --- |
| `DATABASE_URL` | Cadena de conexión de PostgreSQL |
| `HTTP_ADDR` | Dirección del servidor (`:8080`) |
| `APP_ENV` | `development` o `production` |
| `CORS_ORIGIN` | Origen permitido del frontend |
| `JWT_SECRET` | Secreto de firma del JWT |
| `ACCESS_TOKEN_TTL` | Duración de la sesión (`24h`) |
| `COOKIE_SECURE` | `true` en producción (HTTPS) |
| `COOKIE_SAME_SITE` | `lax`, `strict` o `none` |
| `COOKIE_DOMAIN` | Dominio de la cookie (opcional) |

### Frontend (`frontend/.env`)

| Variable | Descripción |
| --- | --- |
| `NUXT_PUBLIC_API_BASE` | URL del backend (`http://localhost:8080`) |
| `NUXT_PUBLIC_WS_BASE` | URL WebSocket (`ws://localhost:8080`) |

## API REST

| Método | Ruta | Estado |
| --- | --- | --- |
| `GET` | `/healthz` | Listo |
| `POST` | `/api/auth/register` | Estructura (repositorio pendiente) |
| `POST` | `/api/auth/login` | Estructura (repositorio pendiente) |
| `POST` | `/api/auth/logout` | Listo |
| `GET` | `/api/me` | Estructura (repositorio pendiente) |
| `GET` | `/api/users` | Estructura (repositorio pendiente) |
| `GET` | `/api/conversations` | Estructura (repositorio pendiente) |
| `POST` | `/api/conversations` | Estructura (repositorio pendiente) |
| `GET` | `/api/conversations/{id}/messages` | Estructura (repositorio pendiente) |
| `POST` | `/api/ws/ticket` | Listo |
| `GET` | `/ws?ticket=...` | Listo (conexión y presencia) |

## Eventos WebSocket

Cliente → servidor:

- `message.send`: `{ conversationId, clientMessageId, content }`
- `typing`: `{ conversationId, active }`
- `message.read`: `{ conversationId, messageId }`

Servidor → cliente:

- `message.new`: `{ message }`
- `message.ack`: `{ clientMessageId, message }`
- `typing`: `{ conversationId, userId, active }`
- `presence`: `{ userId, online }`
- `message.read`: `{ conversationId, userId, messageId }`
- `error`: `{ error }`

## Estado actual

- Estructura completa y compilable.
- Base de datos con migraciones iniciales.
- Autenticación JWT + cookies y tickets de WebSocket implementados.
- Hub de WebSocket con presencia y enrutamiento por conversación.
- Repositorios de `users` y `chat` son stubs que devuelven `ErrNotImplemented`.
- UI estilo WhatsApp con datos de ejemplo vacíos; falta conectar la lógica real.

## Próximos pasos

1. Implementar `users.Repository` y `chat.Repository` con pgx.
2. Aplicar migraciones al arrancar o con una herramienta (`golang-migrate`).
3. Persistir y emitir mensajes desde `message.send`.
4. Marcar mensajes como leídos y calcular no leídos.
5. Tests de integración del hub y de la API.
6. Redis Pub/Sub si se despliegan varias instancias.

## Deploy en Render

### Backend

- Root directory: `backend`
- Build: `go build -o server ./cmd/server`
- Start: `./server`
- Health check: `/healthz`
- Variables: las mismas de `.env`, con `COOKIE_SECURE=true` y un `JWT_SECRET` aleatorio.

### Frontend

- Root directory: `frontend`
- Build: `pnpm install && pnpm build`
- Start: `node .output/server/index.mjs`
- Variables públicas apuntando a las URLs de producción.

Si frontend y backend están en dominios distintos, usar `COOKIE_SAME_SITE=none` y `COOKIE_SECURE=true`. Para máxima seguridad, servirlos bajo el mismo dominio con un proxy inverso.
