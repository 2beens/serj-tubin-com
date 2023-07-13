package test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/2beens/serjtubincom/internal/gymstats/handlers"
	"github.com/2beens/serjtubincom/internal/gymstats/stats"

	"github.com/2beens/serjtubincom/internal/gymstats/repo"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func (s *IntegrationTestSuite) deleteAllExercises(ctx context.Context) {
	_, err := s.dbPool.Exec(ctx, "DELETE FROM exercise")
	require.NoError(s.T(), err)
}

func (s *IntegrationTestSuite) newExerciseRequest(
	ctx context.Context,
	exercise repo.Exercise,
) handlers.AddExerciseResponse {
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

	var addedExercise handlers.AddExerciseResponse
	require.NoError(s.T(), json.Unmarshal(respBytes, &addedExercise))

	return addedExercise
}

func (s *IntegrationTestSuite) updateExerciseRequest(
	ctx context.Context,
	exercise repo.Exercise,
) handlers.UpdateExerciseResponse {
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

	var updateResp handlers.UpdateExerciseResponse
	require.NoError(s.T(), json.Unmarshal(respBytes, &updateResp))
	return updateResp
}

func (s *IntegrationTestSuite) getExerciseHistory(ctx context.Context, exID, muscleGroup string) *stats.ExerciseHistory {
	req, err := http.NewRequestWithContext(
		ctx,
		"GET", fmt.Sprintf(
			"%s/gymstats/exercise/%s/group/%s/history",
			serverEndpoint, exID, muscleGroup,
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

	var history stats.ExerciseHistory
	require.NoError(s.T(), json.Unmarshal(respBytes, &history))

	return &history
}

func (s *IntegrationTestSuite) getExerciseRequest(ctx context.Context, id int) repo.Exercise {
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

	var exercise repo.Exercise
	require.NoError(s.T(), json.Unmarshal(respBytes, &exercise))
	return exercise
}

func (s *IntegrationTestSuite) deleteExerciseRequest(ctx context.Context, id int) handlers.DeleteExerciseResponse {
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

	var deleteResp handlers.DeleteExerciseResponse
	err = json.Unmarshal(respBytes, &deleteResp)
	require.NoError(s.T(), err)

	return deleteResp
}

func (s *IntegrationTestSuite) listExercisesRequest(ctx context.Context, params repo.ListParams) handlers.ExercisesListResponse {
	urlVals := url.Values{}
	if params.MuscleGroup != "" {
		urlVals.Add("group", params.MuscleGroup)
	}
	if params.ExerciseID != "" {
		urlVals.Add("exercise_id", params.ExerciseID)
	}
	if params.OnlyProd {
		urlVals.Add("only_prod", "true")
	}
	if params.ExcludeTestingData {
		urlVals.Add("exclude_testing_data", "true")
	}

	req, err := http.NewRequestWithContext(
		ctx,
		"GET",
		fmt.Sprintf(
			"%s/gymstats/list/page/%d/size/%d?%s",
			serverEndpoint, params.Page, params.Size, urlVals.Encode(),
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

	var exercisesPageResponse handlers.ExercisesListResponse
	require.NoError(s.T(), json.Unmarshal(respBytes, &exercisesPageResponse))

	return exercisesPageResponse
}

func (s *IntegrationTestSuite) TestGymStats() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	now := time.Now().In(time.Local)

	e1 := repo.Exercise{
		ExerciseID:  "ex1",
		MuscleGroup: "triceps",
		Kilos:       10,
		Reps:        10,
		CreatedAt:   now.Add(-time.Minute * 10),
		Metadata: map[string]string{
			"env":     "prod",
			"testing": "false",
		},
	}
	e2 := repo.Exercise{
		ExerciseID:  "ex2",
		MuscleGroup: "legs",
		Kilos:       250,
		Reps:        8,
		CreatedAt:   now.Add(-time.Minute * 5),
		Metadata: map[string]string{
			"env":     "prod",
			"testing": "false",
		},
	}
	e3 := repo.Exercise{
		ExerciseID:  "ex2",
		MuscleGroup: "legs",
		Kilos:       220,
		Reps:        12,
		CreatedAt:   now.Add(-time.Minute * 4),
		Metadata: map[string]string{
			"env":     "prod",
			"testing": "false",
		},
	}
	e4 := repo.Exercise{
		ExerciseID:  "ex3",
		MuscleGroup: "legs",
		Kilos:       210,
		Reps:        10,
		CreatedAt:   now,
		Metadata: map[string]string{
			"env":     "prod",
			"testing": "false",
		},
	}
	e5 := repo.Exercise{
		ExerciseID:  "ex2",
		MuscleGroup: "legs",
		Kilos:       510,
		Reps:        50,
		CreatedAt:   now.Add(time.Minute * 2),
		Metadata: map[string]string{
			"env":     "prod",
			"testing": "true",
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

		req, err = http.NewRequestWithContext(ctx, "GET", fmt.Sprintf("%s/gymstats/list/page/1/size/10", serverEndpoint), nil)
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

		req, err = http.NewRequestWithContext(ctx, "GET", fmt.Sprintf("%s/gymstats/list/page/1/size/10", serverEndpoint), nil)
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
		require.Len(t, s.listExercisesRequest(ctx, repo.ListParams{Page: 1, Size: 10}).Exercises, 0)

		//// now add some exercises
		addedE1 := s.newExerciseRequest(ctx, e1)
		addedE2 := s.newExerciseRequest(ctx, e2)
		addedE3 := s.newExerciseRequest(ctx, e3)
		addedE4 := s.newExerciseRequest(ctx, e4)
		addedE5 := s.newExerciseRequest(ctx, e5)
		e1.ID, e2.ID, e3.ID, e4.ID, e5.ID = addedE1.ID, addedE2.ID, addedE3.ID, addedE4.ID, addedE5.ID

		assert.Equal(t, 1, addedE1.CountToday)
		assert.Equal(t, 1, addedE2.CountToday)
		assert.Equal(t, 2, addedE3.CountToday)
		assert.Equal(t, 1, addedE4.CountToday)
		assert.Equal(t, 2, addedE5.CountToday) // testing one will be ignored, that's why 2 and not 3

		assert.Equal(t, e1.CreatedAt.Truncate(time.Second).In(time.UTC), addedE1.CreatedAt.Truncate(time.Second).In(time.UTC))
		assert.Equal(t, e2.CreatedAt.Truncate(time.Second).In(time.UTC), addedE2.CreatedAt.Truncate(time.Second).In(time.UTC))
		assert.Equal(t, e3.CreatedAt.Truncate(time.Second).In(time.UTC), addedE3.CreatedAt.Truncate(time.Second).In(time.UTC))
		assert.Equal(t, e4.CreatedAt.Truncate(time.Second).In(time.UTC), addedE4.CreatedAt.Truncate(time.Second).In(time.UTC))
		addedE1.CreatedAt = e1.CreatedAt
		addedE2.CreatedAt = e2.CreatedAt
		addedE3.CreatedAt = e3.CreatedAt
		addedE4.CreatedAt = e4.CreatedAt

		assert.Equal(t, e1, addedE1.Exercise)
		assert.Equal(t, e2, addedE2.Exercise)
		assert.Equal(t, e3, addedE3.Exercise)
		assert.Equal(t, e4, addedE4.Exercise)

		ex2history := s.getExerciseHistory(ctx, "ex2", "legs")
		assert.Len(t, ex2history.Stats, 1)
		assert.Equal(t, "ex2", ex2history.ExerciseID)
		assert.Equal(t, "legs", ex2history.MuscleGroup)
		for day, histStats := range ex2history.Stats {
			assert.Equal(t, time.Now().UTC().Truncate(24*time.Hour).Unix(), day.Unix())
			assert.Equal(t, 2, histStats.Sets)
			assert.Equal(t, 235, histStats.AvgKilos)
			assert.Equal(t, 10, histStats.AvgReps)
		}

		ex1history := s.getExerciseHistory(ctx, "ex1", "triceps")
		assert.Len(t, ex1history.Stats, 1)
		assert.Equal(t, "ex1", ex1history.ExerciseID)
		for day, histStats := range ex1history.Stats {
			assert.Equal(t, time.Now().UTC().Truncate(24*time.Hour).Unix(), day.Unix())
			assert.Equal(t, 1, histStats.Sets)
			assert.Equal(t, 10, histStats.AvgKilos)
			assert.Equal(t, 10, histStats.AvgReps)
		}

		emptyHistory := s.getExerciseHistory(ctx, "never-done-before", "triceps")
		assert.Empty(t, emptyHistory.Stats)
		assert.Equal(t, "never-done-before", emptyHistory.ExerciseID)
		assert.Equal(t, "triceps", emptyHistory.MuscleGroup)

		// the testing one will be ignored
		listExercisesResp := s.listExercisesRequest(
			ctx,
			repo.ListParams{
				ExerciseParams: repo.ExerciseParams{
					OnlyProd:           true,
					ExcludeTestingData: true,
				},
				Page: 1,
				Size: 10,
			},
		)
		assert.Len(t, listExercisesResp.Exercises, 4)
		assert.Equal(t, 4, listExercisesResp.Total)

		// the testing one will NOT be ignored
		listExercisesResp = s.listExercisesRequest(ctx, repo.ListParams{Page: 1, Size: 10})
		assert.Len(t, listExercisesResp.Exercises, 5)
		assert.Equal(t, 5, listExercisesResp.Total)

		legsEx2Resp := s.listExercisesRequest(ctx,
			repo.ListParams{
				ExerciseParams: repo.ExerciseParams{
					MuscleGroup:        "legs",
					ExerciseID:         "ex2",
					OnlyProd:           true,
					ExcludeTestingData: true,
				},
				Page: 1,
				Size: 2,
			},
		)
		assert.Len(t, legsEx2Resp.Exercises, 2)
		assert.Equal(t, 2, legsEx2Resp.Total)
		assert.Equal(t, e3.ID, legsEx2Resp.Exercises[0].ID)
		assert.Equal(t, e2.ID, legsEx2Resp.Exercises[1].ID)

		legsResp := s.listExercisesRequest(ctx,
			repo.ListParams{
				ExerciseParams: repo.ExerciseParams{
					MuscleGroup:        "legs",
					OnlyProd:           true,
					ExcludeTestingData: true,
				},
				Page: 1,
				Size: 3,
			},
		)
		assert.Len(t, legsResp.Exercises, 3)
		assert.Equal(t, 3, legsResp.Total)

		// now delete one
		deleteResp := s.deleteExerciseRequest(ctx, addedE2.ID)
		require.Equal(t, addedE2.ID, deleteResp.DeletedID)

		// now list again
		exercisesListResp := s.listExercisesRequest(ctx,
			repo.ListParams{
				ExerciseParams: repo.ExerciseParams{
					OnlyProd:           true,
					ExcludeTestingData: true,
				},
				Page: 1,
				Size: 10,
			},
		)
		require.Len(t, exercisesListResp.Exercises, 3) // sorted by created_at desc
		assert.Equal(t, exercisesListResp.Total, 3)
		assert.Equal(t, e4.ID, exercisesListResp.Exercises[0].ID)
		assert.Equal(t, e3.ID, exercisesListResp.Exercises[1].ID)
		assert.Equal(t, e1.ID, exercisesListResp.Exercises[2].ID)

		exercisesListResp = s.listExercisesRequest(ctx,
			repo.ListParams{
				ExerciseParams: repo.ExerciseParams{
					MuscleGroup:        "legs",
					OnlyProd:           true,
					ExcludeTestingData: true,
				},
				Page: 1,
				Size: 10,
			},
		)
		assert.Len(t, exercisesListResp.Exercises, 2)
		assert.Equal(t, exercisesListResp.Total, 2)
		assert.Equal(t, e4.ID, exercisesListResp.Exercises[0].ID)
		assert.Equal(t, e3.ID, exercisesListResp.Exercises[1].ID)

		// lastly, try update
		newCreatedAt := e3.CreatedAt.Add(-time.Minute * 10).In(time.UTC)
		updateResp := s.updateExerciseRequest(ctx, repo.Exercise{
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
		require.Equal(t, 0, s.listExercisesRequest(ctx, repo.ListParams{Page: 1, Size: 10}).Total)

		// add some exercises
		total := 15
		now := time.Now()
		for i := 0; i < total; i++ {
			s.newExerciseRequest(ctx, repo.Exercise{
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
		exercisesPageResp := s.listExercisesRequest(ctx, repo.ListParams{
			Page: 1,
			Size: 10,
		})
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
		exercisesPageResp = s.listExercisesRequest(ctx, repo.ListParams{
			Page: 2,
			Size: 10,
		})
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

		exercisesPageResp = s.listExercisesRequest(ctx, repo.ListParams{
			Page: 2,
			Size: 3,
		})
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

	s.T().Run("exercises page with authorization present and only prod and no testing", func(t *testing.T) {
		s.deleteAllExercises(context.Background())
		require.Equal(t, 0, s.listExercisesRequest(ctx, repo.ListParams{Page: 1, Size: 10}).Total)

		// add some exercises for stage and no test
		total := 15
		now := time.Now()
		for i := 0; i < total; i++ {
			s.newExerciseRequest(ctx, repo.Exercise{
				ExerciseID:  fmt.Sprintf("exercise-%d", i),
				MuscleGroup: "legs",
				Kilos:       rand.Intn(100),
				Reps:        rand.Intn(20),
				CreatedAt:   now.Add(-time.Minute * time.Duration(i)),
				Metadata: map[string]string{
					"testing": "false",
					"env":     "stage",
				},
			})
		}
		// add some exercises for prod and testing true
		now = time.Now()
		for i := 0; i < total; i++ {
			s.newExerciseRequest(ctx, repo.Exercise{
				ExerciseID:  fmt.Sprintf("exercise-%d", i),
				MuscleGroup: "legs",
				Kilos:       rand.Intn(100),
				Reps:        rand.Intn(20),
				CreatedAt:   now.Add(-time.Minute * time.Duration(i)),
				Metadata: map[string]string{
					"testing": "true",
					"env":     "prod",
				},
			})
		}
		// finally, add 5 exercises for real prod (no testing)
		now = time.Now()
		totalProd := 5
		for i := 0; i < totalProd; i++ {
			s.newExerciseRequest(ctx, repo.Exercise{
				ExerciseID:  fmt.Sprintf("exercise-%d", i),
				MuscleGroup: "legs",
				Kilos:       rand.Intn(100),
				Reps:        rand.Intn(20),
				CreatedAt:   now.Add(-time.Minute * time.Duration(i)),
				Metadata: map[string]string{
					"testing": "false",
					"env":     "prod",
				},
			})
		}

		// get exercises page
		exercisesPageResp := s.listExercisesRequest(ctx, repo.ListParams{
			Page: 1,
			Size: 10,
			ExerciseParams: repo.ExerciseParams{
				OnlyProd:           true,
				ExcludeTestingData: true,
			},
		})
		require.Len(t, exercisesPageResp.Exercises, totalProd)
		assert.Equal(t, totalProd, exercisesPageResp.Total)
		for i := 0; i < totalProd; i++ {
			assert.Equal(t, fmt.Sprintf("exercise-%d", i), exercisesPageResp.Exercises[i].ExerciseID)
			assert.Equal(t, "legs", exercisesPageResp.Exercises[i].MuscleGroup)
			assert.Equal(t, map[string]string{
				"testing": "false",
				"env":     "prod",
			}, exercisesPageResp.Exercises[i].Metadata)
		}

		exercisesPageResp = s.listExercisesRequest(ctx, repo.ListParams{
			Page: 2,
			Size: 2,
			ExerciseParams: repo.ExerciseParams{
				OnlyProd:           true,
				ExcludeTestingData: true,
			},
		})
		require.Len(t, exercisesPageResp.Exercises, 2)
		assert.Equal(t, totalProd, exercisesPageResp.Total)
		for i := 0; i < 2; i++ {
			assert.Equal(t, fmt.Sprintf("exercise-%d", i+2), exercisesPageResp.Exercises[i].ExerciseID)
			assert.Equal(t, "legs", exercisesPageResp.Exercises[i].MuscleGroup)
			assert.Equal(t, map[string]string{
				"testing": "false",
				"env":     "prod",
			}, exercisesPageResp.Exercises[i].Metadata)
		}
	})
}
