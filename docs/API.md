# API — Buffalo Wings POS

Documentación de los endpoints del backend, pensada para quien consume la API desde el frontend (Vue 3).

- **URL base (local):** `http://localhost:8080`
- **Prefijo:** `/api/v1`
- **Formato:** JSON con `snake_case`
- **CORS:** permitido solo para los orígenes de `backend/config/app.yaml` (hoy `http://localhost:5173`). Se aceptan los métodos `GET, POST, PUT, PATCH, DELETE, OPTIONS` y las cabeceras `Authorization` y `Content-Type`.

## Convenciones

### Formato de error

Todos los errores usan el mismo cuerpo:

```json
{ "error": "codigo_corto", "message": "Texto para mostrar al usuario" }
```

El campo `error` es estable (úsalo en el código del frontend). El campo `message` es para mostrar al usuario y puede cambiar.

### Códigos HTTP

| Código | Significado |
|---|---|
| 200 | Éxito |
| 204 | Éxito sin cuerpo (por ejemplo, preflight de CORS) |
| 400 | Petición inválida (JSON mal formado, campos faltantes) |
| 401 | Sin credenciales, o token inválido o expirado |
| 403 | Autenticado, pero sin permiso por rol (o usuario inactivo) |
| 404 | El recurso no existe |
| 409 | Conflicto (por ejemplo, mesa ocupada) |
| 423 | Usuario bloqueado temporalmente por intentos fallidos |
| 429 | Demasiadas peticiones: se superó el límite por minuto. Incluye la cabecera `Retry-After` (segundos) |
| 500 | Error interno. El detalle solo queda en el log del servidor, nunca se envía al cliente |

### Cabeceras

Todas las respuestas llevan `Cache-Control: no-store`.

### Autenticación

1. Se obtiene un token con `POST /api/v1/auth/login`.
2. En cada petición protegida se envía: `Authorization: Bearer <token>`.
3. El servidor valida firma y expiración en **cada** petición.

Detalles del token:

- Dura **60 minutos** (`token_expiration_minutes` en `security.yaml`). No hay renovación: al vencer, hay que volver a iniciar sesión.
- La **inactividad de 180 s** la controla el **cliente** (el frontend debe cerrar la sesión). El valor está en `GET /api/v1/config/public` (`inactivity_timeout_sec`).
- Claims del JWT (HS256): `uid`, `emp` (código de empleado), `role`, `sub`, `iat`, `exp`.

> [!WARNING]
> **Un token de un empleado desactivado sigue siendo válido hasta que expire** (máximo 60 minutos). El servidor valida el token sin consultar la base de datos, para que cada petición sea rápida. La única excepción es `GET /api/v1/auth/me`, que sí consulta la base de datos y responde `403 user_inactive` si el empleado fue desactivado. El empleado desactivado no podrá iniciar sesión de nuevo.

### Roles

`ADMIN`, `MANAGER`, `HOSTESS`, `WAITER`, `KITCHEN`, `CASHIER`.

**`ADMIN` puede acceder a todos los endpoints protegidos**, aunque no aparezca en la columna de roles. `MANAGER` solo accede a los endpoints donde se lista.

### Empleados de desarrollo

> [!CAUTION]
> **Solo para desarrollo.** Estos PINs están en el repositorio (migración `0002_dev_pins.sql`). Nunca se deben usar en producción.

| Código | Nombre | Rol | PIN |
|---|---|---|---|
| `EMP-01` | Gabriela T. (Gerente) | `MANAGER` | `1111` |
| `EMP-02` | Andrea M. (Hostess) | `HOSTESS` | `2222` |
| `EMP-03` | Carlos R. (Mesero) | `WAITER` | `3333` |
| `EMP-04` | Roberto G. (Cocina) | `KITCHEN` | `4444` |
| `EMP-05` | Luis F. (Cajero) | `CASHIER` | `5555` |

---

## Endpoints

### `GET /api/v1/healthz`

Estado del servicio y de la base de datos.

- **Roles:** público (sin token)
- **Respuestas:** `200` (todo bien) o `503` (la base de datos no responde)

```bash
curl http://localhost:8080/api/v1/healthz
```

```json
{
  "status": "ok",
  "service": "buffalo-wings-pos-api",
  "db_status": "up",
  "timestamp_utc": "2026-11-18T21:50:00Z"
}
```

---

### `GET /api/v1/config/public`

Parámetros de negocio que el frontend necesita conocer.

- **Roles:** público (sin token)

```bash
curl http://localhost:8080/api/v1/config/public
```

