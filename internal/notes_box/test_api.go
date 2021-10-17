package notes_box

import "errors"

type TestApi struct {
	notes map[int]*Note
}

func NewTestApi() *TestApi {
	return &TestApi{
		notes: make(map[int]*Note),
	}
}

func (api *TestApi) Add(note *Note) (*Note, error) {
	api.notes[note.Id] = note
	return note, nil
}

func (api *TestApi) Update(note *Note) error {
	if _, err := api.Get(note.Id); err != nil {
		return err
	}
	api.notes[note.Id] = note
	return nil
}

func (api *TestApi) Get(id int) (*Note, error) {
	note, ok := api.notes[id]
	if !ok {
		return nil, errors.New("not found")
	}
	return note, nil
}

func (api *TestApi) Delete(id int) (bool, error) {
	note, ok := api.notes[id]
	if !ok {
		return false, errors.New("not found")
	}
	delete(api.notes, note.Id)
	return true, nil
}

func (api *TestApi) List() ([]Note, error) {
	var notes []Note
	for _, n := range api.notes {
		notes = append(notes, *n)
	}
	return notes, nil
}
