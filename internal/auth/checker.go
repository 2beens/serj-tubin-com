package auth

import "context"

var _ Checker = (*LoginChecker)(nil)
var _ Checker = (*LoginTestChecker)(nil)

type Checker interface {
	IsLogged(ctx context.Context, token string) (bool, error)
}
