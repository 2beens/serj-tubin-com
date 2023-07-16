package netlog

import (
	"context"
	"fmt"
	"net"
	"os"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus/testutil"
	promcl "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/2beens/serjtubincom/internal/telemetry/metrics"
	"github.com/2beens/serjtubincom/pkg"
)

func TestVisitsBackupUnixSocketListenerSetup(t *testing.T) {
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

	/////////////////
	conn, err := net.DialTimeout("unix", addr.String(), 20*time.Second)
	require.NoError(t, err)

	require.NoError(t, conn.SetDeadline(time.Now().Add(2*time.Second)))

	visitsCount := 15
	duration := 12.1234
	_, err = conn.Write([]byte(fmt.Sprintf("visits-count::%d||duration::%f", visitsCount, duration)))
	require.NoError(t, err)

	buf := make([]byte, 1024)
	n, err := conn.Read(buf)
	require.NoError(t, err)

	msgReceived := pkg.BytesToString(buf[:n])
	assert.Equal(t, "ok", msgReceived)

	// stop unix listener
	cancel()

	// https://pkg.go.dev/github.com/prometheus/client_golang/prometheus/testutil
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
	assert.Equal(t, duration, *foundHistMetric.Histogram.SampleSum)
}
