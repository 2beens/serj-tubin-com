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

type AvgWaitResponse struct {
	// AvgWait is the average wait between sets for all exercises ever done
	AvgWait time.Duration `json:"avgWait"`
	// AvgWaitPerDay is the average wait between exercises for each day
	AvgWaitPerDay map[time.Time]time.Duration `json:"avgWaitPerDay"`
}

// AvgWaitBetweenExercises calculates the average wait between exercises
// for all exercises ever done and for each day.
// Accepts the ExerciseParams to filter the exercises, so leave it empty
// to get the average wait for all exercises ever done.
func (a *Analyzer) AvgWaitBetweenExercises(
	ctx context.Context,
	exerciseParams ExerciseParams,
) (*AvgWaitResponse, error) {
	ctx, span := tracing.GlobalTracer.Start(ctx, "analyzer.gymstats.avg-wait")
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

	avgWaitPerDay := make(map[time.Time]time.Duration)
	for day, dayExercises := range day2exercises {
		if len(dayExercises) == 1 {
			continue
		}
		var avgWait time.Duration
		for i, ex := range dayExercises {
			if i == 0 {
				continue
			}
			avgWait += ex.CreatedAt.Sub(dayExercises[i-1].CreatedAt)
		}
		avgWait /= time.Duration(len(dayExercises) - 1)

		// get absolute value of avgWait
		if avgWait < 0 {
			avgWait = -avgWait
		}

		avgWaitPerDay[day] = avgWait
	}

	var avgWait time.Duration
	for _, dayExercises := range avgWaitPerDay {
		avgWait += dayExercises
	}
	avgWait /= time.Duration(len(avgWaitPerDay))

	return &AvgWaitResponse{
		AvgWait:       avgWait,
		AvgWaitPerDay: avgWaitPerDay,
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
