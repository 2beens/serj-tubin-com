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
	MuscleGroup string
	ExerciseId  string
}

func (r *Repo) GetExerciseType(ctx context.Context, exerciseTypeID string) (_ ExerciseType, err error) {
	ctx, span := tracing.GlobalTracer.Start(ctx, "repo.gymstats.exercise_types.get")
	defer func() {
		tracing.EndSpanWithErrCheck(span, err)
	}()

	var exerciseType ExerciseType
	err = r.db.QueryRow(
		ctx,
		`
			SELECT 
			    id, muscle_group, name, description, created_at
			FROM exercise_type
			WHERE id = $1
		`,
		exerciseTypeID,
	).Scan(
		&exerciseType.ID,
		&exerciseType.MuscleGroup,
		&exerciseType.Name,
		&exerciseType.Description,
		&exerciseType.CreatedAt,
	)
	if err != nil {
		return ExerciseType{}, fmt.Errorf("exercise type [query row]: %w", err)
	}

	exerciseType.Images, err = r.GetExerciseTypeImages(ctx, exerciseTypeID)
	if err != nil {
		return ExerciseType{}, fmt.Errorf("exercise type images: %w", err)
	}

	return exerciseType, nil
}

func (r *Repo) GetExerciseTypeImages(ctx context.Context, exerciseTypeID string) (_ []ExerciseImage, err error) {
	ctx, span := tracing.GlobalTracer.Start(ctx, "repo.gymstats.exercise_types.get_images")
	defer func() {
		tracing.EndSpanWithErrCheck(span, err)
	}()

	rows, err := r.db.Query(
		ctx,
		`
			SELECT
			    id, exercise_id, image_path, created_at
			FROM exercise_image
			WHERE exercise_id = $1
		`,
		exerciseTypeID,
	)
	if err != nil {
		return nil, fmt.Errorf("exercise images [query]: %w", err)
	}
	defer rows.Close()

	var exerciseImages []ExerciseImage
	for rows.Next() {
		var exerciseImage ExerciseImage
		err := rows.Scan(
			&exerciseImage.ID,
			&exerciseImage.ExerciseID,
			&exerciseImage.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("exercise images [rows scan]: %w", err)
		}
		exerciseImages = append(exerciseImages, exerciseImage)
	}

	return exerciseImages, nil
}

func (r *Repo) GetExerciseTypes(ctx context.Context, params GetExerciseTypesParams) (_ []ExerciseType, err error) {
	ctx, span := tracing.GlobalTracer.Start(ctx, "repo.gymstats.exercise_types.get_types")
	defer func() {
		tracing.EndSpanWithErrCheck(span, err)
	}()
	if params.MuscleGroup != "" {
		span.SetAttributes(attribute.String("params.muscleGroup", params.MuscleGroup))
	}
	if params.ExerciseId != "" {
		span.SetAttributes(attribute.String("params.exerciseId", params.ExerciseId))
	}

	rows, err := r.db.Query(
		ctx,
		`
			SELECT
			    id, muscle_group, name, description, created_at
			FROM exercise_type
			WHERE ($1::text = '' OR muscle_group = $1) AND ($2::text = '' OR id = $2)
		`,
		params.MuscleGroup,
		params.ExerciseId,
	)
	if err != nil {
		return nil, fmt.Errorf("exercise types [query]: %w", err)
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
			return nil, fmt.Errorf("exercise types [rows scan]: %w", err)
		}

		exerciseType.Images, err = r.GetExerciseTypeImages(ctx, exerciseType.ID)
		if err != nil {
			return nil, fmt.Errorf("get exercise type images for type [%s]: %w", exerciseType.ID, err)
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
			INSERT INTO exercise_type
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

func (r *Repo) AddExerciseTypeImage(ctx context.Context, exerciseImage ExerciseImage) (err error) {
	ctx, span := tracing.GlobalTracer.Start(ctx, "repo.gymstats.exercise_types.add_image")
	defer func() {
		tracing.EndSpanWithErrCheck(span, err)
	}()

	if exerciseImage.CreatedAt.IsZero() {
		exerciseImage.CreatedAt = time.Now()
	}

	_, err = r.db.Exec(
		ctx,
		`
			INSERT INTO exercise_image
			    (id, exercise_id, created_at)
			VALUES ($1, $2, $3)
		`,
		exerciseImage.ID,
		exerciseImage.ExerciseID,
		exerciseImage.CreatedAt,
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
			UPDATE exercise_type
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
			DELETE FROM exercise_type
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
