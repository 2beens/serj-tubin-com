package netlog

import (
	"errors"
	"sort"
	"strings"
	"sync"
)

type TestApi struct {
	// visit ID to Visit
	Visits map[int]Visit
	mutex  sync.Mutex
}

func NewTestApi() *TestApi {
	return &TestApi{
		Visits: map[int]Visit{},
	}
}

func (api *TestApi) CloseDB() {
	// NOP
}

func (api *TestApi) AddVisit(visit *Visit) error {
	api.mutex.Lock()
	defer api.mutex.Unlock()

	nextId := len(api.Visits)

	api.Visits[nextId] = *visit
	return nil
}

func (api *TestApi) GetAllVisits() ([]*Visit, error) {
	return api.GetVisits([]string{}, "url", "all", -1)
}

func (api *TestApi) GetVisits(keywords []string, field string, source string, limit int) ([]*Visit, error) {
	api.mutex.Lock()
	defer api.mutex.Unlock()

	if field != "url" && field != "title" {
		return nil, errors.New("unknown field name")
	}

	var foundVisits []*Visit
	for k := range api.Visits {
		visit := api.Visits[k]
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

func (api *TestApi) CountAll() (int, error) {
	api.mutex.Lock()
	defer api.mutex.Unlock()

	return len(api.Visits), nil
}

func (api *TestApi) Count(keywords []string, field string, source string) (int, error) {
	api.mutex.Lock()
	defer api.mutex.Unlock()

	if field != "url" && field != "title" {
		return -1, errors.New("unknown field name")
	}

	count := 0
	for _, visit := range api.Visits {
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

func (api *TestApi) GetVisitsPage(keywords []string, field string, source string, page int, size int) ([]*Visit, error) {
	if len(api.Visits) <= size {
		return api.GetAllVisits()
	}

	foundVisits, err := api.GetVisits(keywords, field, source, -1)
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
