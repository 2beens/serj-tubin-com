package notes_box

import (
	"context"
	"errors"
)

type TestApi struct {
	notes map[int]*Note
}

func NewTestApi() *TestApi {
	return &TestApi{
		notes: make(map[int]*Note),
	}
}

func (api *TestApi) Add(_ context.Context, note *Note) (*Note, error) {
	api.notes[note.Id] = note
	return note, nil
}

func (api *TestApi) Update(ctx context.Context, note *Note) error {
	if _, err := api.Get(ctx, note.Id); err != nil {
		return err
	}
	api.notes[note.Id] = note
	return nil
}

func (api *TestApi) Get(_ context.Context, id int) (*Note, error) {
	note, ok := api.notes[id]
	if !ok {
		return nil, errors.New("not found")
	}
	return note, nil
}

func (api *TestApi) Delete(_ context.Context, id int) error {
	note, ok := api.notes[id]
	if !ok {
		return ErrNoteNotFound
	}
	delete(api.notes, note.Id)
	return nil
}

func (api *TestApi) List(context.Context) ([]Note, error) {
	var notes []Note
	for _, n := range api.notes {
		notes = append(notes, *n)
	}
	return notes, nil
}
