package auth

type LoginTestChecker struct {
	LoggedSessions map[string]bool
}

func NewLoginTestChecker() *LoginTestChecker {
	return &LoginTestChecker{
		map[string]bool{},
	}
}

func (c *LoginTestChecker) IsLogged(token string) (bool, error) {
	if logged, ok := c.LoggedSessions[token]; !ok {
		return false, nil
	} else {
		return logged, nil
	}
}
