package test

import (
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
		urlVals.Add("id", params.ExerciseId)
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
		assert.Equal(t, "bench_dip", benchDipExType[0].ID)
		assert.Equal(t, "Bench Dip", benchDipExType[0].Name)
		assert.Equal(t, "triceps", benchDipExType[0].MuscleGroup)
	})

	s.T().Run("add exercise type", func(t *testing.T) {
		// TODO:
		t.Skip("not implemented")
	})
}
