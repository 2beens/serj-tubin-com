package notes_box

var _ Api = (*PsqlApi)(nil)
var _ Api = (*TestApi)(nil)

type Api interface {
	Add(note *Note) (Note, error)
	Remove(id int) (bool, error)
	List() ([]Note, error)
}
