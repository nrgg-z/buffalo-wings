-- 0004: combinación de mesas. Una mesa combinada apunta a la mesa principal.
ALTER TABLE dining_tables
    ADD COLUMN merged_into_id INT UNSIGNED NULL,
    ADD CONSTRAINT fk_dining_tables_merged_into
        FOREIGN KEY (merged_into_id) REFERENCES dining_tables(id);
