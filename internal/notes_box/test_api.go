package notes_box

type TestApi struct {
}

func (api *TestApi) Add(note *Note) (*Note, error) {
	panic("not impl")
}

func (api *TestApi) Get(id int) (*Note, error) {
	panic("not impl")
}

func (api *TestApi) Delete(id int) (bool, error) {
	panic("not impl")
}

func (api *TestApi) List() ([]Note, error) {
	panic("not impl")
}
