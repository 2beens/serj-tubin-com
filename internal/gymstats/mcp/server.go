package mcp

import (
	"github.com/2beens/serjtubincom/internal/gymstats/exercises"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// NewServer builds an MCP server with gymstats tools: schema, exercises, exercise types,
// exercise history, exercise percentages, avg set duration.
// Used by the main backend when mounting MCP at /mcp (internal/server).
func NewServer(pool *pgxpool.Pool, repo *exercises.Repo) *mcp.Server {
	analyzer := exercises.NewAnalyzer(repo)
	svc := NewContextService(NewPoolSchemaRepo(pool), repo, analyzer)
	h := NewHandler(svc)
	s := mcp.NewServer(&mcp.Implementation{
		Name:    "gymstats-context",
		Version: "1.0.0",
	}, nil)

	mcp.AddTool(s, &mcp.Tool{
		Name:        "get_gymstats_context",
		Description: "Returns the DB schema for gymstats-related tables (exercise, exercise_type, exercise_image, gymstats_event): table names, columns, types, nullable, default. Use when developing the gymstats app and you need the actual backend schema.",
	}, h.GetGymstatsContextTool())

	mcp.AddTool(s, &mcp.Tool{
		Name:        "get_exercises_for_time_range",
		Description: "Returns exercises (sets) done within the given date range. Optional filters: muscle_group (e.g. chest, legs), exercise_id (e.g. bench_press). Use when you need to see what was logged in a period.",
	}, h.GetExercisesForTimeRangeTool())

	mcp.AddTool(s, &mcp.Tool{
		Name:        "get_exercise_types",
		Description: "Returns all exercise types (id, muscle group, name, description). Optional filters: muscle_group, exercise_id. Use when you need the list of available exercise types.",
	}, h.GetExerciseTypesTool())

	mcp.AddTool(s, &mcp.Tool{
		Name:        "get_exercise_history",
		Description: "Returns per-day stats (avg kilos, avg reps, sets) for an exercise or muscle group in a date range. Args: from_date, to_date (YYYY-MM-DD); optional: muscle_group, exercise_id. Use when you need progression or volume over time (e.g. how has bench press improved).",
	}, h.GetExerciseHistoryTool())

	mcp.AddTool(s, &mcp.Tool{
		Name:        "get_exercise_percentages",
		Description: "Returns the percentage of each exercise type for a muscle group (e.g. what % bench press vs incline for chest). Arg: muscle_group (e.g. chest, legs). Use when you want to see workout mix or balance.",
	}, h.GetExercisePercentagesTool())

	mcp.AddTool(s, &mcp.Tool{
		Name:        "get_avg_set_duration",
		Description: "Returns the average rest time between sets (overall and per day) for a date range. Args: from_date, to_date (YYYY-MM-DD); optional: muscle_group, exercise_id. Use when analyzing rest patterns or session density.",
	}, h.GetAvgSetDurationTool())

	return s
}
