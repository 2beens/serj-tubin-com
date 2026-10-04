package exercises_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/2beens/serjtubincom/internal/gymstats/exercises"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHandler_HandleAdd(t *testing.T) {
	ctrl := gomock.NewController(t)
	repoMock := NewMockexercisesRepo(ctrl)
	h := exercises.NewHandler(repoMock)

	tenMinutesAgo := time.Now().Add(-10 * time.Minute)
	now := time.Now()
	testEx1 := exercises.Exercise{
		ExerciseID:  "test-ex-1",
		MuscleGroup: "legs",
		Kilos:       20,
		Reps:        10,
		CreatedAt:   tenMinutesAgo,
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
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

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

	dayStart := exercises.BerlinDayStart(testEx2.CreatedAt)
	dayEnd := exercises.BerlinDayEnd(testEx2.CreatedAt)
	repoMock.EXPECT().
		ListAll(gomock.Any(), exercises.ExerciseParams{
			ExerciseID:         testEx2.ExerciseID,
			MuscleGroup:        testEx2.MuscleGroup,
			From:               &dayStart,
			To:                 &dayEnd,
			OnlyProd:           true,
			ExcludeTestingData: true,
		}).
		Return([]exercises.Exercise{testEx1, testEx2}, nil)

	repoMock.EXPECT().
		List(gomock.Any(), exercises.ListParams{
			ExerciseParams: exercises.ExerciseParams{
				From:               &dayStart,
				To:                 &dayEnd,
				OnlyProd:           true,
				ExcludeTestingData: true,
			},
			Page: 1,
			Size: 2,
		}).Return([]exercises.Exercise{testEx2, testEx1}, 2, nil)

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
	assert.Equal(t, int(now.Sub(tenMinutesAgo).Seconds()), addExerciseResponse.SecondsSincePreviousSet)
}

func TestBerlinDayEndOnDST(t *testing.T) {
	loc := exercises.TimeLocationBerlin

	// 2026-03-29 is the Europe/Berlin spring-forward day (23h).
	spring := time.Date(2026, 3, 29, 15, 0, 0, 0, loc)
	springEnd := exercises.BerlinDayEnd(spring)
	require.True(t, springEnd.Equal(time.Date(2026, 3, 30, 0, 0, 0, 0, loc)))
	assert.Equal(t, 23*time.Hour, springEnd.Sub(exercises.BerlinDayStart(spring)))

	// 2026-10-25 is the Europe/Berlin fall-back day (25h).
	fall := time.Date(2026, 10, 25, 15, 0, 0, 0, loc)
	fallEnd := exercises.BerlinDayEnd(fall)
	require.True(t, fallEnd.Equal(time.Date(2026, 10, 26, 0, 0, 0, 0, loc)))
	assert.Equal(t, 25*time.Hour, fallEnd.Sub(exercises.BerlinDayStart(fall)))
}

func TestHandler_HandleAdd_NegativeKilos(t *testing.T) {
	ctrl := gomock.NewController(t)
	h := exercises.NewHandler(NewMockexercisesRepo(ctrl))

	body, err := json.Marshal(exercises.Exercise{
		ExerciseID:  "ex",
		MuscleGroup: "legs",
		Kilos:       -1,
		Reps:        8,
	})
	require.NoError(t, err)

	rec := httptest.NewRecorder()
	req, err := http.NewRequest(http.MethodPost, "", bytes.NewReader(body))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	h.HandleAdd(rec, req)
	require.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestHandler_HandleUpdate_NotFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	repoMock := NewMockexercisesRepo(ctrl)
	h := exercises.NewHandler(repoMock)

	repoMock.EXPECT().Get(gomock.Any(), 9).Return(nil, exercises.ErrExerciseNotFound)

	body, err := json.Marshal(exercises.Exercise{
		ID:          9,
		ExerciseID:  "ex",
		MuscleGroup: "legs",
		Kilos:       10,
		Reps:        8,
	})
	require.NoError(t, err)

	rec := httptest.NewRecorder()
	req, err := http.NewRequest(http.MethodPut, "", bytes.NewReader(body))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	h.HandleUpdate(rec, req)
	require.Equal(t, http.StatusNotFound, rec.Code)
}
