package visitor_board

import (
	"context"
	"errors"
)

type mockRepo struct {
	Messages []Message
}

func NewMockMessagesRepo() *mockRepo {
	return &mockRepo{
		Messages: make([]Message, 0),
	}
}

func (m *mockRepo) Add(_ context.Context, message Message) (int, error) {
	message.ID = len(m.Messages) + 1
	m.Messages = append(m.Messages, message)
	return message.ID, nil
}

func (m *mockRepo) Delete(_ context.Context, id int) error {
	for i, msg := range m.Messages {
		if msg.ID == id {
			m.Messages = append(m.Messages[:i], m.Messages[i+1:]...)
			return nil
		}
	}
	return ErrMessageNotFound
}

func (m *mockRepo) List(_ context.Context, options ...func(listOptions *ListOptions)) ([]Message, error) {
	opts := &ListOptions{}
	for _, option := range options {
		option(opts)
	}

	if opts.Limit <= 0 || opts.Limit > len(m.Messages) {
		return m.Messages, nil
	}

	return m.Messages[:opts.Limit], nil
}

func (m *mockRepo) GetMessagesPage(_ context.Context, page, size int) ([]Message, error) {
	if size <= 0 {
		return nil, errors.New("invalid page size")
	}

	start := (page - 1) * size
	end := start + size

	if start >= len(m.Messages) {
		return nil, errors.New("invalid page number")
	}

	if end > len(m.Messages) {
		end = len(m.Messages)
	}

	return m.Messages[start:end], nil
}

func (m *mockRepo) AllMessagesCount(_ context.Context) (int, error) {
	return len(m.Messages), nil
}
