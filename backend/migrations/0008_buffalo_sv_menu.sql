-- Menú adicional de Buffalo Wings El Salvador
-- Ejecutar después de 0007_catalog_expansion.sql. Las inserciones son idempotentes

-- Grupo para los sándwiches disponibles en la carta.
INSERT IGNORE INTO categories (id, name, station, sort_order) VALUES
(7, 'Sándwiches', 'KITCHEN', 7);

-- Las órdenes de alitas y boneless incluyen selección obligatoria de salsa y aderezo.
-- 10 Boneless Wings (id 103) ya existía en 0001 y tampoco tenía estas relaciones.
INSERT IGNORE INTO product_modifier_groups (product_id, group_id)
SELECT p.id, mg.id
FROM products p
JOIN modifier_groups mg
  ON (mg.id = 1 AND mg.name = 'Salsa Obligatoria')
  OR (mg.id = 2 AND mg.name = 'Aderezo Obligatorio')
WHERE p.id IN (103, 107, 108, 109, 110, 111, 112);

-- Selección corta de salsas clásicas y niveles picantes.
-- NOT EXISTS evita repetir opciones si se vuelve a ejecutar la migración.
INSERT INTO modifier_options (group_id, name, extra_price)
SELECT 1, choices.name, 0.00
FROM (
    SELECT 'Mild' AS name
    UNION ALL SELECT 'Nuclear'
    UNION ALL SELECT 'La Suicida'
    UNION ALL SELECT 'Teriyaki'
    UNION ALL SELECT 'Garlic Parmesan'
    UNION ALL SELECT 'Lemon Pepper'
    UNION ALL SELECT 'Honey BBQ'
) AS choices
WHERE NOT EXISTS (
    SELECT 1 FROM modifier_options mo
    WHERE mo.group_id = 1 AND mo.name = choices.name
);

-- Aderezos para acompañar alas, chunks, boneless y entradas.
INSERT INTO modifier_options (group_id, name, extra_price)
SELECT 2, choices.name, 0.00
FROM (
    SELECT 'Ranch' AS name
    UNION ALL SELECT 'Blue Cheese'
    UNION ALL SELECT 'Honey Mustard'
) AS choices
WHERE NOT EXISTS (
    SELECT 1 FROM modifier_options mo
    WHERE mo.group_id = 2 AND mo.name = choices.name
);

-- Platos, bebidas y postres de la carta. Los precios están expresados en USD.
INSERT IGNORE INTO products (id, category_id, name, base_price) VALUES
(127, 4, 'Mozzarella Crunch', 5.95),
(128, 4, 'Jalapeño Poppers', 5.95),
(129, 4, 'Nachos Buffalo', 5.95),
(130, 4, 'Dedos de Queso (Rollers)', 4.99),
(131, 4, 'Buffalo’s Chili Cheese Dip', 5.95),
(132, 4, 'Nachos Monstruosos', 8.99),
(133, 7, 'Buffalo Chicken Sandwich', 8.99),
(134, 7, 'BW Sandwich', 8.99),
(135, 5, 'Naranjada', 2.75),
(136, 5, 'Limonada con Hierbabuena', 2.95),
(137, 5, 'Limonada con Fresa', 2.95),
(138, 5, 'Té Helado con Refill', 2.95),
(139, 5, 'Frozen de Temporada', 3.50),
(140, 6, 'El Único', 4.95),
(141, 6, 'El Otro (Brownie BW)', 4.50),
(142, 6, 'Strawberry Cheesecake', 5.99);
