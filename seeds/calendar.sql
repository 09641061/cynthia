INSERT INTO calendar_people (id, name, active)
SELECT gen_random_uuid(), v.name, TRUE
FROM (VALUES
    ('Lucía Méndez'),
    ('Elena Cruz')
) AS v(name)
WHERE NOT EXISTS (
    SELECT 1 FROM calendar_people p WHERE p.name = v.name
);
