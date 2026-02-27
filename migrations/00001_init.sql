-- +goose Up
-- +goose StatementBegin
CREATE TABLE departments (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(200) NOT NULL CHECK (char_length(btrim(name)) BETWEEN 1 AND 200),
    parent_id BIGINT REFERENCES departments(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX uniq_departments_parent_name
    ON departments ((COALESCE(parent_id, 0)), LOWER(name));

CREATE TABLE employees (
    id BIGSERIAL PRIMARY KEY,
    department_id BIGINT NOT NULL REFERENCES departments(id) ON DELETE CASCADE,
    full_name VARCHAR(200) NOT NULL CHECK (char_length(btrim(full_name)) BETWEEN 1 AND 200),
    position VARCHAR(200) NOT NULL CHECK (char_length(btrim(position)) BETWEEN 1 AND 200),
    hired_at DATE NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_employees_department_id ON employees (department_id);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS employees;
DROP TABLE IF EXISTS departments;
-- +goose StatementEnd
