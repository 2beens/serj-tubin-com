package mcp

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/2beens/serjtubincom/internal/gymstats/exercises"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// mockContextService implements contextService for tests.
type mockContextService struct {
	schema        string
	schemaErr     error
	list          []exercises.Exercise
	listErr       error
	exerciseTypes []exercises.ExerciseType
	typesErr      error
}

func (m *mockContextService) GetSchema(ctx context.Context) (string, error) {
	return m.schema, m.schemaErr
}

func (m *mockContextService) ListExercises(ctx context.Context, params exercises.ExerciseParams) ([]exercises.Exercise, error) {
	return m.list, m.listErr
}

func (m *mockContextService) GetExerciseTypes(ctx context.Context, params exercises.GetExerciseTypesParams) ([]exercises.ExerciseType, error) {
	return m.exerciseTypes, m.typesErr
}

// Tests for GetGymstatsContextTool.
func TestHandler_GetGymstatsContextTool(t *testing.T) {
	t.Run("returns_schema", func(t *testing.T) {
		want := "## exercise\n| col | type |\n"
		svc := &mockContextService{schema: want}
		h := NewHandler(svc)
		fn := h.GetGymstatsContextTool()
		res, _, err := fn(context.Background(), &mcp.CallToolRequest{}, nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if res.IsError {
			t.Fatalf("unexpected IsError")
		}
		if len(res.Content) != 1 {
			t.Fatalf("expected 1 content, got %d", len(res.Content))
		}
		if tc, ok := res.Content[0].(*mcp.TextContent); !ok || tc.Text != want {
			t.Fatalf("content text = %q, want %q", tc.Text, want)
		}
	})

	t.Run("returns_error_when_schema_fails", func(t *testing.T) {
		svc := &mockContextService{schemaErr: errors.New("db gone")}
		h := NewHandler(svc)
		fn := h.GetGymstatsContextTool()
		res, _, err := fn(context.Background(), &mcp.CallToolRequest{}, nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !res.IsError {
			t.Fatalf("expected IsError")
		}
		tc := res.Content[0].(*mcp.TextContent)
		if tc.Text != "Error fetching schema: db gone" {
			t.Fatalf("content text = %q", tc.Text)
		}
	})
}

// Tests for GetExercisesForTimeRangeTool.
func TestHandler_GetExercisesForTimeRangeTool(t *testing.T) {
	t.Run("invalid_from_date", func(t *testing.T) {
		h := NewHandler(&mockContextService{})
		fn := h.GetExercisesForTimeRangeTool()
		res, _, err := fn(context.Background(), &mcp.CallToolRequest{}, ExercisesTimeRangeInput{
			FromDate: "bad",
			ToDate:   "2025-01-15",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !res.IsError {
			t.Fatalf("expected IsError")
		}
		tc := res.Content[0].(*mcp.TextContent)
		if tc.Text != "Invalid from_date: use YYYY-MM-DD" {
			t.Fatalf("content text = %q", tc.Text)
		}
	})

	t.Run("invalid_to_date", func(t *testing.T) {
		h := NewHandler(&mockContextService{})
		fn := h.GetExercisesForTimeRangeTool()
		res, _, err := fn(context.Background(), &mcp.CallToolRequest{}, ExercisesTimeRangeInput{
			FromDate: "2025-01-01",
			ToDate:   "bad",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !res.IsError {
			t.Fatalf("expected IsError")
		}
		tc := res.Content[0].(*mcp.TextContent)
		if tc.Text != "Invalid to_date: use YYYY-MM-DD" {
			t.Fatalf("content text = %q", tc.Text)
		}
	})

	t.Run("returns_exercises", func(t *testing.T) {
		now := time.Now()
		list := []exercises.Exercise{
			{ID: 1, ExerciseID: "bp", Kilos: 80, Reps: 10, CreatedAt: now},
		}
		svc := &mockContextService{list: list}
		h := NewHandler(svc)
		fn := h.GetExercisesForTimeRangeTool()
		res, _, err := fn(context.Background(), &mcp.CallToolRequest{}, ExercisesTimeRangeInput{
			FromDate: "2025-01-01",
			ToDate:   "2025-01-15",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if res.IsError {
			t.Fatalf("unexpected IsError: %s", res.Content[0].(*mcp.TextContent).Text)
		}
		tc := res.Content[0].(*mcp.TextContent)
		if tc.Text == "" || tc.Text == "Error listing exercises: " {
			t.Fatalf("expected JSON body, got %q", tc.Text)
		}
	})

	t.Run("returns_error_when_list_fails", func(t *testing.T) {
		svc := &mockContextService{listErr: errors.New("connection refused")}
		h := NewHandler(svc)
		fn := h.GetExercisesForTimeRangeTool()
		res, _, err := fn(context.Background(), &mcp.CallToolRequest{}, ExercisesTimeRangeInput{
			FromDate: "2025-01-01",
			ToDate:   "2025-01-15",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !res.IsError {
			t.Fatalf("expected IsError")
		}
		tc := res.Content[0].(*mcp.TextContent)
		if tc.Text != "Error listing exercises: connection refused" {
			t.Fatalf("content text = %q", tc.Text)
		}
	})
}

// Tests for GetExerciseTypesTool.
func TestHandler_GetExerciseTypesTool(t *testing.T) {
	t.Run("returns_types", func(t *testing.T) {
		types := []exercises.ExerciseType{
			{ExerciseID: "bp", MuscleGroup: "chest", Name: "Bench Press"},
		}
		svc := &mockContextService{exerciseTypes: types}
		h := NewHandler(svc)
		fn := h.GetExerciseTypesTool()
		res, _, err := fn(context.Background(), &mcp.CallToolRequest{}, ExerciseTypesInput{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if res.IsError {
			t.Fatalf("unexpected IsError: %s", res.Content[0].(*mcp.TextContent).Text)
		}
		tc := res.Content[0].(*mcp.TextContent)
		if tc.Text == "" || tc.Text == "Error fetching exercise types: " {
			t.Fatalf("expected JSON body, got %q", tc.Text)
		}
	})

	t.Run("returns_error_when_types_fail", func(t *testing.T) {
		svc := &mockContextService{typesErr: errors.New("timeout")}
		h := NewHandler(svc)
		fn := h.GetExerciseTypesTool()
		res, _, err := fn(context.Background(), &mcp.CallToolRequest{}, ExerciseTypesInput{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !res.IsError {
			t.Fatalf("expected IsError")
		}
		tc := res.Content[0].(*mcp.TextContent)
		if tc.Text != "Error fetching exercise types: timeout" {
			t.Fatalf("content text = %q", tc.Text)
		}
	})
}
