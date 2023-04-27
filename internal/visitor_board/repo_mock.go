package visitor_board

import (
	"context"
	"errors"
	"sort"
	"time"
)

type repoMock struct {
	Messages []Message
}

func NewMockMessagesRepo() *repoMock {
	now := time.Now()
	return &repoMock{
		Messages: []Message{
			{
				ID:        0,
				Author:    "serj",
				Message:   "test message blabla",
				CreatedAt: now.Add(-time.Hour),
			},
			{
				ID:        1,
				Author:    "serj",
				Message:   "test message gragra",
				CreatedAt: now,
			},
			{
				ID:        2,
				Author:    "ana",
				Message:   "test message aaaaa",
				CreatedAt: now.Add(-2 * time.Hour),
			},
			{
				ID:        3,
				Author:    "drago",
				Message:   "drago's test message aaaaa sve",
				CreatedAt: now.Add(-5 * 24 * time.Hour),
			},
			{
				ID:        4,
				Author:    "rodjak nenad",
				Message:   "ja se mislim sta'e bilo",
				CreatedAt: now.Add(-2 * time.Minute),
			},
		},
	}
}

func (mr *repoMock) Add(_ context.Context, message Message) (int, error) {
	message.ID = len(mr.Messages) + 1
	mr.Messages = append(mr.Messages, message)
	return message.ID, nil
}

func (mr *repoMock) Delete(_ context.Context, id int) error {
	for i, msg := range mr.Messages {
		if msg.ID == id {
			mr.Messages = append(mr.Messages[:i], mr.Messages[i+1:]...)
			return nil
		}
	}
	return ErrMessageNotFound
}

// List returns last n messages, determined by the limit option.
func (mr *repoMock) List(_ context.Context, options ...func(listOptions *ListOptions)) ([]Message, error) {
	opts := &ListOptions{}
	for _, option := range options {
		option(opts)
	}

	sort.Slice(mr.Messages, func(i, j int) bool {
		return mr.Messages[i].CreatedAt.Before(mr.Messages[j].CreatedAt)
	})

	if opts.Limit <= 0 || opts.Limit > len(mr.Messages) {
		return mr.Messages, nil
	}

	return mr.Messages[len(mr.Messages)-opts.Limit:], nil
}

func (mr *repoMock) GetMessagesPage(_ context.Context, page, size int) ([]Message, error) {
	if size <= 0 {
		return nil, errors.New("invalid page size")
	}

	start := (page - 1) * size
	end := start + size

	if start >= len(mr.Messages) {
		return nil, errors.New("invalid page number")
	}

	if end > len(mr.Messages) {
		end = len(mr.Messages)
	}

	return mr.Messages[start:end], nil
}

func (mr *repoMock) AllMessagesCount(_ context.Context) (int, error) {
	return len(mr.Messages), nil
}
