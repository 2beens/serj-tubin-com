package exercises_test

import (
	"context"
	"testing"
	"time"

	"github.com/2beens/serjtubincom/internal/gymstats/exercises"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/goleak"
)

func TestMain(m *testing.M) {
	goleak.VerifyTestMain(m,
		// INFO: https://github.com/go-redis/redis/issues/1029
		goleak.IgnoreTopFunction(
			"github.com/go-redis/redis/v8/internal/pool.(*ConnPool).reaper",
		),
	)
}

func TestAnalyzer_ExerciseHistory_NoExercisesFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	repoMock := NewMockexercisesRepo(ctrl)
	analyzer := exercises.NewAnalyzer(repoMock)

	params := exercises.ExerciseParams{
		ExerciseID:         "ex",
		MuscleGroup:        "mg",
		OnlyProd:           true,
		ExcludeTestingData: true,
	}
	repoMock.EXPECT().ListAll(gomock.Any(), params).Return([]exercises.Exercise{}, nil)

	hist, err := analyzer.ExerciseHistory(context.Background(), params)
	require.NoError(t, err)
	require.NotNil(t, hist)
	assert.Empty(t, hist.Stats)
	assert.Equal(t, "ex", hist.ExerciseID)
	assert.Equal(t, "mg", hist.MuscleGroup)
}

func TestAnalyzer_ExerciseHistory(t *testing.T) {
	ctrl := gomock.NewController(t)
	repoMock := NewMockexercisesRepo(ctrl)
	analyzer := exercises.NewAnalyzer(repoMock)

	dateNow := time.Date(2021, 5, 5, 12, 0, 0, 0, time.UTC)
	dateYesterday := dateNow.AddDate(0, 0, -1)
	dateTenDaysAgo := dateNow.AddDate(0, 0, -10)

	testExercises := []exercises.Exercise{
		{
			Kilos:     20,
			Reps:      10,
			CreatedAt: dateNow,
		},
		{
			Kilos:     75,
			Reps:      10,
			CreatedAt: dateYesterday,
		},
		{
			Kilos:     80,
			Reps:      12,
			CreatedAt: dateYesterday,
		},
		{
			Kilos:     50,
			Reps:      13,
			CreatedAt: dateYesterday,
		},
		{
			Kilos:     20,
			Reps:      13,
			CreatedAt: dateTenDaysAgo,
		},
		{
			Kilos:     15,
			Reps:      13,
			CreatedAt: dateTenDaysAgo,
		},
		{
			Kilos:     10,
			Reps:      12,
			CreatedAt: dateTenDaysAgo,
		},
		{
			Kilos:     25,
			Reps:      10,
			CreatedAt: dateTenDaysAgo,
		},
		{
			Kilos:     35,
			Reps:      8,
			CreatedAt: dateTenDaysAgo,
		},
		{
			Kilos:     35,
			Reps:      8,
			CreatedAt: dateTenDaysAgo,
		},
	}

	for i := range testExercises {
		testExercises[i].ID = i + 1
		testExercises[i].MuscleGroup = "mg"
		testExercises[i].ExerciseID = "ex"
	}

	params := exercises.ExerciseParams{
		ExerciseID:         "ex",
		MuscleGroup:        "mg",
		OnlyProd:           true,
		ExcludeTestingData: true,
	}
	repoMock.EXPECT().ListAll(gomock.Any(), params).Return(testExercises, nil)

	hist, err := analyzer.ExerciseHistory(context.Background(), params)
	require.NoError(t, err)
	require.NotNil(t, hist)
	require.NotEmpty(t, hist.Stats)
	assert.Equal(t, "ex", hist.ExerciseID)
	assert.Equal(t, "mg", hist.MuscleGroup)

	dateNowStats, ok := hist.Stats[dateNow.Truncate(24*time.Hour)]
	require.True(t, ok)
	dateYesterdayStats, ok := hist.Stats[dateYesterday.Truncate(24*time.Hour)]
	require.True(t, ok)
	dateTenDaysAgoStats, ok := hist.Stats[dateTenDaysAgo.Truncate(24*time.Hour)]
	require.True(t, ok)

	assert.Equal(t, 20, dateNowStats.AvgKilos)
	assert.Equal(t, 10, dateNowStats.AvgReps)
	assert.Equal(t, 1, dateNowStats.Sets)

	assert.Equal(t, 68, dateYesterdayStats.AvgKilos)
	assert.Equal(t, 11, dateYesterdayStats.AvgReps)
	assert.Equal(t, 3, dateYesterdayStats.Sets)

	assert.Equal(t, 23, dateTenDaysAgoStats.AvgKilos)
	assert.Equal(t, 10, dateTenDaysAgoStats.AvgReps)
	assert.Equal(t, 6, dateTenDaysAgoStats.Sets)
}

