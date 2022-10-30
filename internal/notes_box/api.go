package notes_box

import "context"

var _ Api = (*PsqlApi)(nil)
var _ Api = (*TestApi)(nil)

type Api interface {
	Add(ctx context.Context, note *Note) (*Note, error)
	Update(ctx context.Context, note *Note) error
	Get(ctx context.Context, id int) (*Note, error)
	Delete(ctx context.Context, id int) error
	List(ctx context.Context) ([]Note, error)
}
