package test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"testing"

	"github.com/2beens/serjtubincom/internal/gymstats/exercises"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func (s *IntegrationTestSuite) addExerciseTypeRequest(
	ctx context.Context,
	authToken string,
	exType exercises.ExerciseType,
	expectedStatusCode int,
) {
	exTypeBytes, err := json.Marshal(exType)
	require.NoError(s.T(), err)

	req, err := http.NewRequestWithContext(
		ctx,
		"POST",
		fmt.Sprintf("%s/gymstats/types", serverEndpoint),
		bytes.NewReader(exTypeBytes),
	)
	require.NoError(s.T(), err)
	req.Header.Set("User-Agent", "test-agent")
	req.Header.Set("X-SERJ-TOKEN", authToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.httpClient.Do(req)
	require.NoError(s.T(), err)
	if resp.StatusCode != expectedStatusCode {
		respBytes, err := io.ReadAll(resp.Body)
		require.NoError(s.T(), err)
		fmt.Printf(
			"unexpected status code: %d, response body: %s\n",
			resp.StatusCode, string(respBytes),
		)
		require.Equal(s.T(), expectedStatusCode, resp.StatusCode)
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	require.NoError(s.T(), err)

	fmt.Printf("add ex. type, response body: [%s]\n", string(respBytes))
}

func (s *IntegrationTestSuite) getAllExerciseTypesRequest(
	ctx context.Context,
	authToken string,
	params exercises.GetExerciseTypesParams,
	expectedStatusCode int,
) []exercises.ExerciseType {
	urlVals := url.Values{}
	if params.MuscleGroup != "" {
		urlVals.Add("muscleGroup", params.MuscleGroup)
	}
	if params.ExerciseId != "" {
		urlVals.Add("exerciseId", params.ExerciseId)
	}

	req, err := http.NewRequestWithContext(
		ctx,
		"GET",
		fmt.Sprintf(
			"%s/gymstats/types?%s",
			serverEndpoint, urlVals.Encode(),
		),
		nil,
	)
	require.NoError(s.T(), err)
	req.Header.Set("User-Agent", "test-agent")
	req.Header.Set("X-SERJ-TOKEN", authToken)

	resp, err := s.httpClient.Do(req)
	require.NoError(s.T(), err)
	require.Equal(s.T(), expectedStatusCode, resp.StatusCode)
	defer resp.Body.Close()

	if expectedStatusCode >= 300 {
		return []exercises.ExerciseType{}
	}

	respBytes, err := io.ReadAll(resp.Body)
	require.NoError(s.T(), err)

	var exerciseTypes []exercises.ExerciseType
	require.NoError(s.T(), json.Unmarshal(respBytes, &exerciseTypes))

	return exerciseTypes
}

func (s *IntegrationTestSuite) TestGymStats_ExerciseTypes() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	authToken := s.doLogin(ctx)

	s.T().Run("get all", func(t *testing.T) {
		exerciseTypes := s.getAllExerciseTypesRequest(
			ctx, "invalid-token",
			exercises.GetExerciseTypesParams{}, http.StatusUnauthorized,
		)
		assert.Len(t, exerciseTypes, 0)

		exerciseTypes = s.getAllExerciseTypesRequest(ctx, authToken, exercises.GetExerciseTypesParams{}, http.StatusOK)
		assert.Len(t, exerciseTypes, 68)

		bicepsExTypes := s.getAllExerciseTypesRequest(ctx, authToken,
			exercises.GetExerciseTypesParams{
				MuscleGroup: "biceps",
			},
			http.StatusOK,
		)
		assert.Len(t, bicepsExTypes, 8)

		benchDipExType := s.getAllExerciseTypesRequest(ctx, authToken,
			exercises.GetExerciseTypesParams{
				MuscleGroup: "triceps",
				ExerciseId:  "bench_dip",
			},
			http.StatusOK,
		)
		require.Len(t, benchDipExType, 1)
		assert.Equal(t, "bench_dip", benchDipExType[0].ExerciseID)
		assert.Equal(t, "Bench Dip", benchDipExType[0].Name)
		assert.Equal(t, "triceps", benchDipExType[0].MuscleGroup)
	})

	s.T().Run("add exercise type", func(t *testing.T) {
		newExType := exercises.ExerciseType{
			ExerciseID:  "some_id",
			MuscleGroup: "legs",
			Name:        "Some Ex1",
			Description: "Some Ex1 description",
		}

		s.addExerciseTypeRequest(ctx, authToken, newExType, http.StatusCreated)

		// try to get it now
		addedExType := s.getAllExerciseTypesRequest(ctx, authToken,
			exercises.GetExerciseTypesParams{
				MuscleGroup: newExType.MuscleGroup,
				ExerciseId:  newExType.ExerciseID,
			},
			http.StatusOK,
		)
		require.Len(t, addedExType, 1)
		assert.Equal(t, newExType.ExerciseID, addedExType[0].ExerciseID)
		assert.Equal(t, newExType.MuscleGroup, addedExType[0].MuscleGroup)
		assert.Equal(t, newExType.Name, addedExType[0].Name)
		assert.Equal(t, newExType.Description, addedExType[0].Description)
		assert.False(t, addedExType[0].CreatedAt.IsZero())

		// try to add the same exercise type again
		s.addExerciseTypeRequest(ctx, authToken, newExType, http.StatusConflict)

		// same id but different group
		newExType2 := exercises.ExerciseType{
			ExerciseID:  "some_id",
			MuscleGroup: "shoulders",
			Name:        "Some Ex2",
			Description: "Some Ex2 description",
		}
		s.addExerciseTypeRequest(ctx, authToken, newExType2, http.StatusCreated)

		// try to get it now
		addedExType2 := s.getAllExerciseTypesRequest(ctx, authToken,
			exercises.GetExerciseTypesParams{
				MuscleGroup: newExType2.MuscleGroup,
				ExerciseId:  newExType2.ExerciseID,
			},
			http.StatusOK,
		)
		require.Len(t, addedExType2, 1)
		assert.Equal(t, newExType2.ExerciseID, addedExType2[0].ExerciseID)
		assert.Equal(t, newExType2.MuscleGroup, addedExType2[0].MuscleGroup)
		assert.Equal(t, newExType2.Name, addedExType2[0].Name)
		assert.Equal(t, newExType2.Description, addedExType2[0].Description)
		assert.False(t, addedExType2[0].CreatedAt.IsZero())

		// try to add the same exercise type again
		s.addExerciseTypeRequest(ctx, authToken, exercises.ExerciseType{
			ExerciseID:  "some_id1",
			MuscleGroup: "other",
			Name:        "Some Ex1",
			Description: "Some Ex1 description",
		}, http.StatusCreated)
		s.addExerciseTypeRequest(ctx, authToken, exercises.ExerciseType{
			ExerciseID:  "some_id2",
			MuscleGroup: "other",
			Name:        "Some Ex2",
			Description: "Some Ex2 description",
		}, http.StatusCreated)

		bicepsExTypes := s.getAllExerciseTypesRequest(ctx, authToken,
			exercises.GetExerciseTypesParams{
				MuscleGroup: "other",
			},
			http.StatusOK,
		)
		assert.Len(t, bicepsExTypes, 13)

		bicepsExTypes = s.getAllExerciseTypesRequest(ctx, authToken,
			exercises.GetExerciseTypesParams{
				ExerciseId: "some_id",
			},
			http.StatusOK,
		)
		assert.Len(t, bicepsExTypes, 2)
	})
}
