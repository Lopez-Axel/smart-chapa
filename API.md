# API Reference — Smart Chapa

**Base URL:** `http://<host>:8080/api`

**Autenticación:** Los endpoints protegidos requieren header `Authorization: Bearer <token>`. El token se obtiene de `POST /auth/login` y expira en 7 días.

---

## Autenticación (público)

### `POST /auth/register`

Registra un nuevo usuario.

| Campo | Tipo | Obligatorio |
|-------|------|-------------|
| `name` | string | sí |
| `email` | string | sí |
| `password` | string | sí |

**Respuesta `201`:**
```json
{"id": 1, "name": "Axel", "email": "axel@mail.com", "created_at": "..."}
```
**Errores:** `400` (faltan campos), `409` (email ya registrado)

---

### `POST /auth/login`

Inicia sesión y devuelve un JWT.

| Campo | Tipo | Obligatorio |
|-------|------|-------------|
| `email` | string | sí |
| `password` | string | sí |

**Respuesta `200`:**
```json
{"token": "eyJhbGciOiJIUzI1NiIs..."}
```
**Errores:** `400` (faltan campos), `401` (credenciales incorrectas)

---

## Casas (requiere JWT)

### `POST /houses`

Crea una casa. El creador se asigna automáticamente como `owner`.

| Campo | Tipo | Obligatorio |
|-------|------|-------------|
| `name` | string | sí |
| `address` | string | no |
| `city` | string | no |
| `country` | string | no |
| `latitude` | number | no |
| `longitude` | number | no |

**Respuesta `201`:**
```json
{
  "id": 1,
  "name": "Mi Casa",
  "address": "Av. Siempre Viva 742",
  "city": "Buenos Aires",
  "country": "Argentina",
  "latitude": -34.6037,
  "longitude": -58.3816,
  "created_at": "..."
}
```
**Errores:** `400` (name requerido)

---

### `GET /houses`

Lista las casas a las que el usuario tiene acceso (owner o member).

**Respuesta `200`:**
```json
[
  {
    "id": 1,
    "name": "Mi Casa",
    "address": "Av. Siempre Viva 742",
    "city": "Buenos Aires",
    "country": "Argentina",
    "latitude": -34.6037,
    "longitude": -58.3816,
    "created_at": "..."
  }
]
```

---

### `GET /houses/{id}/devices`

Lista los dispositivos asignados a una casa.

| Parámetro URL | Tipo |
|---------------|------|
| `id` | integer (ID de la casa) |

**Respuesta `200`:**
```json
[
  {
    "id": 3,
    "name": "ESP32 Cocina",
    "token": "a1b2c3d4e5f6...",
    "user_id": 1,
    "house_id": 1,
    "created_at": "..."
  }
]
```
**Errores:** `404` (casa no encontrada o sin acceso)

---

### `POST /houses/{id}/members`

Agrega un miembro a una casa. Solo el `owner` puede hacerlo.

| Parámetro URL | Tipo |
|---------------|------|
| `id` | integer (ID de la casa) |

| Campo | Tipo | Obligatorio |
|-------|------|-------------|
| `user_id` | integer | sí |
| `role` | string | no (default `"member"`) |

**Respuesta `200`:** `{"status": "ok"}`

**Errores:** `400` (user_id requerido), `403` (solo el propietario)

---

## Dispositivos (requiere JWT)

### `POST /devices`

Crea un dispositivo físico (ESP32). Genera un token único automáticamente.

| Campo | Tipo | Obligatorio |
|-------|------|-------------|
| `name` | string | sí |
| `house_id` | integer | no (0 = sin casa) |

**Respuesta `201`:**
```json
{
  "id": 3,
  "name": "ESP32 Cocina",
  "token": "a1b2c3d4e5f6...",
  "user_id": 1,
  "house_id": 1,
  "created_at": "..."
}
```
**Errores:** `400` (name requerido), `404` (house_id no existe), `403` (sin acceso a la casa)

---

### `GET /devices`

Lista los dispositivos del usuario autenticado.

**Respuesta `200`:**
```json
[
  {
    "id": 3,
    "name": "ESP32 Cocina",
    "token": "a1b2c3d4e5f6...",
    "user_id": 1,
    "house_id": 1,
    "created_at": "..."
  }
]
```

---

### `DELETE /devices/{id}`

Elimina un dispositivo. También elimina en cascada sus actuadores y eventos.

| Parámetro URL | Tipo |
|---------------|------|
| `id` | integer (ID del dispositivo) |

**Respuesta `200`:** `{"status": "ok"}`

**Errores:** `404` (no existe), `403` (no te pertenece)

---

## Actuadores (requiere JWT)

### `POST /actuators`

Crea un actuador (relé) en un dispositivo.

| Campo | Tipo | Obligatorio |
|-------|------|-------------|
| `device_id` | integer | sí |
| `name` | string | sí |
| `type` | string | sí (ej: `"lights"`, `"door"`, `"gate"`, `"window"`) |
| `relay_num` | integer | no (default 0, pero se recomienda 1+) |

**Respuesta `201`:**
```json
{
  "id": 5,
  "device_id": 3,
  "name": "Luz Cocina",
  "type": "lights",
  "relay_num": 1,
  "state": "off",
  "created_at": "..."
}
```
**Errores:** `400` (faltan campos), `404` (device_id no existe), `403` (sin acceso al dispositivo), `409` (relay_num duplicado en el mismo device)

---

### `GET /actuators?device_id=X`

