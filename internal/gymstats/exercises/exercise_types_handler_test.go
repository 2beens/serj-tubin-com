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

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestTypesHandler_HandleAdd(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockExerciseTypesRepo := NewMockexerciseTypesRepo(ctrl)
	mockDiskApi := NewMockdiskApi(ctrl)

	handler := exercises.NewTypesHandler(mockDiskApi, mockExerciseTypesRepo)

	exerciseType := exercises.ExerciseType{
		ID:          "100",
		MuscleGroup: "biceps",
		Name:        "curl",
		Description: "some desc",
	}
	exerciseTypeJson, err := json.Marshal(exerciseType)
	require.NoError(t, err)

	mockExerciseTypesRepo.EXPECT().
		AddExerciseType(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, exType exercises.ExerciseType) error {
			assert.Equal(t, exerciseType.ID, exType.ID)
			assert.Equal(t, exerciseType.MuscleGroup, exType.MuscleGroup)
			assert.Equal(t, exerciseType.Name, exType.Name)
			assert.Equal(t, exerciseType.Description, exType.Description)
			assert.True(t, time.Now().Sub(exType.CreatedAt) < time.Minute)
			return nil
		})

	rr := httptest.NewRecorder()
	req, err := http.NewRequest("POST", "/", bytes.NewBuffer(exerciseTypeJson))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	handler.HandleAdd(rr, req)
	require.Equal(t, http.StatusCreated, rr.Code)
}
