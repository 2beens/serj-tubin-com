package exercises

import (
	"context"
	"time"

	"github.com/2beens/serjtubincom/internal/telemetry/tracing"
)

// ExerciseHistory represents the history of an exercise
// so that, for each day, we get the average kilos and reps per set
type ExerciseHistory struct {
	ExerciseID  string                      `json:"exerciseId"`
	MuscleGroup string                      `json:"muscleGroup"`
	Stats       map[time.Time]ExerciseStats `json:"stats"`
}

type ExerciseStats struct {
	AvgKilos int `json:"avgKilos"`
	AvgReps  int `json:"avgReps"`
	Sets     int `json:"sets"`
}

type Analyzer struct {
	repo exercisesRepo
}

func NewAnalyzer(repo exercisesRepo) *Analyzer {
	return &Analyzer{
		repo: repo,
	}
}

type AvgSetDurationResponse struct {
	// Duration is the average duration between sets for all exercises ever done
	Duration time.Duration `json:"duration"`
	// DurationPerDay is the average set duration between exercises for each day
	DurationPerDay map[time.Time]time.Duration `json:"durationPerDay"`
}

// AvgSetDuration calculates the average duration between sets
// for all exercises ever done and for each day.
// Accepts the ExerciseParams to filter the exercises, so leave it empty
// to get the average wait for all exercises ever done.
func (a *Analyzer) AvgSetDuration(
	ctx context.Context,
	exerciseParams ExerciseParams,
) (*AvgSetDurationResponse, error) {
	ctx, span := tracing.GlobalTracer.Start(ctx, "analyzer.gymstats.avg-set-duration")
	defer span.End()

	exercises, err := a.repo.ListAll(ctx, exerciseParams)
	if err != nil {
		return nil, err
	}

	day2exercises := make(map[time.Time][]Exercise)
	for _, ex := range exercises {
		day := ex.CreatedAt.Truncate(24 * time.Hour)
		day2exercises[day] = append(day2exercises[day], ex)
	}

	avgDurationPerDay := make(map[time.Time]time.Duration)
	for day, dayExercises := range day2exercises {
		if len(dayExercises) == 1 {
			continue
		}
		var avgDuration time.Duration
		for i, ex := range dayExercises {
			if i == 0 {
				continue
			}
			avgDuration += ex.CreatedAt.Sub(dayExercises[i-1].CreatedAt)
		}
		avgDuration /= time.Duration(len(dayExercises) - 1)

		// get absolute value of avgDuration
		if avgDuration < 0 {
			avgDuration = -avgDuration
		}

		avgDurationPerDay[day] = avgDuration
	}

	var avgDuration time.Duration
	for _, dayExercises := range avgDurationPerDay {
		avgDuration += dayExercises
	}
	avgDuration /= time.Duration(len(avgDurationPerDay))

	return &AvgSetDurationResponse{
		Duration:       avgDuration,
		DurationPerDay: avgDurationPerDay,
	}, nil
}

func (a *Analyzer) ExerciseHistory(
	ctx context.Context,
	exerciseParams ExerciseParams,
) (*ExerciseHistory, error) {
	ctx, span := tracing.GlobalTracer.Start(ctx, "analyzer.gymstats.exerciseHistory")
	defer span.End()

	exercises, err := a.repo.ListAll(ctx, exerciseParams)
	if err != nil {
		return nil, err
	}

	history := &ExerciseHistory{
		ExerciseID:  exerciseParams.ExerciseID,
		MuscleGroup: exerciseParams.MuscleGroup,
		Stats:       make(map[time.Time]ExerciseStats),
	}

	day2exercises := make(map[time.Time][]Exercise)
	for _, ex := range exercises {
		day := ex.CreatedAt.Truncate(24 * time.Hour)
		day2exercises[day] = append(day2exercises[day], ex)
	}

	for day, dayExercises := range day2exercises {
		var avgKilos, avgReps int
		for _, ex := range dayExercises {
			avgKilos += ex.Kilos
			avgReps += ex.Reps
		}
		avgKilos /= len(dayExercises)
		avgReps /= len(dayExercises)
		history.Stats[day] = ExerciseStats{
			AvgKilos: avgKilos,
			AvgReps:  avgReps,
			Sets:     len(dayExercises),
		}
	}

	return history, nil
}
