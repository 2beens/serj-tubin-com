package exercises

import "time"

// ProgressData represents progress statistics for a specific date
type ProgressData struct {
	Date        time.Time `json:"date"`
	AvgWeight   float64   `json:"avg_weight"`
	MaxWeight    int       `json:"max_weight"`
	TotalVolume  float64   `json:"total_volume"` // sum of (kilos * reps) for the day
	ExerciseCount int     `json:"exercise_count"`
}

