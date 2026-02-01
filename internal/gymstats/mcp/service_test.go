package mcp

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/2beens/serjtubincom/internal/gymstats/exercises"
)

// mockSchemaRepo implements SchemaRepo for service tests.
type mockSchemaRepo struct {
	cols []SchemaColumn
	err  error
}

func (m *mockSchemaRepo) GetGymstatsColumns(ctx context.Context) ([]SchemaColumn, error) {
	return m.cols, m.err
}

// mockExercisesRepo implements ExercisesRepo for service tests.
type mockExercisesRepo struct {
	list          []exercises.Exercise
	listErr       error
	exerciseTypes []exercises.ExerciseType
	typesErr      error
}

func (m *mockExercisesRepo) ListAll(ctx context.Context, params exercises.ExerciseParams) ([]exercises.Exercise, error) {
	return m.list, m.listErr
}

func (m *mockExercisesRepo) GetExerciseTypes(ctx context.Context, params exercises.GetExerciseTypesParams) ([]exercises.ExerciseType, error) {
	return m.exerciseTypes, m.typesErr
}

// mockExerciseAnalyzer implements exerciseAnalyzer for service tests.
type mockExerciseAnalyzer struct {
	history         *exercises.ExerciseHistory
	historyErr      error
	percentages     map[string]exercises.ExercisePercentageInfo
	percentagesErr  error
	avgSetDuration  *exercises.AvgSetDurationResponse
	avgSetDurationErr error
}

func (m *mockExerciseAnalyzer) ExerciseHistory(ctx context.Context, params exercises.ExerciseParams) (*exercises.ExerciseHistory, error) {
	return m.history, m.historyErr
}

func (m *mockExerciseAnalyzer) ExercisePercentages(ctx context.Context, muscleGroup string, onlyProd, excludeTestingData bool) (map[string]exercises.ExercisePercentageInfo, error) {
	return m.percentages, m.percentagesErr
}

func (m *mockExerciseAnalyzer) AvgSetDuration(ctx context.Context, params exercises.ExerciseParams) (*exercises.AvgSetDurationResponse, error) {
	return m.avgSetDuration, m.avgSetDurationErr
}

func TestContextService_GetSchema(t *testing.T) {
	t.Run("returns_formatted_schema", func(t *testing.T) {
		cols := []SchemaColumn{
			{TableSchema: "public", TableName: "exercise", ColumnName: "id", DataType: "integer", IsNullable: "NO", ColumnDef: strPtr("nextval('exercise_id_seq'::regclass)")},
			{TableSchema: "public", TableName: "exercise", ColumnName: "exercise_id", DataType: "text", IsNullable: "NO", ColumnDef: nil},
		}
		schemaRepo := &mockSchemaRepo{cols: cols}
		svc := NewContextService(schemaRepo, &mockExercisesRepo{}, &mockExerciseAnalyzer{})

		got, err := svc.GetSchema(context.Background())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if !strings.Contains(got, "# Gymstats DB Schema") {
			t.Errorf("expected header; got %q", got)
		}
		if !strings.Contains(got, "## exercise") {
			t.Errorf("expected table name; got %q", got)
		}
		if !strings.Contains(got, "| id | integer |") {
			t.Errorf("expected column row; got %q", got)
		}
		if !strings.Contains(got, "| exercise_id | text |") {
			t.Errorf("expected column row; got %q", got)
		}
	})

	t.Run("returns_empty_message_when_no_columns", func(t *testing.T) {
		schemaRepo := &mockSchemaRepo{cols: nil}
		svc := NewContextService(schemaRepo, &mockExercisesRepo{}, &mockExerciseAnalyzer{})

		got, err := svc.GetSchema(context.Background())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if !strings.Contains(got, "No gymstats tables found in the database") {
			t.Errorf("expected empty message; got %q", got)
		}
	})

	t.Run("returns_error_when_repo_fails", func(t *testing.T) {
		wantErr := errors.New("db connection failed")
		schemaRepo := &mockSchemaRepo{err: wantErr}
		svc := NewContextService(schemaRepo, &mockExercisesRepo{}, &mockExerciseAnalyzer{})

		_, err := svc.GetSchema(context.Background())
		if err != wantErr {
			t.Fatalf("err = %v, want %v", err, wantErr)
		}
	})
}

func TestContextService_ListExercises(t *testing.T) {
	t.Run("returns_list_from_repo", func(t *testing.T) {
		now := time.Now()
		want := []exercises.Exercise{
			{ID: 1, ExerciseID: "bp", Kilos: 80, Reps: 10, CreatedAt: now},
		}
		repo := &mockExercisesRepo{list: want}
		svc := NewContextService(&mockSchemaRepo{}, repo, &mockExerciseAnalyzer{})

		params := exercises.ExerciseParams{}
		got, err := svc.ListExercises(context.Background(), params)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(got) != 1 || got[0].ID != want[0].ID || got[0].ExerciseID != want[0].ExerciseID {
			t.Errorf("got %+v, want %+v", got, want)
		}
	})

	t.Run("returns_error_when_repo_fails", func(t *testing.T) {
		wantErr := errors.New("connection refused")
		repo := &mockExercisesRepo{listErr: wantErr}
		svc := NewContextService(&mockSchemaRepo{}, repo, &mockExerciseAnalyzer{})

		_, err := svc.ListExercises(context.Background(), exercises.ExerciseParams{})
		if err != wantErr {
			t.Fatalf("err = %v, want %v", err, wantErr)
		}
	})
}

func TestContextService_GetExerciseTypes(t *testing.T) {
	t.Run("returns_types_from_repo", func(t *testing.T) {
		want := []exercises.ExerciseType{
			{ExerciseID: "bp", MuscleGroup: "chest", Name: "Bench Press"},
		}
		repo := &mockExercisesRepo{exerciseTypes: want}
		svc := NewContextService(&mockSchemaRepo{}, repo, &mockExerciseAnalyzer{})

		params := exercises.GetExerciseTypesParams{}
		got, err := svc.GetExerciseTypes(context.Background(), params)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(got) != 1 || got[0].ExerciseID != want[0].ExerciseID || got[0].Name != want[0].Name {
			t.Errorf("got %+v, want %+v", got, want)
		}
	})

	t.Run("returns_error_when_repo_fails", func(t *testing.T) {
		wantErr := errors.New("timeout")
		repo := &mockExercisesRepo{typesErr: wantErr}
		svc := NewContextService(&mockSchemaRepo{}, repo, &mockExerciseAnalyzer{})

		_, err := svc.GetExerciseTypes(context.Background(), exercises.GetExerciseTypesParams{})
		if err != wantErr {
			t.Fatalf("err = %v, want %v", err, wantErr)
		}
	})
}

func strPtr(s string) *string {
	return &s
}
