package netlog

type Api interface {
	CloseDB()
	AddVisit(visit *Visit) error
	GetVisits(keywords []string, field string, source string, limit int) ([]*Visit, error)
	CountAll() (int, error)
	Count(keywords []string, field string, source string) (int, error)
	GetVisitsPage(keywords []string, field string, source string, page int, size int) ([]*Visit, error)
}
