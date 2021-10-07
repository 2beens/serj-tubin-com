package notes_box

type TestApi struct {
}

func (api *TestApi) Add(note *Note) (Note, error) {
	panic("not impl")
}

func (api *TestApi) Remove(id int) (Note, error) {
	panic("not impl")
}

func (api *TestApi) List() ([]Note, error) {
	panic("not impl")
}
