package gymstats

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/2beens/serjtubincom/internal/telemetry/tracing"
	log "github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrExerciseNotFound = errors.New("exercise not found")

type ListParams struct {
	ExerciseID  *string
	MuscleGroup *string
	Limit       int
}

type Exercise struct {
	ID          int               `json:"id"`
	ExerciseID  string            `json:"exerciseId"`
	MuscleGroup string            `json:"muscleGroup"`
	Kilos       int               `json:"kilos"`
	Reps        int               `json:"reps"`
	CreatedAt   time.Time         `json:"createdAt"`
	Metadata    map[string]string `json:"metadata"`
}

type Repo struct {
	db *pgxpool.Pool
}

func NewRepo(db *pgxpool.Pool) *Repo {
	return &Repo{
		db: db,
	}
}

func (r *Repo) Add(ctx context.Context, exercise *Exercise) (*Exercise, error) {
	metadataJson, err := json.Marshal(exercise.Metadata)
	if err != nil {
		return nil, fmt.Errorf("marshal metadata: %w", err)
	}

	rows, err := r.db.Query(
		ctx,
		`INSERT INTO exercise 
				(exercise_id, muscle_group, kilos, reps, metadata, created_at)
				VALUES ($1, $2, $3, $4, $5, $6)
			RETURNING id;`,
		exercise.ExerciseID, exercise.MuscleGroup, exercise.Kilos, exercise.Reps, metadataJson, exercise.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	if err := rows.Err(); err != nil {
		return nil, err
	}

	if !rows.Next() {
		return nil, errors.New("unexpected error [no rows next]")
	}

	var id int
	if err := rows.Scan(&id); err != nil {
		return nil, fmt.Errorf("rows scan: %w", err)
	}

	exercise.ID = id
	return exercise, nil
}

func (r *Repo) Update(ctx context.Context, exercise *Exercise) error {
	tag, err := r.db.Exec(
		ctx,
		`UPDATE exercise SET exercise_id = $1, muscle_group = $2, kilos = $3, reps = $4, metadata = $5, created_at = $6 WHERE id = $7;`,
		exercise.ExerciseID, exercise.MuscleGroup, exercise.Kilos, exercise.Reps, exercise.Metadata, exercise.CreatedAt, exercise.ID,
	)
	if err != nil {
		return err
	}

	if tag.RowsAffected() == 0 {
		return ErrExerciseNotFound
	}

	return nil
}

func (r *Repo) Delete(ctx context.Context, id int) error {
	tag, err := r.db.Exec(
		ctx,
		`DELETE FROM exercise WHERE id = $1`,
		id,
	)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrExerciseNotFound
	}
	return nil
}

func (r *Repo) Get(ctx context.Context, id int) (*Exercise, error) {
	rows, err := r.db.Query(
		ctx,
		`
			SELECT
				id, exercise_id, muscle_group, kilos, reps, metadata, created_at
			FROM exercise
			WHERE id = $1;`,
		id,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	if err := rows.Err(); err != nil {
		return nil, err
	}

	exercises, err := r.rows2exercises(rows)
	if err != nil {
		return nil, err
	}

	if len(exercises) != 1 {
		return nil, ErrExerciseNotFound
	}

	return &exercises[0], nil
}

func (r *Repo) List(ctx context.Context, params ListParams) ([]Exercise, error) {
	rows, err := r.db.Query(
		ctx,
		`
			SELECT
				id, exercise_id, muscle_group, kilos, reps, metadata, created_at
			FROM exercise
				WHERE ($1::text IS NULL OR exercise_id = $1)
				AND ($2::text IS NULL OR muscle_group = $2)
			ORDER BY created_at DESC
			LIMIT $3;`,
		params.ExerciseID, params.MuscleGroup, params.Limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return r.rows2exercises(rows)
}

func (r *Repo) ListPage(ctx context.Context, page, size int) (_ []Exercise, total int, err error) {
	ctx, span := tracing.GlobalTracer.Start(ctx, "repo.gymstats.page")
	span.SetAttributes(attribute.Int("page", page))
	span.SetAttributes(attribute.Int("size", size))
	defer func() {
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
		}
		span.End()
	}()

	limit := size
	offset := (page - 1) * size
	countAll, err := r.ExercisesCount(ctx)
	if err != nil {
		return nil, -1, err
	}

	if countAll <= limit {
		all, err := r.List(ctx, ListParams{Limit: limit})
		return all, len(all), err
	}

	if countAll-offset < limit {
		offset = countAll - limit
	}

	log.Tracef("getting exercises, total count %d, limit %d, offset %d", countAll, limit, offset)

	rows, err := r.db.Query(
		ctx,
		`
			SELECT * FROM exercise
			ORDER BY created_at DESC
			LIMIT $1
			OFFSET $2;
		`,
		limit,
		offset,
	)
	if err != nil {
		return nil, -1, err
	}
	defer rows.Close()

	if err := rows.Err(); err != nil {
		return nil, -1, err
	}

	exercises, err := r.rows2exercises(rows)
	if err != nil {
		return nil, -1, err
	}
	return exercises, countAll, nil
}

func (r *Repo) ExercisesCount(ctx context.Context) (int, error) {
	ctx, span := tracing.GlobalTracer.Start(ctx, "repo.gymstats.count")
	defer span.End()

	rows, err := r.db.Query(ctx, `SELECT COUNT(*) FROM exercise`)
	if err != nil {
		return -1, err
	}
	defer rows.Close()

	if err := rows.Err(); err != nil {
		return -1, err
	}

	if rows.Next() {
		var count int
		if err := rows.Scan(&count); err == nil {
			return count, nil
		}
	}

	return -1, errors.New("unexpected error, failed to get exercises count")
}

func (r *Repo) rows2exercises(rows pgx.Rows) ([]Exercise, error) {
	var exercises []Exercise
	for rows.Next() {
		var id int
		var exerciseID string
		var muscleGroup string
		var kilos int
		var reps int
		var metadataBytes []byte
		var createdAt time.Time
		if err := rows.Scan(&id, &exerciseID, &muscleGroup, &kilos, &reps, &metadataBytes, &createdAt); err != nil {
			return nil, err
		}

		e := Exercise{
			ID:          id,
			ExerciseID:  exerciseID,
			MuscleGroup: muscleGroup,
			Kilos:       kilos,
			Reps:        reps,
			CreatedAt:   createdAt,
		}

		// parse metadata field from JSON to map[string]string
		if len(metadataBytes) > 0 {
			var metadataMap map[string]interface{}
			if err := json.Unmarshal(metadataBytes, &metadataMap); err != nil {
				return nil, fmt.Errorf("unmarshal metadata for exercise %d: %w", id, err)
			}

			e.Metadata = make(map[string]string)
			for k, v := range metadataMap {
				e.Metadata[k] = v.(string)
			}
		} else {
			e.Metadata = make(map[string]string)
		}

		exercises = append(exercises, e)
	}

	if exercises == nil {
		exercises = make([]Exercise, 0)
	}

	return exercises, nil
}