```json
{
  "pin_length": 4,
  "inactivity_timeout_sec": 180,
  "tax_rate": 0.13,
  "suggested_tip_rate": 0.1,
  "kds_warning_minutes": 10,
  "kds_critical_minutes": 18,
  "enabled_payment_methods": ["CASH", "CARD", "QR"]
}
```

---

### `POST /api/v1/auth/login`

Inicia sesión con código de empleado y PIN.

- **Roles:** público (sin token)
- **Límite:** **15 peticiones por minuto por IP** (`login_rate_limit_per_min`). Al superarlo responde `429` con `Retry-After`.
- **Bloqueo:** tras **3 PINs incorrectos** seguidos, el usuario queda bloqueado **5 minutos** (`423`).
- **Auditoría:** cada intento queda en la bitácora (`LOGIN_SUCCESS`, `LOGIN_FAILED`, `USER_LOCKED`). El PIN nunca se guarda.

**Petición**

| Campo | Tipo | Descripción |
|---|---|---|
| `employee_code` | string | Código del empleado, por ejemplo `EMP-03` |
| `pin` | string | PIN de 4 dígitos (como texto, no número) |

No se aceptan campos adicionales.

```bash
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"employee_code":"EMP-03","pin":"3333"}'
```

**Respuesta `200`**

```json
{
  "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "expires_at": "2026-11-19T04:53:25.567151351Z",
  "user": {
    "id": 3,
    "employee_code": "EMP-03",
    "full_name": "Carlos R. (Mesero)",
    "role": "WAITER"
  }
}
```

**Errores**

| Código | `error` | Cuándo |
|---|---|---|
| 400 | `invalid_request` | JSON inválido, campos faltantes o campos desconocidos |
| 401 | `invalid_credentials` | Código o PIN incorrecto (el mismo error si el código no existe) |
| 403 | `user_inactive` | El empleado está desactivado |
| 423 | `user_locked` | Bloqueado por intentos fallidos |
| 429 | `too_many_requests` | Se superó el límite por minuto (ver `Retry-After`) |
| 500 | `internal_error` | Error interno, incluida una falla al escribir la bitácora |

---

### `GET /api/v1/auth/me`

Devuelve los datos actuales del empleado dueño del token.

- **Roles:** cualquier usuario autenticado
- **Consulta la base de datos** (a diferencia de otras rutas protegidas), así que refleja el nombre y el estado reales.

```bash
curl http://localhost:8080/api/v1/auth/me \
  -H "Authorization: Bearer $TOKEN"
```

**Respuesta `200`**

```json
{
  "user": {
    "id": 3,
    "employee_code": "EMP-03",
    "full_name": "Carlos R. (Mesero)",
    "role": "WAITER"
  },
  "token_expires_at": "2026-11-19T04:53:25Z"
}
```

**Errores**

| Código | `error` | Cuándo |
|---|---|---|
| 401 | `unauthorized` | Falta la cabecera `Authorization` o no es de tipo `Bearer` |
| 401 | `invalid_token` | Token manipulado, expirado, o su usuario ya no existe |
| 403 | `user_inactive` | El empleado fue desactivado después de iniciar sesión |

---

## Bitácora de auditoría (referencia)

La tabla `audit_logs` es **inmutable** (RF-07): solo admite inserciones. Un trigger de MySQL rechaza cualquier `UPDATE` o `DELETE`.

| `action` | Cuándo | `detail` |
|---|---|---|
| `LOGIN_SUCCESS` | Login correcto | `employee_code` |
| `LOGIN_FAILED` | PIN incorrecto, código inexistente, formato inválido, usuario inactivo o intento estando bloqueado | `employee_code`, `reason` (`bad_pin`, `unknown_user`, `invalid_pin_format`, `inactive`, `locked`) y, para `bad_pin`, `failed_attempts` |
| `USER_LOCKED` | Se alcanzó el máximo de intentos | `employee_code`, `failed_attempts`, `locked_until` |

Las peticiones rechazadas por el límite de intentos (429) no llegan al servicio de login, por lo que no generan entrada en la bitácora.

## Cómo proteger un endpoint nuevo (para quien programa el backend)

```go
// Cualquier usuario autenticado:
mux.Handle("GET /api/v1/ejemplo", authhttp.Protected(verifier)(handler))

// Solo ciertos roles (ADMIN siempre pasa):
mux.Handle("POST /api/v1/tables/{id}/assign",
    authhttp.Protected(verifier, domain.RoleHostess, domain.RoleManager)(handler))

// Dentro del handler, quién hizo la petición:
principal, _ := authhttp.PrincipalFrom(r.Context())
```
