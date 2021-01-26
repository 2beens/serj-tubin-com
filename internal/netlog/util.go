package netlog

import (
	"fmt"
	"strings"
)

// getQueryLikeCondition will make a SQL "like" condition
// keywords starting with "-" will be filtered out with `url NOT LIKE ...`
func getQueryLikeCondition(keywords []string) string {
	var sbQueryLike strings.Builder
	if len(keywords) > 0 {
		sbQueryLike.WriteString("WHERE ")
		for i, word := range keywords {
			if strings.HasPrefix(word, "-") {
				word = strings.TrimPrefix(word, "-")
				sbQueryLike.WriteString(fmt.Sprintf("url NOT LIKE '%%%s%%' ", word))
			} else {
				sbQueryLike.WriteString(fmt.Sprintf("url LIKE '%%%s%%' ", word))
			}
			if i < len(keywords)-1 {
				sbQueryLike.WriteString("AND ")
			}
		}
	}
	return sbQueryLike.String()
}
