package internal

import (
	"context"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/2beens/serjtubincom/internal/instrumentation"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewServer_netlogBackupSocketSetup(t *testing.T) {
	server := &Server{
		instr: instrumentation.NewTestInstrumentation(),
	}

	dir, err := ioutil.TempDir("", "serj-server-unix")
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

	addr, err := server.netlogBackupSocketSetup(ctx, dir, socket)
	require.NoError(t, err)

	/////////////////
	conn, err := net.DialTimeout("unix", addr.String(), 20*time.Second)
	require.NoError(t, err)

	require.NoError(t, conn.SetDeadline(time.Now().Add(2*time.Second)))

	visitsCount := 15
	_, err = conn.Write([]byte(fmt.Sprintf("visits-count::%d", visitsCount)))
	require.NoError(t, err)

	buf := make([]byte, 1024)
	n, err := conn.Read(buf)
	require.NoError(t, err)

	msgReceived := string(buf[:n])
	assert.Equal(t, "ok", msgReceived)

	// stop unix listener
	cancel()

	// https://pkg.go.dev/github.com/prometheus/client_golang/prometheus/testutil
	counterVisitsBackups := testutil.CollectAndCount(server.instr.CounterVisitsBackups, "backend_test_server_netlog_visits_backed_up")
	assert.Equal(t, 1, counterVisitsBackups)
	assert.Equal(t, float64(visitsCount), testutil.ToFloat64(server.instr.CounterVisitsBackups))
}

func Test_panicRecoveryMiddleware_nonPanic(t *testing.T) {
	server := &Server{
		instr: instrumentation.NewTestInstrumentation(),
	}

	handler := server.panicRecoveryMiddleware()
	next := &panicRecTestHandler{}
	handlerFunc := handler(next)

	rr := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/", nil)
	handlerFunc.ServeHTTP(rr, req)

	assert.True(t, next.called)
	// panic did not happen
	assert.Equal(t, float64(0), testutil.ToFloat64(server.instr.CounterHandleRequestPanic))
}

func Test_panicRecoveryMiddleware_panic(t *testing.T) {
	server := &Server{
		instr: instrumentation.NewTestInstrumentation(),
	}

	handler := server.panicRecoveryMiddleware()
	next := &panicRecTestHandler{panic: true}
	handlerFunc := handler(next)

	rr := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/", nil)
	handlerFunc.ServeHTTP(rr, req)

	assert.True(t, next.called)
	// panic DID happen
	assert.Equal(t, float64(1), testutil.ToFloat64(server.instr.CounterHandleRequestPanic))
}

type panicRecTestHandler struct {
	panic  bool
	called bool
}

func (p *panicRecTestHandler) ServeHTTP(http.ResponseWriter, *http.Request) {
	p.called = true
	if p.panic {
		panic("YOLO")
	}
}
