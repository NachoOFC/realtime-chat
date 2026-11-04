# Realtime Chat

Chat 1:1 en tiempo real estilo WhatsApp. Incluye acceso con Google o apodo temporal, conversaciones privadas, persistencia en PostgreSQL, WebSockets y soporte HTTPS/WSS.

## Stack

| Capa | Tecnología |
| --- | --- |
| Frontend | Nuxt 3 + Vue 3 + Pinia + Tailwind CSS |
| Backend | Go + Chi |
| Tiempo real | WebSocket (`coder/websocket`) |
| Base de datos | PostgreSQL 16 + pgx |
| Deploy | Render |

No se usa Next.js ni React. Los usuarios pueden entrar con **Google** o con un **apodo de invitado**. La sesión usa una cookie HttpOnly; el acceso de invitado es temporal y sus datos se eliminan al salir.

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
- Los mensajes se guardan en PostgreSQL antes de emitirse por WebSocket. HTTPS/WSS cifran la conexión; el contenido no tiene cifrado de extremo a extremo.

## Requisitos

- Go 1.24+
- Node.js 20+
- pnpm 9+
- PostgreSQL local o una base remota como Neon. Docker es opcional.

## Puesta en marcha

1. Copiar variables de entorno:

   ```powershell
   Copy-Item .env.example .env
   Copy-Item frontend/.env.example frontend/.env
   ```

2. Levantar PostgreSQL y aplicar la migración.

   Opción Docker:

   ```powershell
   docker compose up -d
   Get-ChildItem backend/migrations/*.up.sql | Sort-Object Name | ForEach-Object {
     Get-Content $_ | docker compose exec -T postgres psql -U chat -d chat
   }
   ```

   Opción sin Docker: PostgreSQL local o Neon (ajusta `DATABASE_URL` en `.env`). Para usar los comandos siguientes, instala `psql` y define también `$env:DATABASE_URL` en esta terminal con la misma conexión; PowerShell no carga `.env` automáticamente:

   ```powershell
   Get-ChildItem backend/migrations/*.up.sql | Sort-Object Name | ForEach-Object {
     psql $env:DATABASE_URL -f $_
   }
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

## Backend con PM2 (sin Docker)

PM2 se instala como dependencia del proyecto con `pnpm install`; no necesitas instalarlo globalmente. Los comandos siguientes se ejecutan desde la raíz del repositorio.

Antes de iniciar, configura `DATABASE_URL` en el `.env` de la raíz y aplica las migraciones a tu PostgreSQL local o remoto. El backend necesita la base de datos para arrancar. Docker queda como alternativa opcional.

```powershell
pnpm install
pnpm backend:start
pnpm backend:status
pnpm backend:logs
```

`backend:start` compila Go en `backend/bin/` y ejecuta ese binario con la configuración de `ecosystem.config.cjs`. PM2 inicia una sola instancia, reinicia el proceso si termina y espera 3 segundos entre intentos. Tras 10 reinicios inestables se detiene para evitar un bucle cuando hay una configuración incorrecta o no está disponible PostgreSQL. Revisa los logs, corrige la causa y ejecuta `pnpm backend:restart`.

Para detenerlo o reiniciarlo después de cambiar `.env`:

```powershell
pnpm backend:stop
pnpm backend:restart
```

Para actualizar el código, detén el backend antes de recompilar (Windows bloquea el ejecutable mientras está en uso):

```powershell
pnpm backend:stop
pnpm backend:build
pnpm backend:restart
```

Comprueba la API en `http://localhost:8080/healthz`. PM2 mantiene el backend en segundo plano cuando cierras la terminal, pero esta configuración no lo inicia automáticamente al reiniciar Windows. Tampoco detecta por sí sola una API bloqueada que mantiene el proceso abierto.

## Acceso con Google o apodo

En `/login` puedes entrar con Google o como invitado con apodo. Google permite recuperar la misma cuenta y su historial; al salir como invitado se eliminan su perfil y sus mensajes. Para entrar necesitas el backend y PostgreSQL con las tres migraciones aplicadas. [Configurar Google y conocer la política de invitados](docs/google-y-invitados.md).

## Chat y TLS

Entra con un apodo y selecciona **Nuevo chat** para buscar otro usuario. Para probar dos sesiones, abre una ventana de incógnito con otro apodo. El chat guarda los mensajes, recupera el historial por páginas, confirma lecturas e indica quién está escribiendo.

- [Configurar Google e invitados temporales](docs/google-y-invitados.md)
- [Configurar HTTPS/WSS local y en producción](docs/tls.md)
- [Ejemplo de lo que guarda cada tabla](docs/datos-ejemplo.md)

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
| `WS_TICKET_TTL` | Vida del ticket de WebSocket (`30s`) |
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
| `POST` | `/api/auth/guest` | Listo (apodo, sin correo ni contraseña) |
| `POST` | `/api/auth/register` | Disponible en API; la interfaz usa apodo |
| `POST` | `/api/auth/login` | Disponible en API; la interfaz usa apodo |
| `POST` | `/api/auth/logout` | Listo |
| `GET` | `/api/me` | Listo |
| `GET` | `/api/users` | Listo |
| `GET` | `/api/conversations` | Listo |
| `POST` | `/api/conversations` | Listo |
| `GET` | `/api/conversations/{id}/messages` | Listo, con paginación |
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
- Repositorios PostgreSQL de usuarios y chat implementados, con control de participantes, idempotencia y lecturas persistentes.
- Interfaz conectada al backend con selección de usuarios, historial, indicadores de escritura, presencia y temas claro/oscuro.

## Próximos pasos

1. Ampliar búsqueda de usuarios y paginación de conversaciones.
2. Aplicar migraciones al arrancar o con una herramienta (`golang-migrate`).
3. Reintentar mensajes pendientes después de recuperar la conexión.
4. Agregar archivos adjuntos.
5. Ejecutar las pruebas de integración en CI.
6. Redis Pub/Sub si se despliegan varias instancias.

## Despliegue en Netlify + Render

El frontend se publica en Netlify y el backend Go en Render. El repositorio incluye `netlify.toml` y `render.yaml` para que cada servicio construya su carpeta. La API HTTP pasa por un proxy de Nuxt en el dominio del frontend; el WebSocket usa la URL pública de Render y un ticket de un solo uso.

Sigue [la guía de despliegue](docs/despliegue.md) para configurar el mismo repositorio en ambos servicios, las variables y el callback de Google. Los archivos `.env.production.example` y `frontend/.env.production.example` son referencias sin credenciales.

En producción, Render y Netlify gestionan HTTPS. Los certificados locales y PM2 se usan solo en los entornos donde los configuraste; Render compila y supervisa el binario Go directamente.
