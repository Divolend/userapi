CREATE TABLE users(
    id SERIAL PRIMARY KEY,
    name varchar NOT NULL UNIQUE,
    age int NOT NULL CHECK (age > 0)
);