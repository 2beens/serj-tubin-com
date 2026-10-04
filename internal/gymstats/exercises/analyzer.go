package exercises

import (
	"context"
	"fmt"
	"math"
	"slices"
	"time"

	"github.com/2beens/serjtubincom/internal/telemetry/tracing"

	"go.opentelemetry.io/otel/attribute"
)

// maxSetRest is the longest gap still counted as rest between sets.
// Longer gaps are a break between sessions. ponytail: fixed 30m, make it a query param if that cutoff is wrong.
const maxSetRest = 30 * time.Minute

// ExerciseHistory represents the history of an exercise
// so that, for each day, we get the average kilos and reps per set
type ExerciseHistory struct {
	ExerciseID  string                      `json:"exerciseId"`
	MuscleGroup string                      `json:"muscleGroup"`
	Stats       map[time.Time]ExerciseStats `json:"stats"`
}

type ExerciseStats struct {
	AvgKilos float64 `json:"avgKilos"`
	AvgReps  float64 `json:"avgReps"`
	Sets     int     `json:"sets"`
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
		day := BerlinDayStart(ex.CreatedAt)
		day2exercises[day] = append(day2exercises[day], ex)
	}

	avgDurationPerDay := make(map[time.Time]time.Duration)
	var totalRest time.Duration
	var gaps int
	for day, dayExercises := range day2exercises {
		if len(dayExercises) < 2 {
			continue
		}

		slices.SortFunc(dayExercises, func(a, b Exercise) int {
			return a.CreatedAt.Compare(b.CreatedAt)
		})

		var dayRest time.Duration
		var dayGaps int
		for i := 1; i < len(dayExercises); i++ {
			gap := dayExercises[i].CreatedAt.Sub(dayExercises[i-1].CreatedAt)
			if gap <= 0 || gap > maxSetRest {
				continue
			}
			dayRest += gap
			dayGaps++
		}
		if dayGaps == 0 {
			continue
		}

		avgDurationPerDay[day] = dayRest / time.Duration(dayGaps)
		totalRest += dayRest
		gaps += dayGaps
	}

	var avgDuration time.Duration
	if gaps > 0 {
		avgDuration = totalRest / time.Duration(gaps)
	}

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
		day := BerlinDayStart(ex.CreatedAt)
		day2exercises[day] = append(day2exercises[day], ex)
	}

	for day, dayExercises := range day2exercises {
		var sumKilos, sumReps int
		for _, ex := range dayExercises {
			sumKilos += ex.Kilos
			sumReps += ex.Reps
		}
		n := float64(len(dayExercises))
		history.Stats[day] = ExerciseStats{
			AvgKilos: math.Round(float64(sumKilos)/n*100) / 100,
			AvgReps:  math.Round(float64(sumReps)/n*100) / 100,
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
	ctx, span := tracing.GlobalTracer.Start(ctx, "analyzer.gymstats.exercisePercentages")
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
