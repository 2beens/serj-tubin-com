package netlog

type Api interface {
	CloseDB()
	AddVisit(visit *Visit) error
	GetVisits(keywords []string, limit int) ([]*Visit, error)
	CountAll() (int, error)
	Count(keywords []string) (int, error)
	GetVisitsPage(keywords []string, page, size int) ([]*Visit, error)
}
