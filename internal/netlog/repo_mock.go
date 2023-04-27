package netlog

import (
	"context"
	"errors"
	"sort"
	"strings"
	"sync"
	"time"
)

var _ netlogRepo = (*repoMock)(nil)

type repoMock struct {
	// visit ID to Visit
	Visits map[int]Visit
	mutex  sync.Mutex
}

func NewRepoMock() *repoMock {
	repo := &repoMock{
		Visits: map[int]Visit{},
	}

	now := time.Now()
	visit0 := Visit{
		Id:        0,
		Title:     "test title 0",
		Source:    "chrome",
		URL:       "test:url:0",
		Timestamp: now,
	}
	visit1 := Visit{
		Id:        1,
		Title:     "test title 1",
		Source:    "chrome",
		URL:       "test:url:1",
		Timestamp: now,
	}
	repo.Visits[0] = visit0
	repo.Visits[1] = visit1

	return repo
}

func (r *repoMock) AddVisit(_ context.Context, visit *Visit) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	nextId := len(r.Visits)

	r.Visits[nextId] = *visit
	return nil
}

func (r *repoMock) GetAllVisits(ctx context.Context) ([]*Visit, error) {
	return r.GetVisits(ctx, []string{}, "url", "all", -1)
}

func (r *repoMock) GetVisits(_ context.Context, keywords []string, field string, source string, limit int) ([]*Visit, error) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if field != "url" && field != "title" {
		return nil, errors.New("unknown field name")
	}

	var foundVisits []*Visit
	for k := range r.Visits {
		visit := r.Visits[k]
		if len(keywords) > 0 {
			for _, keyword := range keywords {
				field := visit.URL
				if field == "title" {
					field = visit.Title
				}
				if !strings.Contains(field, keyword) {
					continue
				}
				if source != "all" && visit.Source != source {
					continue
				}
				foundVisits = append(foundVisits, &visit)
				break
			}
		} else {
			foundVisits = append(foundVisits, &visit)
		}
		if limit >= 0 && len(foundVisits) == limit {
			break
		}
	}
	return foundVisits, nil
}

func (r *repoMock) CountAll(_ context.Context) (int, error) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	return len(r.Visits), nil
}

func (r *repoMock) Count(_ context.Context, keywords []string, field string, source string) (int, error) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if field != "url" && field != "title" {
		return -1, errors.New("unknown field name")
	}

	count := 0
	for _, visit := range r.Visits {
		for _, keyword := range keywords {
			field := visit.URL
			if field == "title" {
				field = visit.Title
			}
			if !strings.Contains(field, keyword) {
				continue
			}
			if source != "all" && visit.Source != source {
				continue
			}
			count++
			break
		}
	}
	return count, nil
}

func (r *repoMock) GetVisitsPage(ctx context.Context, keywords []string, field string, source string, page int, size int) ([]*Visit, error) {
	if len(r.Visits) <= size {
		return r.GetAllVisits(ctx)
	}

	foundVisits, err := r.GetVisits(ctx, keywords, field, source, -1)
	if err != nil {
		return nil, err
	}

	if foundVisits == nil {
		return []*Visit{}, nil
	}

	sort.Slice(foundVisits, func(i, j int) bool {
		return foundVisits[i].Timestamp.Before(foundVisits[j].Timestamp)
	})

	startIndex := (page - 1) * size
	endIndex := startIndex + size - 1

	if endIndex > (len(foundVisits) - 1) {
		endIndex = len(foundVisits) - 1
	}

	// overflow
	if startIndex >= len(foundVisits) {
		return []*Visit{}, errors.New("index overflow")
	}

	var visits []*Visit
	for i := startIndex; i <= endIndex; i++ {
		visits = append(visits, foundVisits[i])
	}
	return visits, nil
}