Lista actuadores de un dispositivo.

| Query param | Tipo | Obligatorio |
|-------------|------|-------------|
| `device_id` | integer | sí |

**Respuesta `200`:**
```json
[
  {
    "id": 5,
    "device_id": 3,
    "name": "Luz Cocina",
    "type": "lights",
    "relay_num": 1,
    "state": "off",
    "created_at": "..."
  }
]
```
**Errores:** `400` (device_id requerido/inválido), `404` (device no existe), `403` (sin acceso)

---

### `GET /actuators/{id}`

Estado individual de un actuador.

| Parámetro URL | Tipo |
|---------------|------|
| `id` | integer (ID del actuador) |

**Respuesta `200`:**
```json
{
  "id": 5,
  "device_id": 3,
  "name": "Luz Cocina",
  "type": "lights",
  "relay_num": 1,
  "state": "off",
  "created_at": "..."
}
```
**Errores:** `404` (no existe), `403` (sin acceso)

**Estado posible de `state`:**

| Valor | Significado |
|-------|-------------|
| `off` | Apagado |
| `on` | Encendido |
| `pending_on` | Comando de encendido enviado, esperando confirmación del ESP32 |
| `pending_off` | Comando de apagado enviado, esperando confirmación del ESP32 |

---

### `POST /actuators/{id}/on`

Enciende un actuador. Publica MQTT en `{device_id}/{type}/cmd` y guarda el evento en `pending_on`.

| Parámetro URL | Tipo |
|---------------|------|
| `id` | integer (ID del actuador) |

**Respuesta `200`:** `{"status": "ok"}`

**MQTT publish:** `3/lights/cmd` → `{"relay":1,"state":"turn_on"}`

**Errores:** `409` (si ya está encendido, si hay un apagado pendiente)

---

### `POST /actuators/{id}/off`

Apaga un actuador. Idem anterior con `"turn_off"`.

| Parámetro URL | Tipo |
|---------------|------|
| `id` | integer (ID del actuador) |

**Respuesta `200`:** `{"status": "ok"}`

**Errores:** `409` (si ya está apagado, si hay un encendido pendiente)

---

### `GET /actuators/{id}/events`

Últimos 50 eventos de un actuador (orden descendente).

| Parámetro URL | Tipo |
|---------------|------|
| `id` | integer (ID del actuador) |

**Respuesta `200`:**
```json
[
  {
    "id": 102,
    "actuator_id": 5,
    "state": "pending_on",
    "source": "http",
    "details": "{\"relay\":1,\"state\":\"turn_on\"}",
    "created_at": "2026-06-07 12:30:00"
  }
]
```

| Campo | Descripción |
|-------|-------------|
| `state` | Estado en el momento del evento (`on`, `off`, `pending_on`, `pending_off`, o el payload raw si no se pudo parsear) |
| `source` | `"http"` (comando desde la UI) o `"mqtt"` (respuesta del ESP32) |
| `details` | Payload MQTT completo recibido del ESP32 (incluye `parse_us`, etc.) |

---

## MQTT — Comunicación con ESP32

### Backend → ESP32 (publish)

**Topic:** `{device_id}/{type}/cmd`  
**Ejemplo:** `3/lights/cmd`

**Payload:**
```json
{"relay":1,"state":"turn_on"}
```

| Campo | Valor posible |
|-------|---------------|
| `state` | `"turn_on"` o `"turn_off"` |

---

### Máquina de estados

```
off  →  POST /on   →  pending_on  →  ESP32 responde  →  on
on   →  POST /off  →  pending_off →  ESP32 responde  →  off
```

- Si el actuador está `pending_on` y se vuelve a pedir `POST /on`, se reenvía el comando MQTT (retry)
- Si el actuador está `pending_off` y se vuelve a pedir `POST /off`, se reenvía el comando MQTT (retry)
- No se puede enviar `off` si está `pending_on`, ni `on` si está `pending_off`

### ESP32 → Backend (subscribe)

**Topic:** `{device_id}/{type}/status`  
**Wildcard backend:** `+/+/status`  
**Ejemplo:** `3/lights/status`

**Payload confirmación (después de comando):**
```json
{"relay":1,"state":"turn_on","parse_us":479}
```

**Payload estado directo:**
```json
{"relay":1,"state":"on"}
```

| Campo | Valor posible |
|-------|---------------|
| `state` | `"on"`, `"off"`, `"relay_on"`, `"relay_off"`, `"turn_on"`, `"turn_off"` |

El backend normaliza los estados: `relay_on`/`turn_on` → `"on"`, `relay_off`/`turn_off` → `"off"`.

Cuando el ESP32 responde:
- Si el actuador está `pending_on` → pasa a `on`
- Si el actuador está `pending_off` → pasa a `off`
- Caso contrario → actualiza solo si el estado cambió

El payload completo se guarda en el campo `details` del evento (incluye `parse_us` y cualquier otro campo extra).

---

## Códigos de error HTTP

| Código | Significado |
|--------|-------------|
| `400` | Bad Request — falta un campo obligatorio o formato inválido |
| `401` | Unauthorized — token faltante o inválido |
| `403` | Forbidden — el recurso existe pero no te pertenece |
| `404` | Not Found — el recurso no existe |
| `409` | Conflict — duplicado (email ya registrado, relay_num duplicado), o estado bloqueado (actuador pendiente, o ya en el estado solicitado) |
| `500` | Internal Server Error — error inesperado del servidor |
