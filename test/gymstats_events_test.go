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
	"time"

	"github.com/2beens/serjtubincom/internal/gymstats/events"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func (s *IntegrationTestSuite) deleteAllEvents(ctx context.Context) {
	_, err := s.dbPool.Exec(ctx, "DELETE FROM gymstats_event")
	require.NoError(s.T(), err)
}

func (s *IntegrationTestSuite) newTrainingStartRequest(ctx context.Context, timestamp time.Time) events.TrainingStart {
	ts := events.TrainingStart{
		Timestamp: timestamp,
	}
	tsJson, err := json.Marshal(ts)
	require.NoError(s.T(), err)

	req, err := http.NewRequestWithContext(
		ctx,
		"POST", fmt.Sprintf("%s/gymstats/events/training/start", serverEndpoint),
		bytes.NewReader(tsJson),
	)
	require.NoError(s.T(), err)
	req.Header.Set("User-Agent", "GymStats/1")
	req.Header.Set("Authorization", testGymStatsIOSAppSecret)
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.httpClient.Do(req)
	require.NoError(s.T(), err)
	require.Equal(s.T(), http.StatusCreated, resp.StatusCode)
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	require.NoError(s.T(), err)

	var tsResponse events.TrainingStart
	require.NoError(s.T(), json.Unmarshal(respBytes, &tsResponse))

	return tsResponse
}

func (s *IntegrationTestSuite) newTrainingFinishRequest(ctx context.Context, tf events.TrainingFinish) events.TrainingFinish {
	tfJson, err := json.Marshal(tf)
	require.NoError(s.T(), err)

	req, err := http.NewRequestWithContext(
		ctx,
		"POST", fmt.Sprintf("%s/gymstats/events/training/finish", serverEndpoint),
		bytes.NewReader(tfJson),
	)
	require.NoError(s.T(), err)
	req.Header.Set("User-Agent", "GymStats/1")
	req.Header.Set("Authorization", testGymStatsIOSAppSecret)
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.httpClient.Do(req)
	require.NoError(s.T(), err)
	require.Equal(s.T(), http.StatusCreated, resp.StatusCode)
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	require.NoError(s.T(), err)

	var tfResponse events.TrainingFinish
	require.NoError(s.T(), json.Unmarshal(respBytes, &tfResponse))

	return tfResponse
}

func (s *IntegrationTestSuite) newWeightReportRequest(ctx context.Context, wr events.WeightReport) events.WeightReport {
	wrJson, err := json.Marshal(wr)
	require.NoError(s.T(), err)

	req, err := http.NewRequestWithContext(
		ctx,
		"POST", fmt.Sprintf("%s/gymstats/events/report/weight", serverEndpoint),
		bytes.NewReader(wrJson),
	)
	require.NoError(s.T(), err)
	req.Header.Set("User-Agent", "GymStats/1")
	req.Header.Set("Authorization", testGymStatsIOSAppSecret)
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.httpClient.Do(req)
	require.NoError(s.T(), err)
	require.Equal(s.T(), http.StatusCreated, resp.StatusCode)
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	require.NoError(s.T(), err)

	var wrResponse events.WeightReport
	require.NoError(s.T(), json.Unmarshal(respBytes, &wrResponse))

	return wrResponse
}

func (s *IntegrationTestSuite) getAllEventsRequest(ctx context.Context, params events.ListParams) events.ListResponse {
	urlVals := url.Values{}
	if params.OnlyProd {
		urlVals.Add("only_prod", "true")
	}
	if params.ExcludeTestingData {
		urlVals.Add("exclude_testing_data", "true")
	}
	if params.Type != nil {
		urlVals.Add("type", params.Type.String())
	}

	req, err := http.NewRequestWithContext(
		ctx,
		"GET",
		fmt.Sprintf(
			"%s/gymstats/events/list/page/%d/size/%d?%s",
			serverEndpoint, params.Page, params.Size, urlVals.Encode(),
		),
		nil,
	)
	require.NoError(s.T(), err)
	req.Header.Set("User-Agent", "GymStats/1")
	req.Header.Set("Authorization", testGymStatsIOSAppSecret)

	resp, err := s.httpClient.Do(req)
	require.NoError(s.T(), err)
	require.Equal(s.T(), http.StatusOK, resp.StatusCode)
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	require.NoError(s.T(), err)

	var events events.ListResponse
	require.NoError(s.T(), json.Unmarshal(respBytes, &events))

	return events
}

