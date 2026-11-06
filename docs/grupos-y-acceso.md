# Grupos y acceso a la última cuenta

En **Nuevo chat → Nuevo grupo**, escribe un nombre y selecciona las personas. Cada grupo admite hasta 50 integrantes, incluido quien lo crea. Los mensajes muestran el nombre de su autor y llegan en tiempo real a los integrantes conectados.

Desde **Personas** puedes consultar los integrantes, agregar personas y salir del grupo. Cualquier integrante puede invitar. Quien se incorpora puede consultar el historial anterior; quien sale pierde acceso al grupo. Salir conserva sus mensajes para los demás. Cuando se elimina una cuenta de invitado al cerrar sesión o vencer, también se eliminan sus mensajes en los grupos. El grupo sigue existiendo mientras tenga integrantes.

## Migración

Aplica `backend/migrations/0004_group_conversations.up.sql` una sola vez para habilitar grupos. Agrega el tipo y el nombre de las conversaciones y permite que los grupos no tengan una clave de conversación directa. Conserva usuarios, mensajes y conversaciones existentes. La versión actual también necesita la migración 0005 para el borrado de chats. Las pruebas de integración aplican las cinco migraciones en un esquema temporal separado.

La migración de reversión se detiene si existen grupos para evitar perderlos.

## Cuenta recordada

Al salir de una cuenta Google se conserva únicamente su nombre y correo en el almacenamiento local de ese navegador. La pantalla de acceso muestra **Continuar como…**, **Usar otra cuenta de Google** y una opción para olvidar la cuenta recordada.

El acceso rápido abre Google con `login_hint`; Google sigue autenticando y el backend verifica el token, PKCE, el estado y el nonce. Recordar la cuenta no conserva una sesión abierta ni guarda contraseñas, tokens o claves en el almacenamiento local. Los invitados no se recuerdan como cuentas recuperables.

La cuenta recordada se muestra solo en ese navegador y dispositivo. Usa **Olvidar cuenta** si compartes el navegador.
