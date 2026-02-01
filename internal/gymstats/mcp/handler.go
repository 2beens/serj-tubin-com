package mcp

import (
	"context"
	"encoding/json"
	"time"

	"github.com/2beens/serjtubincom/internal/gymstats/exercises"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Handler handles MCP tool requests and responses: parses input, calls the service, formats MCP result.
type Handler struct {
	service contextService
}

// NewHandler builds a handler with the given service.
func NewHandler(service contextService) *Handler {
	return &Handler{
		service: service,
	}
}

// GetGymstatsContextTool returns the MCP tool handler for get_gymstats_context.
func (h *Handler) GetGymstatsContextTool() func(context.Context, *mcp.CallToolRequest, any) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, _ *mcp.CallToolRequest, _ any) (*mcp.CallToolResult, any, error) {
		text, err := h.service.GetSchema(ctx)
		if err != nil {
			return &mcp.CallToolResult{
				Content: []mcp.Content{&mcp.TextContent{Text: "Error fetching schema: " + err.Error()}},
				IsError: true,
			}, nil, nil
		}
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: text}},
		}, nil, nil
	}
}

// ExercisesTimeRangeInput is the input for get_exercises_for_time_range.
type ExercisesTimeRangeInput struct {
	FromDate    string `json:"from_date" jsonschema:"Start date (YYYY-MM-DD)"`
	ToDate      string `json:"to_date" jsonschema:"End date (YYYY-MM-DD)"`
	MuscleGroup string `json:"muscle_group,omitempty" jsonschema:"Filter by muscle group (e.g. chest, legs)"`
	ExerciseID  string `json:"exercise_id,omitempty" jsonschema:"Filter by exercise type id (e.g. bench_press)"`
}

// GetExercisesForTimeRangeTool returns the MCP tool handler for get_exercises_for_time_range.
func (h *Handler) GetExercisesForTimeRangeTool() func(context.Context, *mcp.CallToolRequest, ExercisesTimeRangeInput) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, _ *mcp.CallToolRequest, in ExercisesTimeRangeInput) (*mcp.CallToolResult, any, error) {
		from, err := time.Parse("2006-01-02", in.FromDate)
		if err != nil {
			return &mcp.CallToolResult{
				Content: []mcp.Content{&mcp.TextContent{Text: "Invalid from_date: use YYYY-MM-DD"}},
				IsError: true,
			}, nil, nil
		}
		to, err := time.Parse("2006-01-02", in.ToDate)
		if err != nil {
			return &mcp.CallToolResult{
				Content: []mcp.Content{&mcp.TextContent{Text: "Invalid to_date: use YYYY-MM-DD"}},
				IsError: true,
			}, nil, nil
		}
		to = time.Date(to.Year(), to.Month(), to.Day(), 23, 59, 59, 999999999, to.Location())

		params := exercises.ExerciseParams{
			From:        &from,
			To:          &to,
			MuscleGroup: in.MuscleGroup,
			ExerciseID:  in.ExerciseID,
		}
		list, err := h.service.ListExercises(ctx, params)
		if err != nil {
			return &mcp.CallToolResult{
				Content: []mcp.Content{&mcp.TextContent{Text: "Error listing exercises: " + err.Error()}},
				IsError: true,
			}, nil, nil
		}
		raw, err := json.MarshalIndent(list, "", "  ")
		if err != nil {
			return &mcp.CallToolResult{
				Content: []mcp.Content{&mcp.TextContent{Text: "Error encoding response: " + err.Error()}},
				IsError: true,
			}, nil, nil
		}
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: string(raw)}},
		}, nil, nil
	}
}

// ExerciseTypesInput is the input for get_exercise_types.
type ExerciseTypesInput struct {
	MuscleGroup string `json:"muscle_group,omitempty" jsonschema:"Filter by muscle group (e.g. chest, legs)"`
	ExerciseID  string `json:"exercise_id,omitempty" jsonschema:"Filter by exercise type id (e.g. bench_press)"`
}

// GetExerciseTypesTool returns the MCP tool handler for get_exercise_types.
func (h *Handler) GetExerciseTypesTool() func(context.Context, *mcp.CallToolRequest, ExerciseTypesInput) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, _ *mcp.CallToolRequest, in ExerciseTypesInput) (*mcp.CallToolResult, any, error) {
		params := exercises.GetExerciseTypesParams{
			MuscleGroup: in.MuscleGroup,
			ExerciseId:  in.ExerciseID,
		}
		types, err := h.service.GetExerciseTypes(ctx, params)
		if err != nil {
			return &mcp.CallToolResult{
				Content: []mcp.Content{&mcp.TextContent{Text: "Error fetching exercise types: " + err.Error()}},
				IsError: true,
			}, nil, nil
		}
		raw, err := json.MarshalIndent(types, "", "  ")
		if err != nil {
			return &mcp.CallToolResult{
				Content: []mcp.Content{&mcp.TextContent{Text: "Error encoding response: " + err.Error()}},
				IsError: true,
			}, nil, nil
		}
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: string(raw)}},
		}, nil, nil
	}
}
