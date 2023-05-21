package test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"testing"
	"time"

	"github.com/2beens/serjtubincom/internal/gymstats"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func (s *IntegrationTestSuite) deleteAllExercises(ctx context.Context) {
	_, err := s.dbPool.Exec(ctx, "DELETE FROM exercise")
	require.NoError(s.T(), err)
}

func (s *IntegrationTestSuite) newExerciseRequest(
	ctx context.Context,
	exercise gymstats.Exercise,
) gymstats.Exercise {
	exerciseJson, err := json.Marshal(exercise)
	require.NoError(s.T(), err)

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

func (s *IntegrationTestSuite) getExercisesPageRequest(ctx context.Context, page, size int) gymstats.ExercisesPageResponse {
	req, err := http.NewRequestWithContext(
		ctx,
		"GET",
		fmt.Sprintf("%s/gymstats/page/%d/size/%d",
			serverEndpoint, page, size,
		),
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

	var exercisesPageResponse gymstats.ExercisesPageResponse
	err = json.Unmarshal(respBytes, &exercisesPageResponse)
	require.NoError(s.T(), err)

	return exercisesPageResponse
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

	s.T().Run("authorization missing", func(t *testing.T) {
		e1Json, err := json.Marshal(e1)
		require.NoError(t, err)
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
		e1Json, err := json.Marshal(e1)
		require.NoError(t, err)
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
		s.deleteAllExercises(context.Background())
		// before we add anything, no exercises should be returned
		require.Len(t, s.listExercisesRequest(ctx), 0)

		//// now add some exercises
		addedE1 := s.newExerciseRequest(ctx, e1)
		addedE2 := s.newExerciseRequest(ctx, e2)
		addedE3 := s.newExerciseRequest(ctx, e3)
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

	s.T().Run("exercises page with authorization present", func(t *testing.T) {
		s.deleteAllExercises(context.Background())
		require.Len(t, s.listExercisesRequest(ctx), 0)

		// add some exercises
		total := 15
		now := time.Now()
		for i := 0; i < total; i++ {
			s.newExerciseRequest(ctx, gymstats.Exercise{
				ExerciseID:  fmt.Sprintf("exercise-%d", i),
				MuscleGroup: "legs",
				Kilos:       rand.Intn(100),
				Reps:        rand.Intn(20),
				CreatedAt:   now.Add(-time.Minute * time.Duration(i)),
				Metadata: map[string]string{
					"test": "false",
					"env":  "stage",
				},
			})
		}

		// get exercises page
		exercisesPageResp := s.getExercisesPageRequest(ctx, 1, 10)
		require.Len(t, exercisesPageResp.Exercises, 10)
		assert.Equal(t, total, exercisesPageResp.Total)
		for i := 0; i < 10; i++ {
			assert.Equal(t, fmt.Sprintf("exercise-%d", i), exercisesPageResp.Exercises[i].ExerciseID)
			assert.Equal(t, "legs", exercisesPageResp.Exercises[i].MuscleGroup)
			assert.Equal(t, map[string]string{
				"test": "false",
				"env":  "stage",
			}, exercisesPageResp.Exercises[i].Metadata)
		}

		// will move the offset from 10 to 5, and get last 10
		exercisesPageResp = s.getExercisesPageRequest(ctx, 2, 10)
		require.Len(t, exercisesPageResp.Exercises, 10)
		assert.Equal(t, total, exercisesPageResp.Total)
		for i := 0; i < 10; i++ {
			assert.Equal(t, fmt.Sprintf("exercise-%d", i+5), exercisesPageResp.Exercises[i].ExerciseID)
			assert.Equal(t, "legs", exercisesPageResp.Exercises[i].MuscleGroup)
			assert.Equal(t, map[string]string{
				"test": "false",
				"env":  "stage",
			}, exercisesPageResp.Exercises[i].Metadata)
		}

		exercisesPageResp = s.getExercisesPageRequest(ctx, 2, 3)
		require.Len(t, exercisesPageResp.Exercises, 3)
		assert.Equal(t, total, exercisesPageResp.Total)
		for i := 0; i < 3; i++ {
			assert.Equal(t, fmt.Sprintf("exercise-%d", i+3), exercisesPageResp.Exercises[i].ExerciseID)
			assert.Equal(t, "legs", exercisesPageResp.Exercises[i].MuscleGroup)
			assert.Equal(t, map[string]string{
				"test": "false",
				"env":  "stage",
			}, exercisesPageResp.Exercises[i].Metadata)
		}
	})
}
