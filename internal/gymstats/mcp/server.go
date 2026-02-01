package mcp

import (
	"github.com/2beens/serjtubincom/internal/gymstats/exercises"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// NewServer builds an MCP server with gymstats tools: get_gymstats_context,
// get_exercises_for_time_range, get_exercise_types.
// Used both by the standalone stdio cmd (cmd/gymstats_mcp) and by the main
// backend when mounting MCP over HTTP (internal/server).
func NewServer(pool *pgxpool.Pool, repo *exercises.Repo) *mcp.Server {
	svc := NewContextService(NewPoolSchemaRepo(pool), repo)
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

	return s
}
