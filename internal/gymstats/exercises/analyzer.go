package exercises

import (
	"context"
	"fmt"
	"time"

	"github.com/2beens/serjtubincom/internal/telemetry/tracing"
	"go.opentelemetry.io/otel/attribute"
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
) (_ *AvgSetDurationResponse, err error) {
	ctx, span := tracing.GlobalTracer.Start(ctx, "analyzer.gymstats.avg-set-duration")
	defer func() {
		tracing.EndSpanWithErrCheck(span, err)
	}()

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

	if len(avgDurationPerDay) == 0 {
		return &AvgSetDurationResponse{
			Duration:       0,
			DurationPerDay: avgDurationPerDay,
		}, nil
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
) (_ *ExerciseHistory, err error) {
	ctx, span := tracing.GlobalTracer.Start(ctx, "analyzer.gymstats.exerciseHistory")
	defer func() {
		tracing.EndSpanWithErrCheck(span, err)
	}()

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

type ExercisePercentageInfo struct {
	ExerciseName string  `json:"exerciseName"`
	Percentage   float64 `json:"percentage"`
}

// ExercisePercentages returns the percentages of the exercises worked out for a given muscle group
func (a *Analyzer) ExercisePercentages(
	ctx context.Context,
	muscleGroup string,
	onlyProd, excludeTestingData bool,
) (_ map[string]ExercisePercentageInfo, err error) {
	ctx, span := tracing.GlobalTracer.Start(ctx, "analyzer.gymstats.exerciseHistory")
	defer func() {
		tracing.EndSpanWithErrCheck(span, err)
	}()

	span.SetAttributes(attribute.String("muscle_group", muscleGroup))

	exercises, err := a.repo.ListAll(ctx, ExerciseParams{
		MuscleGroup:        muscleGroup,
		OnlyProd:           onlyProd,
		ExcludeTestingData: excludeTestingData,
	})
	if err != nil {
		return nil, err
	}

	exercise2count := make(map[string]int)
	for _, ex := range exercises {
		exercise2count[ex.ExerciseID]++
	}

	exercise2name := make(map[string]string)
	for _, ex := range exercises {
		exercise2name[ex.ExerciseID] = ex.ExerciseName
	}

	exercise2percentage := make(map[string]ExercisePercentageInfo)
	for exercise, count := range exercise2count {
		p := float64(count) / float64(len(exercises)) * 100
		// leave only 2 decimals
		p = float64(int(p*100)) / 100
		exercise2percentage[exercise] = ExercisePercentageInfo{
			ExerciseName: exercise2name[exercise],
			Percentage:   p,
		}
	}

	// get all exercise types, even if there are no exercises for them
	// and set their percentage to 0
	exTypes, err := a.repo.GetExerciseTypes(ctx, GetExerciseTypesParams{
		MuscleGroup: muscleGroup,
	})
	if err != nil {
		return nil, fmt.Errorf("get exercise types: %w", err)
	}

	for _, exType := range exTypes {
		if _, ok := exercise2percentage[exType.ExerciseID]; !ok {
			exercise2percentage[exType.ExerciseID] = ExercisePercentageInfo{
				ExerciseName: exType.Name,
				Percentage:   0,
			}
		}
	}

	return exercise2percentage, nil
}
