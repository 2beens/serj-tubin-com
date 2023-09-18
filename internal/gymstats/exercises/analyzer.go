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

func (a *Analyzer) AvgWaitBetweenExercises(
	ctx context.Context,
	exerciseParams ExerciseParams,
) (time.Duration, error) {
	ctx, span := tracing.GlobalTracer.Start(ctx, "analyzer.gymstats.avg-wait")
	defer span.End()

	exercises, err := a.repo.ListAll(ctx, exerciseParams)
	if err != nil {
		return 0, err
	}

	// TODO: wait has to be calculated between sets in the same muscle group and the same exercise
	// plan:
	// - iterate by by each day
	// - in that day, iterate by each exercise sets
	// - if the exercise is the same, calculate the wait between sets
	//    - something like:
	//	  - (if exercise[i-1].muscleGroup == exercise[i].muscleGroup) && (exercise[i-1].exerciseID == exercise[i].exerciseID)
	// maybe also add the option to get the avg wait between all exercise sets in a single day
	// or - return an object that contains avgWait for all exercises and avgWait for exercises in a single day

	var totalWait time.Duration

	// TODO: calculate
	totalWait = time.Minute

	return totalWait / time.Duration(len(exercises)-1), nil
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
