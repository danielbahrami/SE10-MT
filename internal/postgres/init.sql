CREATE TABLE organizations (
    id SERIAL PRIMARY KEY,
    name VARCHAR(50) NOT NULL,
    default_permissions JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    org_id INT NOT NULL REFERENCES organizations(id),
    name VARCHAR(50) NOT NULL,
    hashed_token TEXT NOT NULL,
    override_permissions JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE logs (
    id SERIAL PRIMARY KEY,
    user_id INT NOT NULL REFERENCES users(id),
    query TEXT NOT NULL,
    decision VARCHAR(20) NOT NULL CHECK (decision IN ('Allowed', 'Blocked', 'Rewritten')),
    rewritten_query TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
