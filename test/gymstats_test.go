package test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/2beens/serjtubincom/internal/gymstats"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func (s *IntegrationTestSuite) newExerciseRequest(
	ctx context.Context,
	exerciseJson []byte,
) gymstats.Exercise {
	req, err := http.NewRequestWithContext(
		ctx,
		"POST", fmt.Sprintf("%s/gymstats", serverEndpoint),
		bytes.NewReader(exerciseJson),
	)
	require.NoError(s.T(), err)
	req.Header.Set("User-Agent", "test-agent")
	req.Header.Set("Authorization", testGymStatsIOSAppSecret)
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.httpClient.Do(req)
	require.NoError(s.T(), err)
	require.Equal(s.T(), http.StatusCreated, resp.StatusCode)
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	require.NoError(s.T(), err)

	var addedExercise gymstats.Exercise
	require.NoError(s.T(), json.Unmarshal(respBytes, &addedExercise))

	return addedExercise
}

func (s *IntegrationTestSuite) updateExerciseRequest(
	ctx context.Context,
	exercise gymstats.Exercise,
) gymstats.UpdateExerciseResponse {
	exerciseJson, err := json.Marshal(exercise)
	require.NoError(s.T(), err)

	req, err := http.NewRequestWithContext(
		ctx,
		"POST", fmt.Sprintf("%s/gymstats/%d", serverEndpoint, exercise.ID),
		bytes.NewReader(exerciseJson),
	)
	require.NoError(s.T(), err)
	req.Header.Set("User-Agent", "test-agent")
	req.Header.Set("Authorization", testGymStatsIOSAppSecret)
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.httpClient.Do(req)
	require.NoError(s.T(), err)
	require.Equal(s.T(), http.StatusOK, resp.StatusCode)
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	require.NoError(s.T(), err)

	var updateResp gymstats.UpdateExerciseResponse
	require.NoError(s.T(), json.Unmarshal(respBytes, &updateResp))
	return updateResp
}

func (s *IntegrationTestSuite) getExerciseRequest(ctx context.Context, id int) gymstats.Exercise {
	req, err := http.NewRequestWithContext(
		ctx,
		"GET", fmt.Sprintf("%s/gymstats/exercise/%d", serverEndpoint, id),
		nil,
	)
	require.NoError(s.T(), err)
	req.Header.Set("User-Agent", "test-agent")
	req.Header.Set("Authorization", testGymStatsIOSAppSecret)

	resp, err := s.httpClient.Do(req)
	require.NoError(s.T(), err)
	require.Equal(s.T(), http.StatusOK, resp.StatusCode)
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	require.NoError(s.T(), err)

	var exercise gymstats.Exercise
	require.NoError(s.T(), json.Unmarshal(respBytes, &exercise))
	return exercise
}

func (s *IntegrationTestSuite) deleteExerciseRequest(ctx context.Context, id int) gymstats.DeleteExerciseResponse {
	req, err := http.NewRequestWithContext(
		ctx,
		"DELETE", fmt.Sprintf("%s/gymstats/%d", serverEndpoint, id),
		nil,
	)
	require.NoError(s.T(), err)
	req.Header.Set("User-Agent", "test-agent")
	req.Header.Set("Authorization", testGymStatsIOSAppSecret)

	resp, err := s.httpClient.Do(req)
	require.NoError(s.T(), err)
	require.Equal(s.T(), http.StatusOK, resp.StatusCode)
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	require.NoError(s.T(), err)

	var deleteResp gymstats.DeleteExerciseResponse
	err = json.Unmarshal(respBytes, &deleteResp)
	require.NoError(s.T(), err)

	return deleteResp
}

func (s *IntegrationTestSuite) listExercisesRequest(ctx context.Context) []gymstats.Exercise {
	req, err := http.NewRequestWithContext(ctx, "GET", fmt.Sprintf("%s/gymstats/list", serverEndpoint), nil)
	require.NoError(s.T(), err)
	req.Header.Set("User-Agent", "test-agent")
	req.Header.Set("Authorization", testGymStatsIOSAppSecret)

	resp, err := s.httpClient.Do(req)
	require.NoError(s.T(), err)
	require.Equal(s.T(), http.StatusOK, resp.StatusCode)
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	require.NoError(s.T(), err)

	var exercises []gymstats.Exercise
	err = json.Unmarshal(respBytes, &exercises)
	require.NoError(s.T(), err)

	return exercises
}

