package netlog

type Api interface {
	CloseDB()
	AddVisit(visit *Visit) error
	GetVisits(keywords []string, byField string, limit int) ([]*Visit, error)
	CountAll() (int, error)
	Count(keywords []string, byField string) (int, error)
	GetVisitsPage(keywords []string, byField string, page int, size int) ([]*Visit, error)
}
