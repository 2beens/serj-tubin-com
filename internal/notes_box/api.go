package notes_box

var _ Api = (*PsqlApi)(nil)
var _ Api = (*TestApi)(nil)

type Api interface {
	Add(note *Note) (*Note, error)
	Get(id int) (*Note, error)
	Delete(id int) (bool, error)
	List() ([]Note, error)
}
