# TakoduDev2026!

INSERT INTO iam_roles (id, name, position, permissions)
VALUES
    (
        gen_random_uuid(),
        'Coordinación',
        1,
        ARRAY['calendar.manage', 'crm.manage', 'catalog.manage', 'organization.manage', 'iam.manage_users', 'iam.manage_roles']
    ),
    (
        gen_random_uuid(),
        'Recepción',
        2,
        ARRAY['calendar.manage', 'crm.manage', 'catalog.manage', 'organization.manage']
    ),
    (
        gen_random_uuid(),
        'Equipo de servicio',
        3,
        ARRAY['calendar.manage', 'scheduling.appear']
    )
ON CONFLICT (name) DO NOTHING;

INSERT INTO iam_users (id, email, first_name, last_name, password_hash, type)
VALUES
    (
        gen_random_uuid(),
        'lucia@estudionube.mx',
        'Lucía',
        'Méndez',
        '$argon2id$v=19$m=65536,t=3,p=2$T6oEB+ocOzQTi9xc1P/tnQ$smVLRKqsT2o9btPb8keTwFaMNWvWzt9iiPeUj0/iFeU',
        'owner'
    ),
    (
        gen_random_uuid(),
        'carlos@estudionube.mx',
        'Carlos',
        'Rivera',
        '$argon2id$v=19$m=65536,t=3,p=2$T6oEB+ocOzQTi9xc1P/tnQ$smVLRKqsT2o9btPb8keTwFaMNWvWzt9iiPeUj0/iFeU',
        'user'
    ),
    (
        gen_random_uuid(),
        'sofia@estudionube.mx',
        'Sofía',
        'Navarro',
        '$argon2id$v=19$m=65536,t=3,p=2$T6oEB+ocOzQTi9xc1P/tnQ$smVLRKqsT2o9btPb8keTwFaMNWvWzt9iiPeUj0/iFeU',
        'user'
    ),
    (
        gen_random_uuid(),
        'mateo@estudionube.mx',
        'Mateo',
        'Paredes',
        '$argon2id$v=19$m=65536,t=3,p=2$T6oEB+ocOzQTi9xc1P/tnQ$smVLRKqsT2o9btPb8keTwFaMNWvWzt9iiPeUj0/iFeU',
        'user'
    ),
    (
        gen_random_uuid(),
        'ana@estudionube.mx',
        'Ana',
        'Torres',
        '$argon2id$v=19$m=65536,t=3,p=2$T6oEB+ocOzQTi9xc1P/tnQ$smVLRKqsT2o9btPb8keTwFaMNWvWzt9iiPeUj0/iFeU',
        'user'
    )
ON CONFLICT (email) DO NOTHING;

INSERT INTO iam_member_roles (user_id, role_id)
SELECT u.id, r.id
FROM (VALUES
    ('carlos@estudionube.mx', 'Coordinación'),
    ('sofia@estudionube.mx', 'Recepción'),
    ('mateo@estudionube.mx', 'Equipo de servicio')
) AS m(email, role_name)
JOIN iam_users u ON u.email = m.email
JOIN iam_roles r ON r.name = m.role_name
ON CONFLICT (user_id, role_id) DO NOTHING;
