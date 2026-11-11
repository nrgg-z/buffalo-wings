-- 1. Roles y Empleados
CREATE TABLE IF NOT EXISTS roles (
    id TINYINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    code VARCHAR(20) NOT NULL UNIQUE,
    name VARCHAR(50) NOT NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS users (
    id INT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    employee_code VARCHAR(20) NOT NULL UNIQUE,
    full_name VARCHAR(100) NOT NULL,
    pin_hash VARCHAR(255) NOT NULL,
    role_id TINYINT UNSIGNED NOT NULL,
    failed_attempts TINYINT UNSIGNED NOT NULL DEFAULT 0,
    locked_until DATETIME NULL,
    is_active TINYINT(1) NOT NULL DEFAULT 1,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    FOREIGN KEY (role_id) REFERENCES roles(id)
);

-- 2. Bitácora de Auditoría
CREATE TABLE IF NOT EXISTS audit_logs (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    user_id INT UNSIGNED NULL,
    action VARCHAR(50) NOT NULL,
    entity_type VARCHAR(30) NULL,
    entity_id BIGINT UNSIGNED NULL,
    detail JSON NULL,
    ip_address VARCHAR(45) NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_audit_action (action),
    INDEX idx_audit_created_at (created_at),
    FOREIGN KEY (user_id) REFERENCES users(id)
);

-- 3. Mesas del Restaurante
CREATE TABLE IF NOT EXISTS dining_tables (
    id INT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    code VARCHAR(15) NOT NULL UNIQUE,
    zone ENUM('BARRA', 'SALON_CENTRAL', 'TERRAZA') NOT NULL,
    capacity TINYINT UNSIGNED NOT NULL,
    status ENUM('FREE', 'OCCUPIED', 'RESERVED', 'CLEANING') NOT NULL DEFAULT 'FREE',
    current_guests TINYINT UNSIGNED NOT NULL DEFAULT 0,
    assigned_waiter_id INT UNSIGNED NULL,
    version INT UNSIGNED NOT NULL DEFAULT 1,
    is_active TINYINT(1) NOT NULL DEFAULT 1,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    FOREIGN KEY (assigned_waiter_id) REFERENCES users(id)
);

-- 4. Categorías, Productos y Modificadores
CREATE TABLE IF NOT EXISTS categories (
    id INT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    name VARCHAR(60) NOT NULL,
    station ENUM('KITCHEN', 'BAR') NOT NULL,
    sort_order TINYINT UNSIGNED NOT NULL DEFAULT 0,
    is_active TINYINT(1) NOT NULL DEFAULT 1
);

CREATE TABLE IF NOT EXISTS products (
    id INT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    category_id INT UNSIGNED NOT NULL,
    name VARCHAR(100) NOT NULL,
    base_price DECIMAL(10,2) NOT NULL,
    is_available TINYINT(1) NOT NULL DEFAULT 1,
    is_active TINYINT(1) NOT NULL DEFAULT 1,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    FOREIGN KEY (category_id) REFERENCES categories(id)
);

CREATE TABLE IF NOT EXISTS modifier_groups (
    id INT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    name VARCHAR(60) NOT NULL,
    is_required TINYINT(1) NOT NULL DEFAULT 1,
    min_selections TINYINT UNSIGNED NOT NULL DEFAULT 1,
    max_selections TINYINT UNSIGNED NOT NULL DEFAULT 1,
    is_active TINYINT(1) NOT NULL DEFAULT 1
);

CREATE TABLE IF NOT EXISTS modifier_options (
    id INT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    group_id INT UNSIGNED NOT NULL,
    name VARCHAR(60) NOT NULL,
    extra_price DECIMAL(10,2) NOT NULL DEFAULT 0.00,
    is_available TINYINT(1) NOT NULL DEFAULT 1,
    is_active TINYINT(1) NOT NULL DEFAULT 1,
    FOREIGN KEY (group_id) REFERENCES modifier_groups(id)
);

CREATE TABLE IF NOT EXISTS product_modifier_groups (
    product_id INT UNSIGNED NOT NULL,
    group_id INT UNSIGNED NOT NULL,
    PRIMARY KEY (product_id, group_id),
    FOREIGN KEY (product_id) REFERENCES products(id),
    FOREIGN KEY (group_id) REFERENCES modifier_groups(id)
);

-- INSERCIÓN DE DATOS SEMILLA (Idempotente)
INSERT IGNORE INTO roles (id, code, name) VALUES
(1, 'ADMIN', 'Administrador del Sistema'),
(2, 'MANAGER', 'Gerente de Turno'),
(3, 'HOSTESS', 'Anfitriona de Entrada'),
(4, 'WAITER', 'Mesero de Salón'),
(5, 'KITCHEN', 'Jefe de Línea Cocina'),
(6, 'CASHIER', 'Cajero Principal');

INSERT IGNORE INTO users (id, employee_code, full_name, pin_hash, role_id) VALUES
(1, 'EMP-01', 'Gabriela T. (Gerente)', '$2a$10$iKq1K070P486jW/7r1wLDeR56nB1eT1aNnkl/WvH3R0kM8dZ0dFmu', 2),
(2, 'EMP-02', 'Andrea M. (Hostess)', '$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy', 3),
(3, 'EMP-03', 'Carlos R. (Mesero)', '$2a$10$oX0Z49mBqJ9Bw5X9uI3.OeAepuY2Wj66kF6HqU2r4hL5X7p8Kx5e.', 4),
(4, 'EMP-04', 'Roberto G. (Cocina)', '$2a$10$wB5mS1ZgL7jK.g0F9rL7U.l9Zt3pW0kY5mR2eQ1jH4uK8dL6mP9ey', 5),
(5, 'EMP-05', 'Luis F. (Cajero)', '$2a$10$qR7tY8uI9oP0aB1c2d3e4.mK5nL6jH7gF8dE9sA0bC1vD2wE3fG4h', 6);

INSERT IGNORE INTO dining_tables (id, code, zone, capacity) VALUES
(1, 'M-01', 'BARRA', 2),
(2, 'M-02', 'BARRA', 2),
(3, 'M-03', 'BARRA', 2),
(4, 'M-04', 'SALON_CENTRAL', 4),
(5, 'M-05', 'SALON_CENTRAL', 4),
(6, 'M-06', 'SALON_CENTRAL', 6),
(7, 'M-07', 'SALON_CENTRAL', 6),
(8, 'M-08', 'TERRAZA', 4),
(9, 'M-09', 'TERRAZA', 4),
(10, 'M-10', 'TERRAZA', 8);

INSERT IGNORE INTO categories (id, name, station, sort_order) VALUES
(1, 'Alitas Tradicionales', 'KITCHEN', 1),
(2, 'Boneless', 'KITCHEN', 2),
(3, 'Hamburguesas', 'KITCHEN', 3),
(4, 'Acompañamientos', 'KITCHEN', 4),
(5, 'Bebidas', 'BAR', 5);

INSERT IGNORE INTO products (id, category_id, name, base_price) VALUES
(101, 1, '12 Traditional Wings', 14.50),
(102, 1, '24 Traditional Wings', 26.99),
(103, 2, '10 Boneless Wings', 11.99),
(104, 3, 'Buffalo Burger', 9.50),
(105, 4, 'Papas Twister Grande', 4.25),
(106, 5, 'Cerveza Nacional', 2.75);

INSERT IGNORE INTO modifier_groups (id, name, is_required, min_selections, max_selections) VALUES
(1, 'Salsa Obligatoria', 1, 1, 1),
(2, 'Aderezo Obligatorio', 1, 1, 1);

INSERT IGNORE INTO modifier_options (id, group_id, name, extra_price) VALUES
(10, 1, 'Atomic Hot', 0.00),
(11, 1, 'Mango Habanero', 0.00),
(12, 1, 'Miel Mostaza', 0.00),
(13, 1, 'BBQ Clásica', 0.00),
(20, 2, 'Blue Cheese', 0.00),
(21, 2, 'Ranch', 0.00);

INSERT IGNORE INTO product_modifier_groups (product_id, group_id) VALUES
(101, 1),
(101, 2),
(102, 1),
(102, 2);