func TestAnalyzer_AvgSetDuration(t *testing.T) {
	ctrl := gomock.NewController(t)
	repoMock := NewMockexercisesRepo(ctrl)
	analyzer := exercises.NewAnalyzer(repoMock)

	dateNow := time.Date(2021, 5, 5, 12, 0, 0, 0, time.UTC)
	dateYesterday := dateNow.AddDate(0, 0, -1)
	dateTenDaysAgo := dateNow.AddDate(0, 0, -10)

	testExercises := []exercises.Exercise{
		{
			Kilos:     20,
			Reps:      10,
			CreatedAt: dateNow,
		},
		{
			Kilos:     75,
			Reps:      10,
			CreatedAt: dateYesterday,
		},
		{
			Kilos:     80,
			Reps:      12,
			CreatedAt: dateYesterday.Add(2 * time.Minute),
		},
		{
			Kilos:     50,
			Reps:      13,
			CreatedAt: dateYesterday.Add(5 * time.Minute),
		},
		{
			Kilos:     20,
			Reps:      13,
			CreatedAt: dateTenDaysAgo.Add(1 * time.Minute),
		},
		{
			Kilos:     15,
			Reps:      13,
			CreatedAt: dateTenDaysAgo.Add(5 * time.Minute),
		},
		{
			Kilos:     10,
			Reps:      12,
			CreatedAt: dateTenDaysAgo.Add(6 * time.Minute),
		},
		{
			Kilos:     25,
			Reps:      10,
			CreatedAt: dateTenDaysAgo.Add(7 * time.Minute),
		},
		{
			Kilos:     35,
			Reps:      8,
			CreatedAt: dateTenDaysAgo.Add(9 * time.Minute),
		},
		{
			Kilos:     35,
			Reps:      8,
			CreatedAt: dateTenDaysAgo.Add(11 * time.Minute),
		},
	}

	for i := range testExercises {
		testExercises[i].ID = i + 1
		testExercises[i].MuscleGroup = "mg"
		testExercises[i].ExerciseID = "ex"
	}

	repoMock.EXPECT().
		ListAll(gomock.Any(), exercises.ExerciseParams{}).
		Return(testExercises, nil)

	res, err := analyzer.AvgSetDuration(context.Background(), exercises.ExerciseParams{})
	require.NoError(t, err)
	assert.Equal(t, int64(135000), res.Duration.Milliseconds())
	require.Equal(t, 2, len(res.DurationPerDay))

	avgDurationForDateYesterday, ok := res.DurationPerDay[dateYesterday.Truncate(24*time.Hour)]
	require.True(t, ok)
	assert.Equal(t, int64(150000), avgDurationForDateYesterday.Milliseconds())
	avgDurationForDateTenDaysAgo, ok := res.DurationPerDay[dateTenDaysAgo.Truncate(24*time.Hour)]
	require.True(t, ok)
	assert.Equal(t, int64(120000), avgDurationForDateTenDaysAgo.Milliseconds())
}

func TestAnalyzer_AvgSetDuration_NoExercisesFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	repoMock := NewMockexercisesRepo(ctrl)
	analyzer := exercises.NewAnalyzer(repoMock)

	repoMock.EXPECT().
		ListAll(gomock.Any(), exercises.ExerciseParams{}).
		Return([]exercises.Exercise{}, nil)

	res, err := analyzer.AvgSetDuration(context.Background(), exercises.ExerciseParams{})
	require.NoError(t, err)
	assert.Equal(t, int64(0), res.Duration.Milliseconds())
	require.Empty(t, res.DurationPerDay)
}

