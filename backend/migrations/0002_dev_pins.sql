-- 0002: PINs de empleados para DESARROLLO.
--
-- ⚠️ SOLO PARA DESARROLLO. NO aplicar en producción: estos PINs son públicos
-- (están en el repositorio). En un despliegue real, cada empleado debe tener
-- un PIN propio y secreto.
--
-- Los hashes de la migración 0001 eran de relleno y no correspondían a PINs
-- de 4 dígitos. La 0001 no se edita (ya fue aplicada), por eso se corrigen aquí.
--
--   EMP-01 (Gerente)   -> 1111
--   EMP-02 (Anfitriona)-> 2222
--   EMP-03 (Mesero)    -> 3333
--   EMP-04 (Cocina)    -> 4444
--   EMP-05 (Cajero)    -> 5555
--
-- Es idempotente: se puede ejecutar varias veces. También limpia intentos
-- fallidos y bloqueos para dejar a cada empleado listo para entrar.

UPDATE users SET pin_hash = '$2a$10$IZ9XWditklffIsKYGhEibuCSb0L9oQUZGyv1vmaSAadHqEBU4kAde', failed_attempts = 0, locked_until = NULL WHERE employee_code = 'EMP-01';
UPDATE users SET pin_hash = '$2a$10$ULuu9IC.WAp1Kr0GOrumy.2XglTQcC836ZFiozo.zr3xuKmv6YvjO', failed_attempts = 0, locked_until = NULL WHERE employee_code = 'EMP-02';
UPDATE users SET pin_hash = '$2a$10$xwk5C1dV6zuELgsmufQRZeAJWkTk3FkQ2weTgV4lrYLRViQ0mx37u', failed_attempts = 0, locked_until = NULL WHERE employee_code = 'EMP-03';
UPDATE users SET pin_hash = '$2a$10$NcBdDCT.WW.Nyif5nhKP3Ok7efKBu03bPbXjLTfwsqsPmdJ3B7AWO', failed_attempts = 0, locked_until = NULL WHERE employee_code = 'EMP-04';
UPDATE users SET pin_hash = '$2a$10$qaL/nMboppLrihMwRtx/I.kZlqtUmFshzdqNdHckhXo7acxnTnvjS', failed_attempts = 0, locked_until = NULL WHERE employee_code = 'EMP-05';
