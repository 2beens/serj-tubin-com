package gymstats

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Exercise struct {
	ID          int       `json:"id"`
	ExerciseID  string    `json:"exerciseId"`
	MuscleGroup string    `json:"muscleGroup"`
	Kilos       int       `json:"kilos"`
	Reps        int       `json:"reps"`
	CreatedAt   time.Time `json:"createdAt"`
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
	rows, err := r.db.Query(
		ctx,
		`INSERT INTO exercise 
    			(exercise_id, muscle_group, kilos, reps, created_at) 
				VALUES ($1, $2, $3, $4, $5) 
			RETURNING id;`,
		exercise.ExerciseID, exercise.MuscleGroup, exercise.Kilos, exercise.Reps, exercise.CreatedAt,
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

func (r *Repo) List(ctx context.Context) ([]Exercise, error) {
	rows, err := r.db.Query(
		ctx,
		`
			SELECT
				id, exercise_id, muscle_group, kilos, reps, created_at
			FROM exercise
			ORDER BY created_at DESC;`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	if err := rows.Err(); err != nil {
		return nil, err
	}

	var notes []Exercise
	for rows.Next() {
		var id int
		var exerciseID string
		var muscleGroup string
		var kilos int
		var reps int
		var createdAt time.Time
		if err := rows.Scan(&id, &exerciseID, &muscleGroup, &kilos, &reps, &createdAt); err != nil {
			return nil, err
		}
		notes = append(notes, Exercise{
			ID:          id,
			ExerciseID:  exerciseID,
			MuscleGroup: muscleGroup,
			Kilos:       kilos,
			Reps:        reps,
			CreatedAt:   createdAt,
		})
	}

	return notes, nil
}
