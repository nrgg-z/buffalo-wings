# Decisiones de diseño

Una línea por decisión. Más reciente al final de cada sección.

## HU-03 — Mesas

- `AGENTS.md` no existe en el repo; se siguió `backend/README.md` (arquitectura hexagonal, estilo del módulo auth).
- Módulo nuevo `internal/tables` con domain / port / service / adapter (mysqlrepo, httpapi), igual que auth.
- La traducción de estados (FREE→LIBRE, etc.) vive en el dominio (`Status.APIName`); la base y el código usan inglés.
- Migración `0004_tables_merge.sql`: solo `merged_into_id` (FK a `dining_tables`); sin tabla de adyacencia.
- Adyacencia = misma zona + códigos consecutivos (mismo prefijo, número +1), calculado en el dominio.
- Combinar: la mesa de código más bajo es la principal; las demás apuntan a ella con `merged_into_id`.
- La capacidad efectiva de la principal es la suma de las combinadas (`effective_capacity`); `capacity` no se modifica.
- Una mesa combinada no se puede asignar ni volver a combinar (409 `table_merged` / `merge_not_allowed`).
- No hay ruta para deshacer: al marcar limpia la principal se liberan las combinadas (`merged_into_id = NULL`).
- Concurrencia: el cliente envía `version`; todo UPDATE lleva `WHERE id = ? AND version = ?` y sube la versión; 0 filas → 409 `version_conflict`.
- Combinar es transaccional y también sube la versión de la principal para proteger contra carreras.
- Asignar: solo mesa LIBRE, 1 ≤ comensales ≤ capacidad efectiva; pasa a OCUPADA. No se asigna mesero (campo `assigned_waiter_id` solo se lee).
- Marcar limpia: solo desde POR_LIMPIAR; deja LIBRE, 0 comensales y sin mesero.
- Errores de regla de negocio → 409 (estado) o 400 (datos inválidos); `version` ausente o 0 → 400.
- Permisos con `authhttp.Protected` (ADMIN siempre pasa): ver mapa = HOSTESS/WAITER/MANAGER; asignar y combinar = HOSTESS/MANAGER; limpiar = WAITER/HOSTESS/MANAGER.
- Rutas: `GET /tables`, `POST /tables/{id}/assign`, `POST /tables/merge`, `POST /tables/{id}/clean`.
- Sin bitácora de auditoría en mesas (no se pidió); sin transferir cuenta.
- Solo se listan mesas con `is_active = 1`.
- Pruebas automáticas solo del dominio (reglas puras); el flujo completo se probó con un único curl.
