package gymstats_test

import (
	"context"
	"testing"
	"time"

	"github.com/2beens/serjtubincom/internal/gymstats"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAnalyzer_ExerciseHistory_NoExercisesFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	repoMock := NewMockexercisesRepo(ctrl)
	analyzer := gymstats.NewAnalyzer(repoMock)

	repoMock.EXPECT().ListAll(gomock.Any(), gymstats.ExerciseParams{
		ExerciseID:  "ex",
		MuscleGroup: "mg",
	}).Return([]gymstats.Exercise{}, nil)

	hist, err := analyzer.ExerciseHistory(context.Background(), "ex", "mg")
	require.NoError(t, err)
	require.NotNil(t, hist)
	assert.Empty(t, hist.Stats)
	assert.Equal(t, "ex", hist.ExerciseID)
	assert.Equal(t, "mg", hist.MuscleGroup)
}

func TestAnalyzer_ExerciseHistory(t *testing.T) {
	ctrl := gomock.NewController(t)
	repoMock := NewMockexercisesRepo(ctrl)
	analyzer := gymstats.NewAnalyzer(repoMock)

	dateNow := time.Date(2021, 5, 5, 12, 0, 0, 0, time.UTC)
	dateYesterday := dateNow.AddDate(0, 0, -1)
	dateTenDaysAgo := dateNow.AddDate(0, 0, -10)

	exercises := []gymstats.Exercise{
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

	for i := range exercises {
		exercises[i].ID = i + 1
		exercises[i].MuscleGroup = "mg"
		exercises[i].ExerciseID = "ex"
	}

	repoMock.EXPECT().ListAll(gomock.Any(), gymstats.ExerciseParams{
		ExerciseID:  "ex",
		MuscleGroup: "mg",
	}).Return(exercises, nil)

	hist, err := analyzer.ExerciseHistory(context.Background(), "ex", "mg")
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
	assert.Equal(t, 68, dateYesterdayStats.AvgKilos)
	assert.Equal(t, 11, dateYesterdayStats.AvgReps)
	assert.Equal(t, 23, dateTenDaysAgoStats.AvgKilos)
	assert.Equal(t, 10, dateTenDaysAgoStats.AvgReps)
}