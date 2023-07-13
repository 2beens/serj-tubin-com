package stats

import (
	"context"
	"time"

	"github.com/2beens/serjtubincom/internal/gymstats/repo"
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

//go:generate mockgen -source=$GOFILE -destination=mocks_test.go -package=stats_test

type exercisesRepo interface {
	ListAll(ctx context.Context, params repo.ExerciseParams) (_ []repo.Exercise, err error)
}

type Exercises struct {
	repo exercisesRepo
}

func NewExercisesStats(repo exercisesRepo) *Exercises {
	return &Exercises{
		repo: repo,
	}
}

func (a *Exercises) ExerciseHistory(
	ctx context.Context,
	exerciseID, muscleGroup string,
) (*ExerciseHistory, error) {
	ctx, span := tracing.GlobalTracer.Start(ctx, "analyzer.gymstats.exerciseHistory")
	defer span.End()

	exercises, err := a.repo.ListAll(ctx, repo.ExerciseParams{
		ExerciseID:         exerciseID,
		MuscleGroup:        muscleGroup,
		OnlyProd:           true,
		ExcludeTestingData: true,
	})
	if err != nil {
		return nil, err
	}

	history := &ExerciseHistory{
		ExerciseID:  exerciseID,
		MuscleGroup: muscleGroup,
		Stats:       make(map[time.Time]ExerciseStats),
	}

	day2exercises := make(map[time.Time][]repo.Exercise)
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
