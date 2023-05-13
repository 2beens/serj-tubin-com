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

CREATE TABLE public.exercise
(
    id           SERIAL PRIMARY KEY,
    exercise_id  VARCHAR NOT NULL,
    muscle_group VARCHAR NOT NULL,
    kilos        INTEGER NOT NULL,
    reps         INTEGER NOT NULL,
    metadata     JSONB NOT NULL DEFAULT '{}',
    created_at   TIMESTAMP WITHOUT TIME ZONE NOT NULL
);

ALTER TABLE public.exercise OWNER TO postgres;
CREATE INDEX ix_exercise_created_at ON public.exercise (created_at);

CREATE TABLE public.visitor_board_message
(
    id         SERIAL PRIMARY KEY,
    author     VARCHAR,
    message    VARCHAR NOT NULL,
    created_at TIMESTAMP WITHOUT TIME ZONE NOT NULL
);

ALTER TABLE public.visitor_board_message OWNER TO postgres;
CREATE INDEX ix_visitor_board_message_created_at ON public.visitor_board_message (created_at);
