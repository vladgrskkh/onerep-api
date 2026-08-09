-- +goose Up
CREATE SCHEMA IF NOT EXISTS gym;

CREATE TABLE gym.muscle_groups (
    id SERIAL PRIMARY KEY,
    name TEXT NOT NULL UNIQUE
);

INSERT INTO gym.muscle_groups (name) VALUES
('chest'), ('quads'), ('biceps'), ('triceps'), ('shoulders'),
('back'), ('glutes'), ('hamstrings'), ('calves'), ('abs'),
('forearms'), ('traps'), ('lats'), ('adductors'), ('obliques');

CREATE TABLE gym.exercises (
    id UUID PRIMARY KEY,
    name TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    notes TEXT NOT NULL DEFAULT '',
    is_built_in BOOLEAN NOT NULL DEFAULT false,
    created_by_user_id UUID,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at TIMESTAMPTZ,
    version INTEGER NOT NULL DEFAULT 1
);

CREATE TABLE gym.exercise_media (
    id UUID PRIMARY KEY,
    exercise_id UUID NOT NULL REFERENCES gym.exercises(id) ON DELETE CASCADE,
    media_type TEXT NOT NULL,
    sort_order INTEGER NOT NULL DEFAULT 0,
    s3_key TEXT NOT NULL
);

CREATE TABLE gym.exercise_muscle_groups (
    exercise_id UUID NOT NULL REFERENCES gym.exercises(id) ON DELETE CASCADE,
    muscle_group_id INTEGER NOT NULL REFERENCES gym.muscle_groups(id),
    is_primary BOOLEAN NOT NULL DEFAULT false,
    PRIMARY KEY (exercise_id, muscle_group_id)
);

CREATE TABLE gym.templates (
    id UUID PRIMARY KEY,
    name TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    is_public BOOLEAN NOT NULL DEFAULT false,
    created_by_user_id UUID NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at TIMESTAMPTZ,
    version INTEGER NOT NULL DEFAULT 1
);

CREATE TABLE gym.template_media (
    id UUID PRIMARY KEY,
    template_id UUID NOT NULL REFERENCES gym.templates(id) ON DELETE CASCADE,
    media_type TEXT NOT NULL DEFAULT 'photo',
    sort_order INTEGER NOT NULL DEFAULT 0,
    s3_key TEXT NOT NULL
);

CREATE TABLE gym.template_exercises (
    template_id UUID NOT NULL REFERENCES gym.templates(id) ON DELETE CASCADE,
    exercise_id UUID NOT NULL REFERENCES gym.exercises(id),
    sort_order INTEGER NOT NULL DEFAULT 0,
    planned_sets INTEGER NOT NULL DEFAULT 3,
    PRIMARY KEY (template_id, exercise_id)
);

CREATE TABLE gym.workouts (
    id UUID PRIMARY KEY,
    user_id UUID NOT NULL,
    template_id UUID,
    started_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    finished_at TIMESTAMPTZ,
    notes TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at TIMESTAMPTZ,
    version INTEGER NOT NULL DEFAULT 1
);

CREATE TABLE gym.workout_exercises (
    id UUID PRIMARY KEY,
    workout_id UUID NOT NULL REFERENCES gym.workouts(id) ON DELETE CASCADE,
    exercise_id UUID NOT NULL REFERENCES gym.exercises(id),
    sort_order INTEGER NOT NULL DEFAULT 0,
    notes TEXT NOT NULL DEFAULT ''
);

CREATE TABLE gym.workout_sets (
    id UUID PRIMARY KEY,
    workout_exercise_id UUID NOT NULL REFERENCES gym.workout_exercises(id) ON DELETE CASCADE,
    set_number INTEGER NOT NULL,
    weight_kg NUMERIC(8,2) NOT NULL,
    reps INTEGER NOT NULL,
    rpe SMALLINT,
    rest_seconds INTEGER,
    is_warmup BOOLEAN NOT NULL DEFAULT false
);

CREATE TABLE gym.body_weights (
    id UUID PRIMARY KEY,
    user_id UUID NOT NULL,
    weight_kg NUMERIC(5,1) NOT NULL,
    measured_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    version INTEGER NOT NULL DEFAULT 1
);

CREATE TABLE gym.progress_1rm (
    exercise_id UUID NOT NULL REFERENCES gym.exercises(id),
    user_id UUID NOT NULL,
    date DATE NOT NULL,
    estimated_1rm NUMERIC(8,2) NOT NULL,
    PRIMARY KEY (exercise_id, user_id, date)
);

CREATE TABLE gym.progress_volume (
    muscle_group_id INTEGER NOT NULL REFERENCES gym.muscle_groups(id),
    user_id UUID NOT NULL,
    date DATE NOT NULL,
    total_kg NUMERIC(10,2) NOT NULL,
    PRIMARY KEY (muscle_group_id, user_id, date)
);

CREATE INDEX idx_exercises_deleted_at ON gym.exercises (deleted_at);
CREATE INDEX idx_templates_deleted_at ON gym.templates (deleted_at);
CREATE INDEX idx_workouts_user_id ON gym.workouts (user_id);
CREATE INDEX idx_workouts_deleted_at ON gym.workouts (deleted_at);
CREATE INDEX idx_body_weights_user_id ON gym.body_weights (user_id);

-- +goose Down
DROP SCHEMA IF EXISTS gym CASCADE;
