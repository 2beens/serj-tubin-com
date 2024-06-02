package events_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/2beens/serjtubincom/internal/gymstats/events"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHandler_HandleTrainingStart(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockService := NewMockservice(ctrl)
	h := events.NewHandler(mockService)

	now := time.Now().UTC().Truncate(time.Second)
	trainingStart := events.TrainingStart{
		Timestamp: now,
	}
	tsJson, err := json.Marshal(trainingStart)
	require.NoError(t, err)

	req, err := http.NewRequest("POST", "/", bytes.NewBuffer(tsJson))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	handlerFunc := http.HandlerFunc(h.HandleTrainingStart)

	mockService.EXPECT().
		AddTrainingStart(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ any, ts events.TrainingStart) (int, error) {
			assert.Equal(t, now, ts.Timestamp)
			return 1, nil
		})

	// Call the HandlerFunc
	handlerFunc.ServeHTTP(rr, req)
	require.Equal(t, http.StatusCreated, rr.Code)

	var trainingStartResp events.TrainingStart
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &trainingStartResp))
	assert.Equal(t, now, trainingStartResp.Timestamp)
}

func TestHandler_HandleTrainingFinished(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockService := NewMockservice(ctrl)
	h := events.NewHandler(mockService)

	now := time.Now().UTC().Truncate(time.Second)
	trainingEnd := events.TrainingFinish{
		Timestamp: now,
		Calories:  100,
	}
	teJson, err := json.Marshal(trainingEnd)
	require.NoError(t, err)

	req, err := http.NewRequest("POST", "/", bytes.NewBuffer(teJson))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	handlerFunc := http.HandlerFunc(h.HandleTrainingFinished)

	mockService.EXPECT().
		AddTrainingFinish(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ any, tf events.TrainingFinish) (int, error) {
			assert.Equal(t, now, tf.Timestamp)
			assert.Equal(t, 100, tf.Calories)
			return 1, nil
		})

	// Call the HandlerFunc
	handlerFunc.ServeHTTP(rr, req)
	require.Equal(t, http.StatusCreated, rr.Code)

	var trainingEndResp events.TrainingFinish
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &trainingEndResp))
	assert.Equal(t, now, trainingEndResp.Timestamp)
	assert.Equal(t, 100, trainingEndResp.Calories)
}
