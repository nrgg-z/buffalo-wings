-- 0003: Bitácora de auditoría inmutable (RF-07).
--
-- audit_logs es un registro de solo inserciones. El código de la aplicación ya
-- no tiene métodos para modificar ni borrar, y estos triggers lo garantizan
-- también a nivel de base de datos: cualquier UPDATE o DELETE falla con error.
--
-- Nota: crear triggers con el binary log activo exige privilegio SUPER o
-- log_bin_trust_function_creators=1. Por eso esta migración se aplica como root
-- (el usuario buffalo_user no tiene ese privilegio). Ver backend/README.md.
--
-- Para quitar la protección (por ejemplo, al reiniciar datos de desarrollo):
--   DROP TRIGGER audit_logs_no_update; DROP TRIGGER audit_logs_no_delete;

DROP TRIGGER IF EXISTS audit_logs_no_update;
DROP TRIGGER IF EXISTS audit_logs_no_delete;

DELIMITER //

CREATE TRIGGER audit_logs_no_update
BEFORE UPDATE ON audit_logs
FOR EACH ROW
BEGIN
    SIGNAL SQLSTATE '45000'
        SET MESSAGE_TEXT = 'audit_logs es inmutable: no se permite UPDATE';
END//

CREATE TRIGGER audit_logs_no_delete
BEFORE DELETE ON audit_logs
FOR EACH ROW
BEGIN
    SIGNAL SQLSTATE '45000'
        SET MESSAGE_TEXT = 'audit_logs es inmutable: no se permite DELETE';
END//

DELIMITER ;
