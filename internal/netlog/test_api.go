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
	api.Visits[visit.Id] = *visit
	return nil
}

func (api *TestApi) GetAllVisits() ([]*Visit, error) {
	return api.GetVisits([]string{}, -1)
}

func (api *TestApi) GetVisits(keywords []string, limit int) ([]*Visit, error) {
	api.mutex.Lock()
	defer api.mutex.Unlock()

	var foundVisits []*Visit
	for _, visit := range api.Visits {
		if len(keywords) > 0 {
			for _, keyword := range keywords {
				if strings.Contains(visit.URL, keyword) {
					foundVisits = append(foundVisits, &visit)
					break
				}
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

func (api *TestApi) Count(keywords []string) (int, error) {
	api.mutex.Lock()
	defer api.mutex.Unlock()

	count := 0
	for _, visit := range api.Visits {
		for _, keyword := range keywords {
			if strings.Contains(visit.URL, keyword) {
				count++
				break
			}
		}
	}
	return count, nil
}

func (api *TestApi) GetVisitsPage(keywords []string, page, size int) ([]*Visit, error) {
	api.mutex.Lock()
	defer api.mutex.Unlock()

	if len(api.Visits) <= size {
		return api.GetAllVisits()
	}

	foundVisits, err := api.GetVisits(keywords, -1)
	if err != nil {
		return nil, err
	}

	sort.Slice(foundVisits, func(i, j int) bool {
		return foundVisits[i].Timestamp.Before(foundVisits[j].Timestamp)
	})

	startIndex := (page - 1) * size
	endIndex := startIndex + size

	// overflow
	if startIndex >= len(foundVisits) {
		return []*Visit{}, errors.New("index overflow")
	}

	var visits []*Visit
	for i := startIndex; i < endIndex; i++ {
		visits = append(visits, foundVisits[i])
	}
	return visits, nil
}
