package mcp

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/2beens/serjtubincom/internal/gymstats/exercises"
)

// ExercisesRepo provides exercise list and types (for dependency injection and testing).
type ExercisesRepo interface {
	ListAll(ctx context.Context, params exercises.ExerciseParams) ([]exercises.Exercise, error)
	GetExerciseTypes(ctx context.Context, params exercises.GetExerciseTypesParams) ([]exercises.ExerciseType, error)
}

// exerciseAnalyzer provides exercise history and analytics (for dependency injection and testing).
type exerciseAnalyzer interface {
	ExerciseHistory(ctx context.Context, params exercises.ExerciseParams) (*exercises.ExerciseHistory, error)
	AvgSetDuration(ctx context.Context, params exercises.ExerciseParams) (*exercises.AvgSetDurationResponse, error)
	ExercisePercentages(ctx context.Context, muscleGroup string, onlyProd, excludeTestingData bool) (map[string]exercises.ExercisePercentageInfo, error)
}

// contextService provides gymstats context data (schema, exercises, exercise types, analytics).
// Used by Handler for testability.
type contextService interface {
	GetSchema(ctx context.Context) (string, error)
	ListExercises(ctx context.Context, params exercises.ExerciseParams) ([]exercises.Exercise, error)
	GetExerciseTypes(ctx context.Context, params exercises.GetExerciseTypesParams) ([]exercises.ExerciseType, error)
	GetExerciseHistory(ctx context.Context, params exercises.ExerciseParams) (*exercises.ExerciseHistory, error)
	GetExercisePercentages(ctx context.Context, muscleGroup string) (map[string]exercises.ExercisePercentageInfo, error)
	GetAvgSetDuration(ctx context.Context, params exercises.ExerciseParams) (*exercises.AvgSetDurationResponse, error)
}

// ContextService holds dependencies and implements the gymstats context business logic.
type ContextService struct {
	schema    SchemaRepo
	exercises ExercisesRepo
	analyzer  exerciseAnalyzer
}

// NewContextService builds a ContextService with the given dependencies.
func NewContextService(schemaRepo SchemaRepo, exercisesRepo ExercisesRepo, analyzer exerciseAnalyzer) *ContextService {
	return &ContextService{
		schema:    schemaRepo,
		exercises: exercisesRepo,
		analyzer:  analyzer,
	}
}

// GetSchema returns the DB schema (table names, columns, types) for gymstats-related
// tables: exercise, exercise_type, exercise_image, gymstats_event.
func (s *ContextService) GetSchema(ctx context.Context) (string, error) {
	cols, err := s.schema.GetGymstatsColumns(ctx)
	if err != nil {
		return "", err
	}
	return formatGymstatsSchema(cols), nil
}

func formatGymstatsSchema(cols []SchemaColumn) string {
	if len(cols) == 0 {
		return "# Gymstats DB Schema\n\nNo gymstats tables found in the database.\n"
	}

	byTable := make(map[string][]SchemaColumn)
	for _, c := range cols {
		byTable[c.TableName] = append(byTable[c.TableName], c)
	}

	tableOrder := make([]string, 0, len(byTable))
	for t := range byTable {
		tableOrder = append(tableOrder, t)
	}
	sort.Strings(tableOrder)

	var b strings.Builder
	b.WriteString("# Gymstats DB Schema\n\n")
	b.WriteString("Tables: exercise, exercise_type, exercise_image, gymstats_event (schema: public).\n\n")

	for _, tableName := range tableOrder {
		tableCols := byTable[tableName]
		b.WriteString("## ")
		b.WriteString(tableName)
		b.WriteString("\n\n| Column | Type | Nullable | Default |\n|--------|------|----------|--------|\n")
		for _, c := range tableCols {
			def := "—"
			if c.ColumnDef != nil && *c.ColumnDef != "" {
				def = *c.ColumnDef
			}
			b.WriteString(fmt.Sprintf("| %s | %s | %s | %s |\n", c.ColumnName, c.DataType, c.IsNullable, def))
		}
		b.WriteString("\n")
	}

	return strings.TrimSuffix(b.String(), "\n\n") + "\n"
}

// ListExercises returns exercises (sets) for the given params (time range, filters).
func (s *ContextService) ListExercises(ctx context.Context, params exercises.ExerciseParams) ([]exercises.Exercise, error) {
	return s.exercises.ListAll(ctx, params)
}

// GetExerciseTypes returns exercise types, optionally filtered.
func (s *ContextService) GetExerciseTypes(ctx context.Context, params exercises.GetExerciseTypesParams) ([]exercises.ExerciseType, error) {
	return s.exercises.GetExerciseTypes(ctx, params)
}

// GetExerciseHistory returns per-day stats (avg kilos, avg reps, sets) for the given exercise/period.
func (s *ContextService) GetExerciseHistory(ctx context.Context, params exercises.ExerciseParams) (*exercises.ExerciseHistory, error) {
	return s.analyzer.ExerciseHistory(ctx, params)
}

// GetExercisePercentages returns the percentage of each exercise type for the given muscle group.
func (s *ContextService) GetExercisePercentages(ctx context.Context, muscleGroup string) (map[string]exercises.ExercisePercentageInfo, error) {
	return s.analyzer.ExercisePercentages(ctx, muscleGroup, false, false)
}

// GetAvgSetDuration returns the average rest time between sets (overall and per day) for the given period.
func (s *ContextService) GetAvgSetDuration(ctx context.Context, params exercises.ExerciseParams) (*exercises.AvgSetDurationResponse, error) {
	return s.analyzer.AvgSetDuration(ctx, params)
}
