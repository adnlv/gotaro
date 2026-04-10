-- +goose Up
CREATE TABLE projects (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (user_id, name)
);

CREATE INDEX projects_user_id_idx ON projects (user_id);

ALTER TABLE tasks ADD COLUMN project_id BIGINT REFERENCES projects (id) ON DELETE SET NULL;
CREATE INDEX tasks_project_id_idx ON tasks (project_id);

-- +goose Down
ALTER TABLE tasks DROP COLUMN IF EXISTS project_id;
DROP TABLE IF EXISTS projects;
