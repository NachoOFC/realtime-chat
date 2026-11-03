# Ejemplo de los datos guardados

Ejemplo ilustrativo: Ana envía «Hola, Luis 👋» y Luis todavía no abre el chat. Se usan etiquetas cortas para facilitar la lectura; en PostgreSQL los identificadores son UUID completos.

## users

| id | username | email | password_hash | is_guest | created_at |
| --- | --- | --- | --- | --- | --- |
| A | Ana | NULL | NULL | true | 2026-11-02 22:00:00Z |
| B | Luis | NULL | NULL | true | 2026-11-02 22:01:00Z |

`avatar_url` y `last_seen_at` pueden ser NULL. La sesión viaja en una cookie HttpOnly; no se guarda como una columna de esta tabla. `password_hash` solo se usa para cuentas con contraseña, no para los invitados con apodo.

## conversations

| id | direct_key | created_at |
| --- | --- | --- |
| C | A:B | 2026-11-02 22:02:00Z |

En la base, `direct_key` contiene los dos UUID ordenados y separados por `:`. Así Ana y Luis comparten una sola conversación aunque ambos intenten crearla.

## conversation_members

| conversation_id | user_id | last_read_message_id | unread_count | joined_at |
| --- | --- | --- | --- | --- |
| C | A | M | 0 | 2026-11-02 22:02:00Z |
| C | B | NULL | 1 | 2026-11-02 22:02:00Z |

Hay una fila por participante. Cuando Luis abre la conversación y se confirma la lectura, su `last_read_message_id` pasa a M y `unread_count` a 0. Los mensajes posteriores vuelven a aumentar el contador.

## messages

| id | conversation_id | sender_id | content | client_message_id | created_at | edited_at |
| --- | --- | --- | --- | --- | --- | --- |
| M | C | A | Hola, Luis 👋 | X | 2026-11-02 22:03:00Z | NULL |

El navegador genera X antes de enviar el mensaje. Si reintenta el mismo envío con X, el backend devuelve el mensaje existente sin guardar otro ni incrementar dos veces el contador. Reutilizar X con otro contenido o remitente produce un error.

HTTPS y WSS usan TLS para cifrar la comunicación. La conexión del backend a Neon también debe conservar los parámetros TLS de su URL. El campo `content` sigue siendo texto legible para el servidor y para quien tenga permisos de lectura en la base: no hay cifrado de extremo a extremo.