func TestAnalyzer_ExercisePercentages(t *testing.T) {
	ctrl := gomock.NewController(t)
	repoMock := NewMockexercisesRepo(ctrl)
	analyzer := exercises.NewAnalyzer(repoMock)

	dateNow := time.Date(2021, 5, 5, 12, 0, 0, 0, time.UTC)
	dateYesterday := dateNow.AddDate(0, 0, -1)
	dateTenDaysAgo := dateNow.AddDate(0, 0, -10)

	testExercises := []exercises.Exercise{
		{
			ExerciseID:   "ex1",
			MuscleGroup:  "mg1",
			ExerciseName: "Exercise 1",
			Kilos:        20,
			Reps:         10,
			CreatedAt:    dateNow,
			Metadata:     nil,
		},
		{
			ExerciseID:   "ex1",
			MuscleGroup:  "mg1",
			ExerciseName: "Exercise 1",
			Kilos:        75,
			Reps:         10,
			CreatedAt:    dateYesterday,
		},
		{
			ExerciseID:   "ex1",
			MuscleGroup:  "mg1",
			ExerciseName: "Exercise 1",
			Kilos:        80,
			Reps:         12,
			CreatedAt:    dateYesterday,
		},
		{
			ExerciseID:   "ex2",
			MuscleGroup:  "mg1",
			ExerciseName: "Exercise 2",
			Kilos:        50,
			Reps:         13,
			CreatedAt:    dateYesterday,
		},
		{
			ExerciseID:   "ex2",
			MuscleGroup:  "mg1",
			ExerciseName: "Exercise 2",
			Kilos:        20,
			Reps:         13,
			CreatedAt:    dateTenDaysAgo,
		},
		{
			ExerciseID:   "ex3",
			MuscleGroup:  "mg1",
			ExerciseName: "Exercise 3",
			Kilos:        20,
			Reps:         13,
			CreatedAt:    dateTenDaysAgo,
		},
	}

	repoMock.EXPECT().
		ListAll(gomock.Any(), exercises.ExerciseParams{
			MuscleGroup:        "mg2",
			OnlyProd:           true,
			ExcludeTestingData: true,
		}).
		Return(testExercises, nil)

	repoMock.EXPECT().GetExerciseTypes(gomock.Any(), exercises.GetExerciseTypesParams{
		MuscleGroup: "mg2",
	}).Return([]exercises.ExerciseType{
		{
			ExerciseID:  "ex4",
			MuscleGroup: "mg1",
			Name:        "ex4",
			Description: "desc1",
			CreatedAt:   dateNow.AddDate(0, -22, -1),
		},
	}, nil)

	res, err := analyzer.ExercisePercentages(context.Background(), "mg2", true, true)
	require.NoError(t, err)
	require.Equal(t, 4, len(res))
	assert.Equal(t, float64(50), res["ex1"].Percentage)
	assert.Equal(t, float64(33.33), res["ex2"].Percentage)
	assert.Equal(t, float64(16.66), res["ex3"].Percentage)
	assert.Equal(t, float64(0), res["ex4"].Percentage)

	assert.Equal(t, "Exercise 1", res["ex1"].ExerciseName)
	assert.Equal(t, "Exercise 2", res["ex2"].ExerciseName)
	assert.Equal(t, "Exercise 3", res["ex3"].ExerciseName)
}

func TestAnalyzer_ExercisePercentages_NoExercisesFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	repoMock := NewMockexercisesRepo(ctrl)
	analyzer := exercises.NewAnalyzer(repoMock)

	repoMock.EXPECT().
		ListAll(gomock.Any(), exercises.ExerciseParams{
			MuscleGroup:        "mg2",
			OnlyProd:           true,
			ExcludeTestingData: true,
		}).
		Return([]exercises.Exercise{}, nil)

	repoMock.EXPECT().GetExerciseTypes(gomock.Any(), exercises.GetExerciseTypesParams{
		MuscleGroup: "mg2",
	}).Return([]exercises.ExerciseType{
		{
			ExerciseID:  "ex4",
			MuscleGroup: "mg1",
			Name:        "ex4",
			Description: "desc1",
			CreatedAt:   time.Now(),
		},
	}, nil)

	res, err := analyzer.ExercisePercentages(context.Background(), "mg2", true, true)
	require.NoError(t, err)
	require.Equal(t, 1, len(res))
	assert.Equal(t, float64(0), res["ex4"].Percentage)
}
