package gymstats

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
}

type Analyzer struct {
	repo exercisesRepo
}

func NewAnalyzer(repo exercisesRepo) *Analyzer {
	return &Analyzer{
		repo: repo,
	}
}

func (a *Analyzer) ExerciseHistory(
	ctx context.Context,
	exerciseID, muscleGroup string,
) (*ExerciseHistory, error) {
	ctx, span := tracing.GlobalTracer.Start(ctx, "analyzer.gymstats.exerciseHistory")
	defer span.End()

	exercises, err := a.repo.ListAll(ctx, ExerciseParams{
		ExerciseID:  exerciseID,
		MuscleGroup: muscleGroup,
	})
	if err != nil {
		return nil, err
	}

	history := &ExerciseHistory{
		ExerciseID:  exerciseID,
		MuscleGroup: muscleGroup,
		Stats:       make(map[time.Time]ExerciseStats),
	}

	for _, ex := range exercises {
		day := ex.CreatedAt.Truncate(24 * time.Hour)
		stats, ok := history.Stats[day]
		if !ok {
			stats = ExerciseStats{}
		}
		stats.AvgKilos = (stats.AvgKilos + ex.Kilos) / 2
		stats.AvgReps = (stats.AvgReps + ex.Reps) / 2
		history.Stats[day] = stats
	}

	return history, nil
}
