# Acceso con Google e invitados temporales

Hay dos tipos de acceso y ambos permiten conversar entre sí:

| Acceso | Identidad | Al pulsar Salir |
| --- | --- | --- |
| Google | Cuenta estable identificada por el `sub` verificado de Google | Se cierra la sesión. Al volver con la misma cuenta recuperas tu usuario y conversaciones. |
| Invitado | Apodo y acceso temporal, sin cuenta recuperable | Se eliminan el perfil, sus participaciones y todos los mensajes enviados por ese invitado. |

La insignia **Google** indica el proveedor usado para iniciar sesión. No afirma que la persona sea una identidad pública verificada. Los correos de otros usuarios no se exponen en el directorio.

## Preparar la base de datos

En una instalación existente, ejecuta solo `backend/migrations/0003_google_and_temporary_guests.up.sql` después de las migraciones 0001 y 0002. En una base nueva, aplica las tres en orden.

La migración agrega `google_subject` y `guest_expires_at` a `users`. No borra cuentas al ejecutarla; asigna a los invitados anteriores un plazo de transición de 24 horas. Cuando el backend nuevo está activo, elimina los invitados cuyo acceso vence. Las cuentas Google y las cuentas con contraseña permanecen.

## Configurar Google

1. Crea un proyecto en [Google Cloud Console](https://console.cloud.google.com/).
2. En **Google Auth Platform**, configura la pantalla de consentimiento. Si la aplicación está en pruebas, agrega tus cuentas de prueba en Audience/Test users.
3. Crea un cliente OAuth de tipo **Web application**.
4. Registra exactamente esta URI de redirección para desarrollo:

   ```text
   https://localhost:8080/api/auth/google/callback
   ```

5. Completa las tres variables en el `.env` de la raíz (nunca en `frontend/.env`):

   ```env
   GOOGLE_CLIENT_ID=tu-cliente.apps.googleusercontent.com
   GOOGLE_CLIENT_SECRET=tu-secreto
   GOOGLE_REDIRECT_URL=https://localhost:8080/api/auth/google/callback
   ```

6. Reinicia el backend. Abre `https://localhost:3000/login`; «Continuar con Google» se habilita cuando la configuración está completa.

`FRONTEND_URL` es opcional y por defecto toma `CORS_ORIGIN`; debe ser el origen del frontend, por ejemplo `https://localhost:3000`. Para Netlify + Render, registra el callback HTTPS del frontend (`https://tu-chat.netlify.app/api/auth/google/callback`), cuyo proxy lo entrega al backend, y actualiza `GOOGLE_REDIRECT_URL`, `FRONTEND_URL` y `CORS_ORIGIN` con las URLs reales. El tipo de aplicación y la URI exacta son requisitos de [Google OAuth](https://developers.google.com/identity/protocols/oauth2/web-server).

El flujo usa authorization code con PKCE, `state` firmado y cookie HttpOnly, `nonce`, validación de firma RS256, emisor, audiencia, vencimiento y correo verificado. Solo solicita `openid`, `email` y `profile`; no accede a Gmail ni guarda los tokens de Google. La clave de identidad es `sub`, no el correo. No vincula automáticamente una cuenta local o de invitado por compartir correo o apodo. [Google OpenID Connect](https://developers.google.com/identity/openid-connect/openid-connect)

Sin credenciales de Google, el acceso de invitado sigue disponible y `/api/auth/providers` informa que Google no está habilitado. Las pruebas usan un proveedor simulado y tokens firmados de prueba; la verificación interactiva real requiere tu cliente OAuth.

## Qué ocurre con los invitados

La cookie de invitado es una cookie de sesión, sin Max-Age ni Expires. No permite recuperar la identidad después de perderla. Algunos navegadores pueden restaurar cookies de sesión al restaurar ventanas: cerrar el navegador no garantiza que el servidor reciba una notificación de salida.

Por eso la eliminación inmediata se realiza al pulsar **Salir**, tras una confirmación explícita. Si el invitado abandona el navegador sin salir, su acceso vence según `ACCESS_TOKEN_TTL`; el backend elimina los invitados vencidos al iniciar y cada minuto. Si está apagado, esa limpieza se realiza cuando vuelva a iniciar.

Las conexiones WebSocket de un invitado eliminado se cierran y su cookie anterior deja de autenticar peticiones. El apodo queda disponible para una identidad nueva, que no recupera los mensajes anteriores.

Los mensajes de los otros participantes se conservan en un chat cerrado con el nombre «Invitado que salió». Se actualizan las lecturas y los contadores y ya no se puede enviar a ese chat.

Para publicar la app, sigue [la guía de despliegue](despliegue.md).
