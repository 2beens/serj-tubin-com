package auth

var _ Checker = (*LoginChecker)(nil)
var _ Checker = (*LoginTestChecker)(nil)

type Checker interface {
	IsLogged(token string) (bool, error)
}