func (s *IntegrationTestSuite) TestGymStats() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	now := time.Now().In(time.Local)

	e1 := gymstats.Exercise{
		ExerciseID:  "ex1",
		MuscleGroup: "triceps",
		Kilos:       10,
		Reps:        10,
		CreatedAt:   now.Add(-time.Minute * 10),
	}
	e1Json, err := json.Marshal(e1)
	require.NoError(s.T(), err)
	e2 := gymstats.Exercise{
		ExerciseID:  "ex2",
		MuscleGroup: "legs",
		Kilos:       210,
		Reps:        10,
		CreatedAt:   now.Add(-time.Minute * 5),
		Metadata: map[string]string{
			"test": "true",
		},
	}
	e2Json, err := json.Marshal(e2)
	require.NoError(s.T(), err)
	e3 := gymstats.Exercise{
		ExerciseID:  "ex3",
		MuscleGroup: "legs",
		Kilos:       210,
		Reps:        10,
		CreatedAt:   now,
		Metadata: map[string]string{
			"test": "true",
			"env":  "stage",
		},
	}
	e3Json, err := json.Marshal(e3)
	require.NoError(s.T(), err)

	s.T().Run("authorization missing", func(t *testing.T) {
		req, err := http.NewRequestWithContext(
			ctx,
			"POST", fmt.Sprintf("%s/gymstats", serverEndpoint),
			bytes.NewReader(e1Json),
		)
		require.NoError(t, err)
		req.Header.Set("User-Agent", "test-agent")

		resp, err := s.httpClient.Do(req)
		require.NoError(s.T(), err)
		assert.Equal(s.T(), http.StatusUnauthorized, resp.StatusCode)
		resp.Body.Close()

		req, err = http.NewRequestWithContext(ctx, "GET", fmt.Sprintf("%s/gymstats/list", serverEndpoint), nil)
		require.NoError(s.T(), err)
		req.Header.Set("User-Agent", "test-agent")

		resp, err = s.httpClient.Do(req)
		require.NoError(s.T(), err)
		assert.Equal(s.T(), http.StatusUnauthorized, resp.StatusCode)
		resp.Body.Close()
	})

	s.T().Run("authorization present, but invalid", func(t *testing.T) {
		req, err := http.NewRequestWithContext(
			ctx,
			"POST", fmt.Sprintf("%s/gymstats", serverEndpoint),
			bytes.NewReader(e1Json),
		)
		require.NoError(t, err)
		req.Header.Set("User-Agent", "test-agent")
		req.Header.Set("Authorization", "invalid-token")

		resp, err := s.httpClient.Do(req)
		require.NoError(s.T(), err)
		assert.Equal(s.T(), http.StatusUnauthorized, resp.StatusCode)
		resp.Body.Close()

		req, err = http.NewRequestWithContext(ctx, "GET", fmt.Sprintf("%s/gymstats/list", serverEndpoint), nil)
		require.NoError(s.T(), err)
		req.Header.Set("User-Agent", "test-agent")
		req.Header.Set("Authorization", "invalid-token")

		resp, err = s.httpClient.Do(req)
		require.NoError(s.T(), err)
		assert.Equal(s.T(), http.StatusUnauthorized, resp.StatusCode)
		resp.Body.Close()
	})

	s.T().Run("authorization present", func(t *testing.T) {
		// before we add anything, no exercises should be returned
		require.Len(t, s.listExercisesRequest(ctx), 0)

		//// now add some exercises
		addedE1 := s.newExerciseRequest(ctx, e1Json)
		addedE2 := s.newExerciseRequest(ctx, e2Json)
		addedE3 := s.newExerciseRequest(ctx, e3Json)
		e1.ID, e2.ID, e3.ID = 1, 2, 3

		assert.Equal(t, e1.CreatedAt.Truncate(time.Second).In(time.UTC), addedE1.CreatedAt.Truncate(time.Second).In(time.UTC))
		assert.Equal(t, e2.CreatedAt.Truncate(time.Second).In(time.UTC), addedE2.CreatedAt.Truncate(time.Second).In(time.UTC))
		assert.Equal(t, e3.CreatedAt.Truncate(time.Second).In(time.UTC), addedE3.CreatedAt.Truncate(time.Second).In(time.UTC))
		addedE1.CreatedAt = e1.CreatedAt
		addedE2.CreatedAt = e2.CreatedAt
		addedE3.CreatedAt = e3.CreatedAt

		assert.Equal(t, e1, addedE1)
		assert.Equal(t, e2, addedE2)
		assert.Equal(t, e3, addedE3)

		assert.Len(t, s.listExercisesRequest(ctx), 3)

		// now delete one
		deleteResp := s.deleteExerciseRequest(ctx, addedE2.ID)
		require.Equal(t, addedE2.ID, deleteResp.DeletedID)

		// now list again
		exercises := s.listExercisesRequest(ctx)
		require.Len(t, exercises, 2) // sorted by created_at desc
		assert.Equal(t, e3.ID, exercises[0].ID)
		assert.Equal(t, e1.ID, exercises[1].ID)

		// lastly, try update
		newCreatedAt := e3.CreatedAt.Add(-time.Minute * 10).In(time.UTC)
		updateResp := s.updateExerciseRequest(ctx, gymstats.Exercise{
			ID:          e3.ID,
			ExerciseID:  "new-exercise-id",
			MuscleGroup: "legs",
			Kilos:       220,
			Reps:        15,
			CreatedAt:   newCreatedAt,
			Metadata: map[string]string{
				"test": "false",
				"env":  "stage",
			},
		})
		assert.Equal(t, e3.ID, updateResp.UpdatedID)

		// now assert that the update was successful
		updatedEx3 := s.getExerciseRequest(ctx, e3.ID)
		assert.Equal(t, "new-exercise-id", updatedEx3.ExerciseID)
		assert.Equal(t, "legs", updatedEx3.MuscleGroup)
		assert.Equal(t, 220, updatedEx3.Kilos)
		assert.Equal(t, 15, updatedEx3.Reps)
		assert.Equal(t,
			newCreatedAt.Truncate(time.Second),
			updatedEx3.CreatedAt.Truncate(time.Second),
		)
		assert.Equal(t, map[string]string{
			"test": "false",
			"env":  "stage",
		}, updatedEx3.Metadata)
	})
}
