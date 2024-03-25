package exercises

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/2beens/serjtubincom/internal/telemetry/tracing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.opentelemetry.io/otel/attribute"
)

var ErrExerciseNotFound = errors.New("exercise not found")

type ExerciseParams struct {
	ExerciseID         string
	MuscleGroup        string
	From               *time.Time
	To                 *time.Time
	OnlyProd           bool
	ExcludeTestingData bool
}

type ListParams struct {
	ExerciseParams
	Page int
	Size int
}

type Repo struct {
	db *pgxpool.Pool
}

func NewRepo(db *pgxpool.Pool) *Repo {
	return &Repo{
		db: db,
	}
}

func (r *Repo) Add(ctx context.Context, exercise Exercise) (_ *Exercise, err error) {
	ctx, span := tracing.GlobalTracer.Start(ctx, "repo.gymstats.add")
	defer func() {
		tracing.EndSpanWithErrCheck(span, err)
	}()

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

	span.SetAttributes(attribute.Int("exercise.id", id))

	exercise.ID = id
	return &exercise, nil
}

func (r *Repo) Update(ctx context.Context, exercise *Exercise) (err error) {
	ctx, span := tracing.GlobalTracer.Start(ctx, "repo.gymstats.update")
	defer func() {
		tracing.EndSpanWithErrCheck(span, err)
	}()
	span.SetAttributes(attribute.Int("id", exercise.ID))

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

func (r *Repo) Delete(ctx context.Context, id int) (err error) {
	ctx, span := tracing.GlobalTracer.Start(ctx, "repo.gymstats.delete")
	defer func() {
		tracing.EndSpanWithErrCheck(span, err)
	}()
	span.SetAttributes(attribute.Int("id", id))

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

func (r *Repo) Get(ctx context.Context, id int) (_ *Exercise, err error) {
	ctx, span := tracing.GlobalTracer.Start(ctx, "repo.gymstats.get")
	defer func() {
		tracing.EndSpanWithErrCheck(span, err)
	}()
	span.SetAttributes(attribute.Int("id", id))

	rows, err := r.db.Query(
		ctx,
		`
			SELECT
				e.id, e.exercise_id, et.name, e.muscle_group, e.kilos, e.reps, e.metadata, e.created_at
			FROM exercise e
			JOIN exercise_type et ON e.exercise_id = et.exercise_id AND e.muscle_group = et.muscle_group
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

// ListAll returns all exercises for a certain muscle group and exercise ID.
func (r *Repo) ListAll(ctx context.Context, params ExerciseParams) (_ []Exercise, err error) {
	ctx, span := tracing.GlobalTracer.Start(ctx, "repo.gymstats.listall")
	defer func() {
		tracing.EndSpanWithErrCheck(span, err)
	}()
	span.SetAttributes(attribute.String("exercise_id", params.ExerciseID))
	span.SetAttributes(attribute.String("muscle_group", params.MuscleGroup))
	span.SetAttributes(attribute.Bool("only-prod", params.OnlyProd))
	span.SetAttributes(attribute.Bool("exclude-testing-data", params.ExcludeTestingData))
	if params.From != nil {
		span.SetAttributes(attribute.String("from", params.From.String()))
	}
	if params.To != nil {
		span.SetAttributes(attribute.String("to", params.To.String()))
	}

	rows, err := r.db.Query(
		ctx,
		`
			SELECT
				e.id, e.exercise_id, et.name, e.muscle_group, e.kilos, e.reps, e.metadata, e.created_at
			FROM exercise e
			JOIN exercise_type et ON e.exercise_id = et.exercise_id AND e.muscle_group = et.muscle_group
				WHERE ($1::text = '' OR e.exercise_id = $1)
				AND ($2::text = '' OR e.muscle_group = $2)
				AND ($3::timestamp IS NULL OR e.created_at >= $3)
				AND ($4::timestamp IS NULL OR e.created_at <= $4)
				AND ($5::boolean IS FALSE OR e.metadata->>'env' = 'prod' OR e.metadata->>'env' = 'production')
				AND ($6::boolean IS FALSE OR e.metadata->>'testing' != 'true' OR e.metadata->>'test' != 'true')
			ORDER BY e.created_at DESC;`,
		params.ExerciseID, params.MuscleGroup,
		params.From, params.To,
		params.OnlyProd, params.ExcludeTestingData,
	)
	if err != nil {
		return nil, fmt.Errorf("query: %w", err)
	}
	defer rows.Close()

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows: %w", err)
	}

	exercises, err := r.rows2exercises(rows)
	if err != nil {
		return nil, fmt.Errorf("rows2exercises: %w", err)
	}
	return exercises, nil
}

// List is like ListAll, but it returns the specific PAGE for a certain muscle group and exercise ID
// i.e. is used for pagination.
func (r *Repo) List(ctx context.Context, params ListParams) (_ []Exercise, total int, err error) {
	ctx, span := tracing.GlobalTracer.Start(ctx, "repo.gymstats.list")
	defer func() {
		tracing.EndSpanWithErrCheck(span, err)
	}()
	span.SetAttributes(attribute.Int("page", params.Page))
	span.SetAttributes(attribute.Int("size", params.Size))
	span.SetAttributes(attribute.String("exercise_id", params.ExerciseID))
	span.SetAttributes(attribute.String("muscle_group", params.MuscleGroup))
	span.SetAttributes(attribute.Bool("only-prod", params.OnlyProd))
	span.SetAttributes(attribute.Bool("exclude-testing-data", params.ExcludeTestingData))
	if params.From != nil {
		span.SetAttributes(attribute.String("from", params.From.String()))
	}
	if params.To != nil {
		span.SetAttributes(attribute.String("to", params.To.String()))
	}

	if params.Page < 1 {
		return nil, -1, errors.New("page must be greater than 0")
	}
	if params.Size < 1 {
		return nil, -1, errors.New("size must be greater than 0")
	}

	limit := params.Size
	offset := (params.Page - 1) * params.Size
	countAll, err := r.ExercisesCount(ctx, params)
	if err != nil {
		return nil, -1, err
	}

	if countAll <= limit {
		limit = countAll
		offset = 0
	}

	if countAll-offset < limit {
		offset = countAll - limit
	}

	span.SetAttributes(attribute.Int("count_all", countAll))
	span.SetAttributes(attribute.Int("limit", limit))
	span.SetAttributes(attribute.Int("offset", offset))

	rows, err := r.db.Query(
		ctx,
		`
			SELECT
				e.id, e.exercise_id, et.name, e.muscle_group, e.kilos, e.reps, e.metadata, e.created_at
			FROM exercise e
			JOIN exercise_type et ON e.exercise_id = et.exercise_id AND e.muscle_group = et.muscle_group
				WHERE ($1::text = '' OR e.exercise_id = $1)
				AND ($2::text = '' OR e.muscle_group = $2)
				AND ($5::boolean IS FALSE OR e.metadata->>'env' = 'prod' OR e.metadata->>'env' = 'production')
				AND ($6::boolean IS FALSE OR e.metadata->>'testing' != 'true' OR e.metadata->>'test' != 'true')
			ORDER BY e.created_at DESC
			LIMIT $3
			OFFSET $4;`,
		params.ExerciseID, params.MuscleGroup,
		limit, offset,
		params.OnlyProd, params.ExcludeTestingData,
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

func (r *Repo) ExercisesCount(ctx context.Context, params ListParams) (_ int, err error) {
	ctx, span := tracing.GlobalTracer.Start(ctx, "repo.gymstats.count")
	defer func() {
		tracing.EndSpanWithErrCheck(span, err)
	}()

	rows, err := r.db.Query(ctx, `
		SELECT COUNT(*) FROM exercise
			WHERE ($1::text = '' OR exercise_id = $1)
			AND ($2::text = '' OR muscle_group = $2)
		  	AND ($3::timestamp IS NULL OR created_at >= $3)
			AND ($4::timestamp IS NULL OR created_at <= $4)
			AND ($5::boolean IS FALSE OR metadata->>'env' = 'prod' OR metadata->>'env' = 'production')
			AND ($6::boolean IS FALSE OR metadata->>'testing' != 'true' OR metadata->>'test' != 'true');
	`,
		params.ExerciseID, params.MuscleGroup,
		params.From, params.To,
		params.OnlyProd, params.ExcludeTestingData,
	)
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
		var exerciseName string
		var muscleGroup string
		var kilos int
		var reps int
		var metadataBytes []byte
		var createdAt time.Time
		if err := rows.Scan(&id, &exerciseID, &exerciseName, &muscleGroup, &kilos, &reps, &metadataBytes, &createdAt); err != nil {
			return nil, err
		}

		e := Exercise{
			ID:           id,
			ExerciseID:   exerciseID,
			ExerciseName: exerciseName,
			MuscleGroup:  muscleGroup,
			Kilos:        kilos,
			Reps:         reps,
			CreatedAt:    createdAt,
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
