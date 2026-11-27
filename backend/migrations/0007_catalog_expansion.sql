-- Amplía el menú inicial manteniendo las categorías existentes y agrega Postres.
INSERT IGNORE INTO categories (id, name, station, sort_order) VALUES
(6, 'Postres', 'KITCHEN', 6);

INSERT IGNORE INTO products (id, category_id, name, base_price) VALUES
(107, 1, '6 Traditional Wings', 8.50),
(108, 1, '18 Traditional Wings', 20.99),
(109, 1, '30 Traditional Wings', 32.99),
(110, 2, '6 Boneless Wings', 7.99),
(111, 2, '15 Boneless Wings', 16.99),
(112, 2, '20 Boneless Wings', 21.99),
(113, 3, 'Classic Burger', 7.50),
(114, 3, 'Bacon Cheddar Burger', 10.50),
(115, 3, 'BBQ Ranch Burger', 9.99),
(116, 4, 'Papas Twister Pequeñas', 2.99),
(117, 4, 'Aros de Cebolla', 4.50),
(118, 4, 'Ensalada César', 4.75),
(119, 4, 'Papas Fritas', 3.25),
(120, 5, 'Gaseosa', 1.75),
(121, 5, 'Limonada', 2.25),
(122, 5, 'Té Helado', 2.00),
(123, 5, 'Agua Embotellada', 1.50),
(124, 6, 'Brownie con Helado', 5.25),
(125, 6, 'Cheesecake', 4.75),
(126, 6, 'Churros con Chocolate', 4.50);
