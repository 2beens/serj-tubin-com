package exercises

import "time"

type Exercise struct {
	ID          int               `json:"id"`
	ExerciseID  string            `json:"exerciseId"`
	MuscleGroup string            `json:"muscleGroup"`
	Kilos       int               `json:"kilos"`
	Reps        int               `json:"reps"`
	CreatedAt   time.Time         `json:"createdAt"`
	Metadata    map[string]string `json:"metadata"`
}

type ExerciseType struct {
	ID          string    `json:"id"`
	MuscleGroup string    `json:"muscleGroup"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"createdAt"`

	Images []ExerciseImage `json:"images"`
}

type ExerciseImage struct {
	ID         int64     `json:"id"`
	ExerciseID string    `json:"exerciseId"`
	CreatedAt  time.Time `json:"createdAt"`
}
