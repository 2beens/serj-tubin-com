CREATE TABLE public.blog
(
    id         SERIAL PRIMARY KEY,
    title      VARCHAR NOT NULL,
    created_at TIMESTAMP WITHOUT TIME ZONE NOT NULL,
    content    TEXT    NOT NULL,
    claps      INTEGER NOT NULL DEFAULT 0
);

ALTER TABLE public.blog OWNER TO postgres;
CREATE INDEX ix_blog_created_at ON public.blog USING btree (created_at);

-- NETLOG DB SETUP
CREATE SCHEMA netlog;
CREATE TABLE netlog.visit
(
    id        SERIAL PRIMARY KEY,
    title     VARCHAR,
    source    VARCHAR,
    device    VARCHAR,
    url       VARCHAR     NOT NULL,
    timestamp TIMESTAMPTZ NOT NULL
);

ALTER TABLE netlog.visit OWNER TO postgres;
CREATE INDEX ix_visit_created_at ON netlog.visit USING btree (timestamp);
CREATE INDEX ix_visit_url ON netlog.visit (url);

CREATE TABLE public.note
(
    id         SERIAL PRIMARY KEY,
    title      VARCHAR,
    created_at TIMESTAMP WITHOUT TIME ZONE NOT NULL,
    content    TEXT NOT NULL
);

ALTER TABLE public.note OWNER TO postgres;

-- Create exercise table, which will store the exercises (sets) performed
CREATE TABLE public.exercise
(
    id           SERIAL PRIMARY KEY,
    exercise_id  VARCHAR NOT NULL,
    muscle_group VARCHAR NOT NULL,
    kilos        INTEGER NOT NULL,
    reps         INTEGER NOT NULL,
    metadata     JSONB NOT NULL DEFAULT '{}',
    created_at   TIMESTAMP WITHOUT TIME ZONE NOT NULL

    -- TODO: add fk constraint to exercise_type table after DB cleanup
    -- CONSTRAINT fk_exercise_type FOREIGN KEY (exercise_id) REFERENCES public.exercise_type (id)
);

ALTER TABLE public.exercise OWNER TO postgres;
CREATE INDEX ix_exercise_created_at ON public.exercise (created_at);
CREATE INDEX ix_exercise_muscle_group ON public.exercise (muscle_group);
CREATE INDEX ix_exercise_exercise_id ON public.exercise (exercise_id);

-- Create exercise_type table
CREATE TABLE public.exercise_type
(
    id VARCHAR PRIMARY KEY, -- example: deadlift, bench_press, etc.
    muscle_group VARCHAR NOT NULL,
    name VARCHAR NOT NULL,  -- example: "Deadlift", "Bench Press", etc.
    description TEXT,
    created_at TIMESTAMP WITHOUT TIME ZONE NOT NULL
);

-- Assign ownership and create an index on the created_at column
ALTER TABLE public.exercise_type OWNER TO postgres;
CREATE INDEX ix_exercise_type_muscle_group ON public.exercise_type (muscle_group);

-- Create exercise_image table
CREATE TABLE public.exercise_image
(
    id SERIAL PRIMARY KEY,
    exercise_id VARCHAR NOT NULL,
    image_path TEXT NOT NULL,
    created_at TIMESTAMP WITHOUT TIME ZONE NOT NULL,
    CONSTRAINT fk_exercise FOREIGN KEY (exercise_id) REFERENCES public.exercise_type (id)
);

ALTER TABLE public.exercise_image OWNER TO postgres;
CREATE INDEX ix_exercise_image_exercise_id ON public.exercise_image (exercise_id);

CREATE TABLE public.gymstats_event
(
    id         SERIAL PRIMARY KEY,
    type       VARCHAR NOT NULL,
    data       JSONB NOT NULL DEFAULT '{}',
    timestamp  TIMESTAMP WITHOUT TIME ZONE NOT NULL
);

ALTER TABLE public.gymstats_event OWNER TO postgres;
CREATE INDEX ix_gymstats_event_created_at ON public.gymstats_event (timestamp);

CREATE TABLE public.visitor_board_message
(
    id         SERIAL PRIMARY KEY,
    author     VARCHAR,
    message    VARCHAR NOT NULL,
    created_at TIMESTAMP WITHOUT TIME ZONE NOT NULL
);

ALTER TABLE public.visitor_board_message OWNER TO postgres;
CREATE INDEX ix_visitor_board_message_created_at ON public.visitor_board_message (created_at);
