-- 0005_orders.sql: tablas de comandas, ítems (con estado para KDS) y modificadores

CREATE TABLE IF NOT EXISTS orders (
    id INT UNSIGNED NOT NULL AUTO_INCREMENT,
    table_id INT UNSIGNED NOT NULL,
    waiter_id INT UNSIGNED NOT NULL,
    status ENUM('OPEN', 'PAID', 'CANCELLED') NOT NULL DEFAULT 'OPEN',
    subtotal DECIMAL(10,2) NOT NULL DEFAULT 0.00,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    closed_at DATETIME NULL,
    PRIMARY KEY (id),
    KEY idx_orders_table_status (table_id, status),
    CONSTRAINT fk_orders_table FOREIGN KEY (table_id) REFERENCES dining_tables(id),
    CONSTRAINT fk_orders_waiter FOREIGN KEY (waiter_id) REFERENCES users(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS order_items (
    id INT UNSIGNED NOT NULL AUTO_INCREMENT,
    order_id INT UNSIGNED NOT NULL,
    product_id INT UNSIGNED NOT NULL,
    product_name VARCHAR(100) NOT NULL,
    station ENUM('KITCHEN', 'BAR') NOT NULL DEFAULT 'KITCHEN',
    quantity INT UNSIGNED NOT NULL DEFAULT 1,
    unit_price DECIMAL(10,2) NOT NULL,
    notes VARCHAR(255) NULL,
    status ENUM('PENDING', 'IN_PROGRESS', 'READY', 'DELIVERED', 'VOIDED') NOT NULL DEFAULT 'PENDING',
    void_reason VARCHAR(255) NULL,
    voided_by INT UNSIGNED NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    ready_at DATETIME NULL,
    PRIMARY KEY (id),
    KEY idx_order_items_order (order_id),
    KEY idx_order_items_kds (status, station, created_at),
    CONSTRAINT fk_order_items_order FOREIGN KEY (order_id) REFERENCES orders(id) ON DELETE CASCADE,
    CONSTRAINT fk_order_items_product FOREIGN KEY (product_id) REFERENCES products(id),
    CONSTRAINT fk_order_items_voided_by FOREIGN KEY (voided_by) REFERENCES users(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS order_item_modifiers (
    id INT UNSIGNED NOT NULL AUTO_INCREMENT,
    order_item_id INT UNSIGNED NOT NULL,
    modifier_option_id INT UNSIGNED NOT NULL,
    modifier_name VARCHAR(100) NOT NULL,
    extra_price DECIMAL(10,2) NOT NULL DEFAULT 0.00,
    PRIMARY KEY (id),
    KEY idx_oim_item (order_item_id),
    CONSTRAINT fk_oim_item FOREIGN KEY (order_item_id) REFERENCES order_items(id) ON DELETE CASCADE,
    CONSTRAINT fk_oim_option FOREIGN KEY (modifier_option_id) REFERENCES modifier_options(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;