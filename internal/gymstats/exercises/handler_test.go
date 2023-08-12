package exercises_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/2beens/serjtubincom/internal/gymstats/exercises"
)

func TestHandler_HandleAdd(t *testing.T) {
	ctrl := gomock.NewController(t)
	repoMock := NewMockexercisesRepo(ctrl)
	h := exercises.NewHandler(repoMock)

	now := time.Now()
	testEx1 := exercises.Exercise{
		ExerciseID:  "test-ex-1",
		MuscleGroup: "legs",
		Kilos:       20,
		Reps:        10,
		CreatedAt:   now.Add(-10 * time.Minute),
		Metadata: map[string]string{
			"testKey": "test-val",
		},
	}

	testEx2 := exercises.Exercise{
		ExerciseID:  "test-ex-1",
		MuscleGroup: "legs",
		Kilos:       25,
		Reps:        8,
		CreatedAt:   now,
		Metadata: map[string]string{
			"testKey": "test-val",
		},
	}

	testExJson, err := json.Marshal(testEx2)
	require.NoError(t, err)

	rec := httptest.NewRecorder()
	req, err := http.NewRequest("POST", "", bytes.NewReader(testExJson))
	req.Header.Set("Content-Type", "application/json")
	require.NoError(t, err)

	repoMock.EXPECT().
		Add(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, ex exercises.Exercise) (*exercises.Exercise, error) {
			assert.Equal(t, testEx2.ExerciseID, ex.ExerciseID)
			assert.Equal(t, testEx2.MuscleGroup, ex.MuscleGroup)
			assert.Equal(t, testEx2.Kilos, ex.Kilos)
			assert.Equal(t, testEx2.Reps, ex.Reps)
			assert.Equal(t,
				testEx2.CreatedAt.Truncate(time.Second).Unix(),
				ex.CreatedAt.Truncate(time.Second).Unix(),
			)
			assert.Equal(t, testEx2.Metadata, ex.Metadata)
			return &exercises.Exercise{
				ID:          2,
				ExerciseID:  testEx2.ExerciseID,
				MuscleGroup: testEx2.MuscleGroup,
				Kilos:       testEx2.Kilos,
				Reps:        testEx2.Reps,
				CreatedAt:   testEx2.CreatedAt,
				Metadata:    testEx2.Metadata,
			}, nil
		}).Times(1)

	todayMidnight := time.Now().Truncate(24 * time.Hour)
	tomorrowMidnight := todayMidnight.Add(24 * time.Hour)
	repoMock.EXPECT().
		ListAll(gomock.Any(), exercises.ExerciseParams{
			ExerciseID:         testEx2.ExerciseID,
			MuscleGroup:        testEx2.MuscleGroup,
			From:               &todayMidnight,
			To:                 &tomorrowMidnight,
			OnlyProd:           true,
			ExcludeTestingData: true,
		}).
		Return([]exercises.Exercise{testEx1, testEx2}, nil)

	h.HandleAdd(rec, req)
	require.Equal(t, http.StatusCreated, rec.Code)

	var addExerciseResponse exercises.AddExerciseResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &addExerciseResponse))
	assert.Equal(t, 2, addExerciseResponse.ID)
	assert.Equal(t, testEx2.ExerciseID, addExerciseResponse.ExerciseID)
	assert.Equal(t, testEx2.MuscleGroup, addExerciseResponse.MuscleGroup)
	assert.Equal(t, testEx2.Kilos, addExerciseResponse.Kilos)
	assert.Equal(t, testEx2.Reps, addExerciseResponse.Reps)
	assert.Equal(t,
		testEx2.CreatedAt.Truncate(time.Second).Unix(),
		addExerciseResponse.CreatedAt.Truncate(time.Second).Unix(),
	)
	assert.Equal(t, testEx2.Metadata, addExerciseResponse.Metadata)
	assert.Equal(t, 2, addExerciseResponse.CountToday)
}
