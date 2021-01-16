CREATE TABLE public.blog
(
    id SERIAL PRIMARY KEY,
    title VARCHAR NOT NULL,
    created_at TIMESTAMP WITHOUT TIME ZONE NOT NULL,
    content TEXT NOT NULL
);

ALTER TABLE public.blog OWNER TO postgres;
CREATE INDEX ix_blog_created_at ON public.blog USING btree (created_at);

-- NETLOG DB SETUP
CREATE SCHEMA netlog;
CREATE TABLE netlog.visit
(
    id SERIAL PRIMARY KEY,
    title VARCHAR,
    url VARCHAR NOT NULL,
    timestamp TIMESTAMP WITHOUT TIME ZONE NOT NULL
);

ALTER TABLE netlog.visit OWNER TO postgres;
CREATE INDEX ix_visit_created_at ON netlog.visit USING btree (timestamp);
