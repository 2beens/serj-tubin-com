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
	require.Equal(s.T(), http.StatusOK, resp.StatusCode)
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	require.NoError(s.T(), err)

	var exerciseTypes []exercises.ExerciseType
	require.NoError(s.T(), json.Unmarshal(respBytes, &exerciseTypes))

	return exerciseTypes
}

func (s *IntegrationTestSuite) TestGymStats_ExerciseTypes() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	s.T().Run("get all", func(t *testing.T) {
		token := s.doLogin(ctx)
		exerciseTypes := s.getAllExerciseTypesRequest(ctx, token, exercises.GetExerciseTypesParams{})
		require.NotEmpty(t, exerciseTypes)
		assert.Len(t, exerciseTypes, 86)
	})

	s.T().Run("add exercise type", func(t *testing.T) {
		// TODO:
		t.Skip("not implemented")
	})
}
