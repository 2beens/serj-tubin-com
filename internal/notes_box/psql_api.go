package notes_box

type PsqlApi struct {
}

func (api *PsqlApi) Add(note *Note) (Note, error) {
	panic("not impl")
}

func (api *PsqlApi) Remove(id int) (Note, error) {
	panic("not impl")
}

func (api *PsqlApi) List() ([]Note, error) {
	var notes []Note
	return notes, nil
}
