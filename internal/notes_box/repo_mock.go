package notes_box

import (
	"context"
	"errors"
)

type repoMock struct {
	notes map[int]*Note
}

func NewMockNotesRepo() *repoMock {
	return &repoMock{
		notes: make(map[int]*Note),
	}
}

func (r *repoMock) Add(_ context.Context, note *Note) (*Note, error) {
	r.notes[note.Id] = note
	return note, nil
}

func (r *repoMock) Update(ctx context.Context, note *Note) error {
	if _, err := r.Get(ctx, note.Id); err != nil {
		return err
	}
	r.notes[note.Id] = note
	return nil
}

func (r *repoMock) Get(_ context.Context, id int) (*Note, error) {
	note, ok := r.notes[id]
	if !ok {
		return nil, errors.New("not found")
	}
	return note, nil
}

func (r *repoMock) Delete(_ context.Context, id int) error {
	note, ok := r.notes[id]
	if !ok {
		return ErrNoteNotFound
	}
	delete(r.notes, note.Id)
	return nil
}

func (r *repoMock) List(context.Context) ([]Note, error) {
	var notes []Note
	for _, n := range r.notes {
		notes = append(notes, *n)
	}
	return notes, nil
}
