package notes_box

import (
	"context"
	"errors"
	"sort"
)

type repoMock struct {
	notes map[int]*Note
}

func newRepoMock() *repoMock {
	return &repoMock{
		notes: make(map[int]*Note),
	}
}

func (r *repoMock) Add(_ context.Context, note *Note) (*Note, error) {
	r.notes[note.ID] = note
	return note, nil
}

func (r *repoMock) Update(ctx context.Context, note *Note) error {
	if _, err := r.Get(ctx, note.ID); err != nil {
		return err
	}
	r.notes[note.ID] = note
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
	delete(r.notes, note.ID)
	return nil
}

func (r *repoMock) List(context.Context) ([]Note, error) {
	var notes []Note
	for _, n := range r.notes {
		notes = append(notes, *n)
	}
	sort.Slice(notes, func(i, j int) bool {
		return notes[i].CreatedAt.After(notes[j].CreatedAt)
	})
	return notes, nil
}
