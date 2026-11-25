INSERT INTO catalog_categories (id, name, sort_order)
SELECT
    gen_random_uuid(),
    'Cortes y Barbería',
    1
WHERE NOT EXISTS (
    SELECT 1 FROM catalog_categories WHERE sort_order = 1
);

INSERT INTO catalog_services (id, category_id, name, duration_minutes, price_minor_units, sort_order)
SELECT
    gen_random_uuid(),
    c.id,
    s.name,
    s.duration_minutes,
    s.price_minor_units,
    s.sort_order
FROM (
    SELECT id FROM catalog_categories ORDER BY sort_order ASC LIMIT 1
) c
CROSS JOIN (
    VALUES
        ('Corte clásico', 30, 3000::bigint, 1),
        ('Corte moderno', 45, 4000::bigint, 2),
        ('Barba y perfilado', 30, 2500::bigint, 3)
) AS s(name, duration_minutes, price_minor_units, sort_order)
WHERE NOT EXISTS (
    SELECT 1 FROM catalog_services existing
    WHERE existing.category_id = c.id AND existing.sort_order = s.sort_order
);
