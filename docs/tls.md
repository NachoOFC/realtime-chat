# HTTPS y WebSockets seguros

## Desarrollo local

La configuración local ya se puede preparar desde la raíz:

```powershell
node scripts/setup-local-tls.cjs
```

El comando genera una CA exclusiva de desarrollo y un certificado para localhost en `.certs/` (ignorado por Git), y actualiza únicamente las opciones TLS, CORS y cookies de los archivos `.env`. No cambia `DATABASE_URL` ni la clave JWT. No sobrescribe certificados existentes.

Para que el navegador confíe en la CA que acabas de generar, importa **solo esa CA propia** al almacén de tu usuario en Windows:

```powershell
Import-Certificate -FilePath .certs/ca.crt -CertStoreLocation Cert:\CurrentUser\Root
```

Cierra las pestañas anteriores y reinicia los procesos que estaban sirviendo HTTP. Backend:

```powershell
cd backend
go run ./cmd/server
```

En otra terminal, desde la raíz:

```powershell
pnpm.cmd --dir frontend dev
```

Abre `https://localhost:3000`. La API responde en `https://localhost:8080/healthz` y el chat utiliza `wss://localhost:8080/ws`. No uses las URLs HTTP anteriores con esta configuración.

El script de desarrollo del frontend pasa esta CA a Node mediante `NODE_EXTRA_CA_CERTS`, de modo que la verificación de sesión durante el renderizado del servidor también comprueba el certificado del backend. No se desactiva la validación TLS.

Si usas PM2, detén, recompila y reinicia el backend para cargar los cambios:

```powershell
pm2 stop realtime-chat-backend
cd backend
go build -o bin/server.exe ./cmd/server
cd ..
pm2 restart realtime-chat-backend --update-env
```

Los certificados duran un año. Para renovarlos, mueve los archivos viejos fuera de `.certs/`, ejecuta de nuevo el generador e importa la nueva CA. No uses estos certificados de desarrollo en un servidor público.

## Despliegue

Con un hosting o proxy que gestione HTTPS (por ejemplo Caddy), el proxy recibe HTTPS/WSS con un certificado público y reenvía las peticiones HTTP al backend por la interfaz de loopback. En ese caso, deja vacíos `TLS_CERT_FILE` y `TLS_KEY_FILE`, configura `HTTP_ADDR=127.0.0.1:8080`, `COOKIE_SECURE=true` y el origen HTTPS real en `CORS_ORIGIN`.

Si Go recibe TLS directamente, configura `TLS_CERT_FILE` y `TLS_KEY_FILE` con las rutas al certificado público y su clave privada. El backend admite TLS 1.2 y 1.3. Configura el frontend con `NUXT_PUBLIC_API_BASE=https://...` y `NUXT_PUBLIC_WS_BASE=wss://...`.

No publiques `.env`, claves privadas ni `.certs/`. Mantén los parámetros de seguridad que Neon proporciona en `DATABASE_URL`.

## Pruebas

```powershell
cd backend
go test ./...
$env:CHAT_INTEGRATION='1'
go test ./internal/httpapi -run TestChatOverTLS -count=1
```

La prueba de integración toma la conexión del `.env` de la raíz (o `CHAT_TEST_DATABASE_URL`), usa el endpoint directo de Neon y crea un esquema aleatorio para sus tablas. Verifica el aislamiento antes de migrar y elimina el esquema al terminar; no usa ni modifica las tablas `public` de la app. Requiere permisos para crear esquemas.
