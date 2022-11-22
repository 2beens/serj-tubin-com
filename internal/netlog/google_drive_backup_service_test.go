package netlog

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/2beens/serjtubincom/internal/metrics"
	"github.com/prometheus/client_golang/prometheus/testutil"
	promcl "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_trySendMetrics(t *testing.T) {
	metrics, reg := metrics.NewTestManagerAndRegistry()
	dir, err := os.MkdirTemp("", "serj-server-unix")
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if rErr := os.RemoveAll(dir); rErr != nil {
			t.Error(rErr)
		}
	}()

	ctx, cancel := context.WithCancel(context.Background())
	socket := fmt.Sprintf("%d.sock", os.Getpid())

	addr, err := VisitsBackupUnixSocketListenerSetup(ctx, dir, socket, metrics)
	require.NoError(t, err)
	require.NotEmpty(t, addr)

	beginTimestamp := time.Now().Add(-time.Second)
	visitsCount := 100

	// MAIN TESTED FUNCTION
	trySendMetrics(beginTimestamp, visitsCount, dir, socket)

	// stop unix listener
	cancel()

	counterVisitsBackups := testutil.CollectAndCount(metrics.CounterVisitsBackups, "backend_test_server_netlog_visits_backed_up")
	histNetlogBackupDuration, err := testutil.GatherAndCount(reg, "backend_test_server_netlog_backup_duration_seconds")
	require.NoError(t, err)
	assert.Equal(t, 1, counterVisitsBackups)
	assert.Equal(t, 1, histNetlogBackupDuration)
	assert.Equal(t, float64(visitsCount), testutil.ToFloat64(metrics.CounterVisitsBackups))

	require.NotNil(t, reg)
	gathered, err := reg.Gather()
	require.NoError(t, err)
	require.NotNil(t, gathered)

	var foundDurationHistogram *promcl.MetricFamily
	for _, m := range gathered {
		if *m.Name == "backend_test_server_netlog_backup_duration_seconds" {
			foundDurationHistogram = m
			break
		}
	}
	if foundDurationHistogram == nil {
		t.Fatal("found duration histogram is nil")
	}

	require.NotNil(t, foundDurationHistogram.Metric)
	require.Len(t, foundDurationHistogram.Metric, 1)
	foundHistMetric := foundDurationHistogram.Metric[0]
	require.NotNil(t, foundHistMetric)
	require.NotNil(t, foundHistMetric.Histogram)
	// duration [d] is: 1 <= d < 2
	assert.GreaterOrEqual(t, *foundHistMetric.Histogram.SampleSum, float64(1))
	assert.Less(t, *foundHistMetric.Histogram.SampleSum, float64(2))
}
