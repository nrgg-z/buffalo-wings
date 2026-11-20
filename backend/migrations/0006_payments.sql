-- 0006_payments.sql: pagos, recibos y DTE simulado

CREATE TABLE IF NOT EXISTS payments (
    id INT UNSIGNED NOT NULL AUTO_INCREMENT,
    order_id INT UNSIGNED NOT NULL,
    cashier_id INT UNSIGNED NOT NULL,
    payment_method ENUM('CASH', 'CARD', 'QR') NOT NULL,
    subtotal DECIMAL(10,2) NOT NULL,
    tax_amount DECIMAL(10,2) NOT NULL,
    tip_amount DECIMAL(10,2) NOT NULL DEFAULT 0.00,
    total_amount DECIMAL(10,2) NOT NULL,
    amount_received DECIMAL(10,2) NOT NULL,
    change_amount DECIMAL(10,2) NOT NULL DEFAULT 0.00,
    dte_code VARCHAR(64) NOT NULL,
    reference VARCHAR(100) NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (id),
    UNIQUE KEY uq_payments_order (order_id),
    KEY idx_payments_created_at (created_at),
    CONSTRAINT fk_payments_order FOREIGN KEY (order_id) REFERENCES orders(id),
    CONSTRAINT fk_payments_cashier FOREIGN KEY (cashier_id) REFERENCES users(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;