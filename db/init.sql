CREATE TABLE IF NOT EXISTS updates (
    id SERIAL PRIMARY KEY,
    date DATE NOT NULL,
    title TEXT NOT NULL,
    description TEXT NOT NULL
);