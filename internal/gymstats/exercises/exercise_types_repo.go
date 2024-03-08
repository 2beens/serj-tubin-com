package exercises

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/2beens/serjtubincom/internal/telemetry/tracing"

	"go.opentelemetry.io/otel/attribute"
)

var ErrExerciseTypeNotFound = errors.New("exercise type not found")

type GetExerciseTypesParams struct {
	MuscleGroup *string
}

func (r *Repo) GetExerciseTypes(ctx context.Context, params GetExerciseTypesParams) (_ []ExerciseType, err error) {
	ctx, span := tracing.GlobalTracer.Start(ctx, "repo.gymstats.exercise_types.get")
	defer func() {
		tracing.EndSpanWithErrCheck(span, err)
	}()
	span.SetAttributes(attribute.Int("params.muscleGroup", len(*params.MuscleGroup)))

	rows, err := r.db.Query(
		ctx,
		`
			SELECT 
			    id, muscle_group, name, description, created_at
			FROM exercise_types
			WHERE ($1 IS NULL OR muscle_group = $1)
		`,
		params.MuscleGroup,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("exercise types [rows error]: %w", err)
	}

	var exerciseTypes []ExerciseType
	for rows.Next() {
		var exerciseType ExerciseType
		err := rows.Scan(
			&exerciseType.ID,
			&exerciseType.MuscleGroup,
			&exerciseType.Name,
			&exerciseType.Description,
			&exerciseType.CreatedAt,
		)
		if err != nil {
			return nil, err
		}

		exerciseTypes = append(exerciseTypes, exerciseType)
	}

	return exerciseTypes, nil
}

func (r *Repo) AddExerciseType(ctx context.Context, exerciseType ExerciseType) (err error) {
	ctx, span := tracing.GlobalTracer.Start(ctx, "repo.gymstats.exercise_types.add")
	defer func() {
		tracing.EndSpanWithErrCheck(span, err)
	}()

	if exerciseType.CreatedAt.IsZero() {
		exerciseType.CreatedAt = time.Now()
	}

	_, err = r.db.Exec(
		ctx,
		`
			INSERT INTO exercise_types 
			    (id, muscle_group, name, description, created_at)
			VALUES ($1, $2, $3, $4, $5)
		`,
		exerciseType.ID,
		exerciseType.MuscleGroup,
		exerciseType.Name,
		exerciseType.Description,
		exerciseType.CreatedAt,
	)
	if err != nil {
		return err
	}

	return nil
}

func (r *Repo) UpdateExerciseType(ctx context.Context, exerciseType ExerciseType) (err error) {
	ctx, span := tracing.GlobalTracer.Start(ctx, "repo.gymstats.exercise_types.update")
	defer func() {
		tracing.EndSpanWithErrCheck(span, err)
	}()

	_, err = r.db.Exec(
		ctx,
		`
			UPDATE exercise_types
			SET muscle_group = $2, name = $3, description = $4
			WHERE id = $1
		`,
		exerciseType.ID,
		exerciseType.MuscleGroup,
		exerciseType.Name,
		exerciseType.Description,
	)
	if err != nil {
		return err
	}

	return nil
}

func (r *Repo) DeleteExerciseType(ctx context.Context, exerciseTypeID string) (err error) {
	ctx, span := tracing.GlobalTracer.Start(ctx, "repo.gymstats.exercise_types.delete")
	defer func() {
		tracing.EndSpanWithErrCheck(span, err)
	}()

	rows, err := r.db.Exec(
		ctx,
		`
			DELETE FROM exercise_types
			WHERE id = $1
		`,
		exerciseTypeID,
	)
	if err != nil {
		return err
	}

	if rows.RowsAffected() == 0 {
		return ErrExerciseTypeNotFound
	}

	return nil
}