func (s *IntegrationTestSuite) TestGymStats_Events() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	s.T().Run("training start - finish", func(t *testing.T) {
		s.deleteAllEvents(ctx)

		equalTimeWithIgnoredTZ := func(t1, t2 time.Time) bool {
			return t1.Year() == t2.Year() &&
				t1.Month() == t2.Month() &&
				t1.Day() == t2.Day() &&
				t1.Hour() == t2.Hour() &&
				t1.Minute() == t2.Minute() &&
				t1.Second() == t2.Second()
		}

		now := time.Now()
		tsResp := s.newTrainingStartRequest(ctx, now)
		assert.True(t, equalTimeWithIgnoredTZ(now, tsResp.Timestamp))

		tfResp := s.newTrainingFinishRequest(ctx, events.TrainingFinish{
			Timestamp: now.Add(time.Hour),
			Calories:  660,
		})
		assert.True(t, equalTimeWithIgnoredTZ(now.Add(time.Hour), tfResp.Timestamp))
		require.Equal(t, 660, tfResp.Calories)

		allEvents := s.getAllEventsRequest(ctx, events.ListParams{
			EventParams: events.EventParams{
				Type:               nil,
				From:               nil,
				To:                 nil,
				OnlyProd:           false,
				ExcludeTestingData: false,
			},
			Page: 1,
			Size: 10,
		})

		assert.Len(t, allEvents.Events, 2)
		assert.Equal(t, 2, allEvents.Total)
	})

	s.T().Run("list events", func(t *testing.T) {
		s.deleteAllEvents(ctx)

		now := time.Now()
		_ = s.newTrainingStartRequest(ctx, now)
		_ = s.newTrainingFinishRequest(ctx, events.TrainingFinish{
			Timestamp: now.Add(time.Hour),
			Calories:  660,
		})
		_ = s.newTrainingStartRequest(ctx, now)
		_ = s.newTrainingFinishRequest(ctx, events.TrainingFinish{
			Timestamp: now.Add(time.Hour),
			Calories:  760,
		})

		for i := range 20 {
			s.newWeightReportRequest(ctx, events.WeightReport{
				Timestamp: now.Add(time.Duration(i) * time.Minute),
				Weight:    10 * i,
			})
		}

		allEvents := s.getAllEventsRequest(ctx, events.ListParams{
			EventParams: events.EventParams{
				Type:               nil,
				From:               nil,
				To:                 nil,
				OnlyProd:           false,
				ExcludeTestingData: false,
			},
			Page: 1,
			Size: 10,
		})
		assert.Len(t, allEvents.Events, 10)
		assert.Equal(t, 24, allEvents.Total)

		tsEvent := events.EventTypeTrainingStarted
		trStartEvents := s.getAllEventsRequest(ctx, events.ListParams{
			EventParams: events.EventParams{
				Type:               &tsEvent,
				From:               nil,
				To:                 nil,
				OnlyProd:           false,
				ExcludeTestingData: false,
			},
			Page: 1,
			Size: 10,
		})
		require.Len(t, trStartEvents.Events, 2)
		assert.Equal(t, 2, trStartEvents.Total)
		assert.Equal(t, tsEvent, trStartEvents.Events[0].Type)
		assert.Equal(t, tsEvent, trStartEvents.Events[1].Type)

		tfEvent := events.EventTypeTrainingFinished
		trFinishEvents := s.getAllEventsRequest(ctx, events.ListParams{
			EventParams: events.EventParams{
				Type:               &tfEvent,
				From:               nil,
				To:                 nil,
				OnlyProd:           false,
				ExcludeTestingData: false,
			},
			Page: 1,
			Size: 10,
		})
		require.Len(t, trFinishEvents.Events, 2)
		assert.Equal(t, 2, trFinishEvents.Total)
		assert.Equal(t, tfEvent, trFinishEvents.Events[0].Type)
		require.Len(t, trFinishEvents.Events[0].Data, 1)
		require.Equal(t, "660", trFinishEvents.Events[0].Data["calories"])
		assert.Equal(t, tfEvent, trFinishEvents.Events[1].Type)
		require.Len(t, trFinishEvents.Events[1].Data, 1)
		require.Equal(t, "760", trFinishEvents.Events[1].Data["calories"])

		wrEvent := events.EventTypeWeightReport
		wrEvents := s.getAllEventsRequest(ctx, events.ListParams{
			EventParams: events.EventParams{
				Type:               &wrEvent,
				OnlyProd:           false,
				ExcludeTestingData: false,
			},
			Page: 2,
			Size: 5,
		})
		require.Len(t, wrEvents.Events, 5)
		assert.Equal(t, 20, wrEvents.Total)
	})
}
