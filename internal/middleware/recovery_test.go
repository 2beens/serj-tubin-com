package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/2beens/serjtubincom/internal/instrumentation"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
)

func Test_panicRecoveryMiddleware_nonPanic(t *testing.T) {
	instr := instrumentation.NewTestInstrumentation()

	handler := PanicRecovery(instr)
	next := &panicRecTestHandler{}
	handlerFunc := handler(next)

	rr := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/", nil)
	handlerFunc.ServeHTTP(rr, req)

	assert.True(t, next.called)
	// panic did not happen
	assert.Equal(t, float64(0), testutil.ToFloat64(instr.CounterHandleRequestPanic))
}

func Test_panicRecoveryMiddleware_panic(t *testing.T) {
	instr := instrumentation.NewTestInstrumentation()

	handler := PanicRecovery(instr)
	next := &panicRecTestHandler{panic: true}
	handlerFunc := handler(next)

	rr := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/", nil)
	handlerFunc.ServeHTTP(rr, req)

	assert.True(t, next.called)
	// panic DID happen
	assert.Equal(t, float64(1), testutil.ToFloat64(instr.CounterHandleRequestPanic))
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
