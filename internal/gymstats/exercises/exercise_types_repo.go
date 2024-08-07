package exercises

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/2beens/serjtubincom/internal/telemetry/tracing"
	"github.com/2beens/serjtubincom/pkg"

	"go.opentelemetry.io/otel/attribute"
)

var (
	ErrExerciseTypeNotFound = errors.New("exercise type not found")
	ErrAlreadyExists        = errors.New("exercise type already exists")
	ErrExerciseTypeInUse    = errors.New("exercise type is in use")
)

type GetExerciseTypesParams struct {
	MuscleGroup string
	ExerciseId  string
}

func (r *Repo) GetExerciseType(ctx context.Context, exerciseTypeID, muscleGroup string) (_ ExerciseType, err error) {
	ctx, span := tracing.GlobalTracer.Start(ctx, "repo.gymstats.exercise_types.get")
	defer func() {
		tracing.EndSpanWithErrCheck(span, err)
	}()

	var description *string
	var exerciseType ExerciseType
	err = r.db.QueryRow(
		ctx,
		`
			SELECT 
			    exercise_id, muscle_group, name, description, created_at
			FROM exercise_type
			WHERE exercise_id = $1 AND muscle_group = $2
		`,
		exerciseTypeID, muscleGroup,
	).Scan(
		&exerciseType.ExerciseID,
		&exerciseType.MuscleGroup,
		&exerciseType.Name,
		&description,
		&exerciseType.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ExerciseType{}, ErrExerciseTypeNotFound
		}
		return ExerciseType{}, fmt.Errorf("exercise type [query row]: %w", err)
	}

	if description != nil {
		exerciseType.Description = *description
	}

	exerciseType.Images, err = r.GetExerciseTypeImages(ctx, exerciseTypeID, muscleGroup)
	if err != nil {
		return ExerciseType{}, fmt.Errorf("exercise type images: %w", err)
	}

	return exerciseType, nil
}

func (r *Repo) GetExerciseTypeImages(ctx context.Context, exerciseTypeID, muscleGroup string) (_ []ExerciseImage, err error) {
	ctx, span := tracing.GlobalTracer.Start(ctx, "repo.gymstats.exercise_types.get_images")
	defer func() {
		tracing.EndSpanWithErrCheck(span, err)
	}()

	rows, err := r.db.Query(
		ctx,
		`
			SELECT
			    id, exercise_id, muscle_group, created_at
			FROM exercise_image
			WHERE exercise_id = $1 AND muscle_group = $2
		`,
		exerciseTypeID, muscleGroup,
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
			&exerciseImage.MuscleGroup,
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
			    exercise_id, muscle_group, name, description, created_at
			FROM exercise_type
			WHERE ($1::text = '' OR muscle_group = $1) AND ($2::text = '' OR exercise_id = $2)
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
		var description *string
		var exerciseType ExerciseType
		err := rows.Scan(
			&exerciseType.ExerciseID,
			&exerciseType.MuscleGroup,
			&exerciseType.Name,
			&description,
			&exerciseType.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("exercise types [rows scan]: %w", err)
		}

		if description != nil {
			exerciseType.Description = *description
		}

		exerciseType.Images, err = r.GetExerciseTypeImages(ctx, exerciseType.ExerciseID, exerciseType.MuscleGroup)
		if err != nil {
			return nil, fmt.Errorf("get exercise type images for type [%s]: %w", exerciseType.ExerciseID, err)
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
			    (exercise_id, muscle_group, name, description, created_at)
			VALUES ($1, $2, $3, $4, $5)
		`,
		exerciseType.ExerciseID,
		exerciseType.MuscleGroup,
		exerciseType.Name,
		exerciseType.Description,
		exerciseType.CreatedAt,
	)
	if err != nil {
		if pkg.IsUniqueViolationError(err) {
			return ErrAlreadyExists
		}
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
			    (id, exercise_id, muscle_group, created_at)
			VALUES ($1, $2, $3, $4)
		`,
		exerciseImage.ID,
		exerciseImage.ExerciseID,
		exerciseImage.MuscleGroup,
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
			SET exercise_id = $1, muscle_group = $2, name = $3, description = $4
			WHERE exercise_id = $1 AND muscle_group = $2
		`,
		exerciseType.ExerciseID,
		exerciseType.MuscleGroup,
		exerciseType.Name,
		exerciseType.Description,
	)
	if err != nil {
		return err
	}

	return nil
}

func (r *Repo) ExerciseTypeIsInUse(ctx context.Context, exerciseTypeID, muscleGroup string) (_ bool, err error) {
	ctx, span := tracing.GlobalTracer.Start(ctx, "repo.gymstats.exercise_types.is_in_use")
	defer func() {
		tracing.EndSpanWithErrCheck(span, err)
	}()

	var count int
	err = r.db.QueryRow(
		ctx,
		`
			SELECT COUNT(*)
			FROM exercise
			WHERE exercise_id = $1 AND muscle_group = $2
		`,
		exerciseTypeID, muscleGroup,
	).Scan(&count)
	if err != nil {
		return false, err
	}

	return count > 0, nil
}

func (r *Repo) DeleteExerciseType(ctx context.Context, exerciseTypeID, muscleGroup string) (err error) {
	ctx, span := tracing.GlobalTracer.Start(ctx, "repo.gymstats.exercise_types.delete")
	defer func() {
		tracing.EndSpanWithErrCheck(span, err)
	}()

	rows, err := r.db.Exec(
		ctx,
		`
			DELETE FROM exercise_type
			WHERE exercise_id = $1 AND muscle_group = $2
		`,
		exerciseTypeID, muscleGroup,
	)
	if err != nil {
		if pkg.IsForeignKeyViolationError(err) {
			return ErrExerciseTypeInUse
		}
		return err
	}

	if rows.RowsAffected() == 0 {
		return ErrExerciseTypeNotFound
	}

	return nil
}

func (r *Repo) DeleteExerciseTypeImage(ctx context.Context, exerciseImageID int64) (err error) {
	ctx, span := tracing.GlobalTracer.Start(ctx, "repo.gymstats.exercise_types.delete_image")
	defer func() {
		tracing.EndSpanWithErrCheck(span, err)
	}()

	rows, err := r.db.Exec(
		ctx,
		`
			DELETE FROM exercise_image
			WHERE id = $1
		`,
		exerciseImageID,
	)
	if err != nil {
		return err
	}

	if rows.RowsAffected() == 0 {
		return ErrExerciseTypeNotFound
	}

	return nil
}
