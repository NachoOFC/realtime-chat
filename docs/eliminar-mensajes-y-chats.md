# Eliminar mensajes y chats

El menú de cada mensaje propio permite **Eliminar mensaje**. Tras confirmar, el servidor verifica que la persona pertenezca a la conversación y sea la autora. El mensaje se elimina para todos y se actualizan los contadores, el último mensaje y los clientes conectados. Otras personas no pueden borrar mensajes ajenos.

En el menú de la cabecera, **Eliminar chat** limpia el historial visible de esa persona y quita el chat de su lista. Las demás personas conservan su historial. La cuenta continúa perteneciendo al grupo; **Salir del grupo** es una acción distinta. Un mensaje nuevo hace que el chat reaparezca, mostrando únicamente los mensajes posteriores a la limpieza. Abrir de nuevo un chat directo desde **Nuevo chat** también permite conversar sin recuperar el historial eliminado.

IACHAT tiene una opción para borrar su historial local. Sus mensajes no se guardan en PostgreSQL.

## Migración

Aplica `backend/migrations/0005_chat_deletion.up.sql` una sola vez después de las cuatro migraciones anteriores y antes de desplegar el backend actualizado. Agrega `cleared_at` e `is_hidden` a `conversation_members`, sin borrar registros existentes. La reversión se detiene si existen historiales limpiados, para evitar mostrarlos de nuevo.

## Interfaz

La interfaz toma como referencia la distribución y el estilo de [TailAdmin Chat](https://vue-demo.tailadmin.com/chat): navegación lateral, paneles separados, búsqueda, avatares, estado de conexión y burbujas azules. Utiliza componentes propios y conserva las funciones reales de la app.
