-- TakoduDev2026!

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
