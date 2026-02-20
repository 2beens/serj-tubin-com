package exercises

import "time"

// ProgressData represents progress statistics for a specific date.
// When multiple exercise types are requested, ExerciseID is set per row so the client can show separate series.
type ProgressData struct {
	Date          time.Time `json:"date"`
	ExerciseID    string    `json:"exercise_id,omitempty"` // set when progress is grouped by exercise (multi-select)
	AvgWeight     float64   `json:"avg_weight"`
	MaxWeight     int       `json:"max_weight"`
	TotalVolume   float64   `json:"total_volume"`   // sum of (kilos * reps) for the day
	ExerciseCount int       `json:"exercise_count"`
}

// ProgressionRateData represents progression rate comparison between current and past periods
type ProgressionRateData struct {
	Period           string  `json:"period"`            // "30", "60", or "90" days
	CurrentAvgWeight float64 `json:"current_avg_weight"`
	PastAvgWeight    float64 `json:"past_avg_weight"`
	AvgWeightChange  float64 `json:"avg_weight_change"`  // current - past
	AvgWeightChangePercent float64 `json:"avg_weight_change_percent"` // ((current - past) / past) * 100
	
	CurrentMaxWeight int     `json:"current_max_weight"`
	PastMaxWeight    int     `json:"past_max_weight"`
	MaxWeightChange  int     `json:"max_weight_change"`  // current - past
	MaxWeightChangePercent float64 `json:"max_weight_change_percent"` // ((current - past) / past) * 100
	
	CurrentTotalVolume float64 `json:"current_total_volume"`
	PastTotalVolume    float64 `json:"past_total_volume"`
	TotalVolumeChange  float64 `json:"total_volume_change"`  // current - past
	TotalVolumeChangePercent float64 `json:"total_volume_change_percent"` // ((current - past) / past) * 100
	
	CurrentExerciseCount int `json:"current_exercise_count"`
	PastExerciseCount    int `json:"past_exercise_count"`
	ExerciseCountChange  int `json:"exercise_count_change"`  // current - past
}

