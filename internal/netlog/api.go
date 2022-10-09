package netlog

import "context"

type Api interface {
	AddVisit(ctx context.Context, visit *Visit) error
	GetVisits(ctx context.Context, keywords []string, field string, source string, limit int) ([]*Visit, error)
	CountAll(ctx context.Context) (int, error)
	Count(ctx context.Context, keywords []string, field string, source string) (int, error)
	GetVisitsPage(ctx context.Context, keywords []string, field string, source string, page int, size int) ([]*Visit, error)
}
