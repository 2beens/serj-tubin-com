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
    -- because there are exercises added from before, with invalid exercise_id
    -- CONSTRAINT fk_exercise_type FOREIGN KEY (exercise_id) REFERENCES public.exercise_type (id)
);

ALTER TABLE public.exercise OWNER TO postgres;
CREATE INDEX ix_exercise_created_at ON public.exercise (created_at);
CREATE INDEX ix_exercise_muscle_group ON public.exercise (muscle_group);
CREATE INDEX ix_exercise_exercise_id ON public.exercise (exercise_id);

-- Create exercise_type table
CREATE TABLE public.exercise_type
(
    id           VARCHAR PRIMARY KEY, -- example: deadlift, bench_press, etc.
    muscle_group VARCHAR NOT NULL,
    name         VARCHAR NOT NULL,  -- example: "Deadlift", "Bench Press", etc.
    description  TEXT,
    created_at   TIMESTAMP WITHOUT TIME ZONE NOT NULL
);

-- Assign ownership and create an index on the created_at column
ALTER TABLE public.exercise_type OWNER TO postgres;
CREATE INDEX ix_exercise_type_muscle_group ON public.exercise_type (muscle_group);

-- Create exercise_image table
CREATE TABLE public.exercise_image
(
    -- id is an int64 representation of a diskApi file id, and is a primary key for the table
    id          BIGINT PRIMARY KEY,
    exercise_id VARCHAR NOT NULL,
    created_at  TIMESTAMP WITHOUT TIME ZONE NOT NULL,
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

-- add basic exercise types from the javascript code from the web client:
---------------------------------------------------------------------------
-- biceps exercises
INSERT INTO public.exercise_type (id, muscle_group, name, created_at)
VALUES
    ('preacher_curl', 'biceps', 'Preacher Curl', NOW()),
    ('barbell_curl', 'biceps', 'Barbell Curl', NOW()),
    ('barbell_vertical_curl', 'biceps', 'Barbell Curl [vertical grip]', NOW()),
    ('ez_bar_curl', 'biceps', 'EZ Bar Curl', NOW()),
    ('ez_bar_curl_declined', 'biceps', 'EZ Bar Curl [declined]', NOW()),
    ('dumbells', 'biceps', 'Dumbells', NOW()),
    ('dumbells_inclined', 'biceps', 'Dumbells [inclined]', NOW()),
    ('dumbells_declined', 'biceps', 'Dumbells [declined]', NOW());

-- triceps exercises
INSERT INTO public.exercise_type (id, muscle_group, name, created_at)
VALUES
    ('skullcrusher_with_ez_bar', 'triceps', 'Skullcrusher w EZ 💀', NOW()),
    ('triceps_pushdown', 'triceps', 'Tricep Pushdown', NOW()),
    ('skull_crushers', 'triceps', 'Skull Crushers', NOW()),
    ('close_grip_bench_press', 'triceps', 'Close-Grip Bench Press', NOW()),
    ('bench_dip', 'triceps', 'Bench Dip', NOW()),
    ('dumbbell_overhead_triceps_extension', 'triceps', 'Dumbbell Overhead Triceps Extension', NOW()),
    ('cable_overhead_extension_with_rope', 'triceps', 'Cable Overhead Extension With Rope', NOW()),
    ('cable_push_down', 'triceps', 'Cable Push-Down', NOW()),
    ('barbell_push_down', 'triceps', 'Barbell Push-Down', NOW()),
    ('barbell_push_down_ez', 'triceps', 'Barbell EZ Push-Down', NOW());

-- chest exercises
INSERT INTO public.exercise_type (id, muscle_group, name, created_at)
VALUES
    ('bench_press', 'chest', 'Bench Press', NOW()),
    ('bench_press_inclined', 'chest', 'Bench Press [inclined]', NOW()),
    ('bench_press_declined', 'chest', 'Bench Press [declined]', NOW()),
    ('dips', 'chest', 'Dips', NOW()),
    ('chest_fly_machine', 'chest', 'Chest Fly [machine]', NOW()),
    ('chest_fly_machine_inclined', 'chest', 'Chest Fly [machine][inclined]', NOW()),
    ('chest_press_machine', 'chest', 'Chest Press [machine]', NOW()),
    ('pushups', 'chest', 'Push-Ups', NOW()),
    ('dumbbell_pull_over', 'chest', 'Dumbbell Pull-Over', NOW());

-- legs exercises
INSERT INTO public.exercise_type (id, muscle_group, name, created_at)
VALUES
    ('leg_press', 'legs', 'Leg Press', NOW()),
    ('calf_raise_seated', 'legs', 'Calf Raise [seated]', NOW()),
    ('calf_raise_standing', 'legs', 'Calf Raise [standing]', NOW()),
    ('walking_dumbbell_lunges', 'legs', 'Walking Dumbbell Lunges', NOW()),
    ('leg_extension_machine', 'legs', 'Leg Extension Machine', NOW()),
    ('leg_curl_machine', 'legs', 'Leg Curl Machine', NOW());

-- shoulders exercises
INSERT INTO public.exercise_type (id, muscle_group, name, created_at)
VALUES
    ('lateral_raise', 'shoulders', 'Lateral Raise 👐', NOW()),
    ('side_lateral_raise', 'shoulders', 'Side Lateral Raise 👐', NOW()),
    ('lateral_raise_single_arm_cable', 'shoulders', 'Lateral Raise 👐 [single] [cable]', NOW()),
    ('front_raise', 'shoulders', 'Front Raise ➬', NOW()),
    ('front_raise_single_arm_cable', 'shoulders', 'Front Raise 👐 [single] [cable]', NOW()),
    ('back_raise', 'shoulders', 'Back Raise 🔙', NOW()),
    ('back_push_machine', 'shoulders', 'Back Push 🔙 [machine]', NOW()),
    ('press_barbell_standing', 'shoulders', 'Press [barbell] [standing]', NOW()),
    ('press_dumbell_standing', 'shoulders', 'Press [dumbell] [standing]', NOW()),
    ('press_barbell_seated', 'shoulders', 'Press [barbell] [seated]', NOW()),
    ('press_dumbell_seated', 'shoulders', 'Press [dumbell] [seated]', NOW()),
    ('arnold_press', 'shoulders', 'Arnold Press', NOW()),
    ('rear_delt_fly', 'shoulders', 'Rear Delt Fly', NOW());

-- back exercises
INSERT INTO public.exercise_type (id, muscle_group, name, created_at)
VALUES
    ('dumbbell_row_inclined', 'back', 'Inclined Dumbbell Row', NOW()),
    ('barbell_row_inclined', 'back', 'Inclined Barbell Row', NOW()),
    ('single_arm_dumbell_row', 'back', 'Single-Arm Dumbell Row', NOW()),
    ('bent_over_row', 'back', 'Bent Over Row', NOW()),
    ('t_bar_row', 'back', 'T-Bar Row', NOW()),
    ('pull_up', 'back', 'Pull Up', NOW()),
    ('seated_row_barbell', 'back', 'Seated Row [barbell]', NOW()),
    ('seated_row_v_handle', 'back', 'Seated Row [V handle]', NOW()),
    ('hyperextensions', 'back', 'Hyperextensions', NOW()),
    ('lat_pull_down_barbell', 'back', 'Lat Pull-Down [barbell]', NOW()),
    ('lat_pull_down_v_handle', 'back', 'Lat Pull-Down [V handle]', NOW());

-- other exercises
INSERT INTO public.exercise_type (id, muscle_group, name, created_at)
VALUES
    ('oblique_crunch_hyperext_bench', 'other', 'Oblique Crunch [Hyperextension Bench]', NOW()),
    ('crunch', 'other', 'Crunch', NOW()),
    ('tuck_crunch', 'other', 'Tuck Crunch', NOW()),
    ('leg_raise_core', 'other', 'Leg Raise [core]', NOW()),
    ('squat', 'other', 'Squat', NOW()),
    ('squat_jump', 'other', 'Squat Jump', NOW()),
    ('burpee', 'other', 'Burpee', NOW()),
    ('plank', 'other', 'Plank', NOW()),
    ('hanging_knee_raise', 'other', 'Hanging Knee Raise', NOW()),
    ('v_sit', 'other', 'V-sit', NOW()),
    ('test', 'other', '🛠️Test🛠️[dummy]🛠️', NOW());
