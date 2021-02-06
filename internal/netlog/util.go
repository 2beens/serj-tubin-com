package netlog

import (
	"fmt"
	"strings"
)

// getQueryLikeCondition will make a SQL "like" condition
// keywords starting with "-" will be filtered out with `url NOT LIKE ...`
// column - the name of the column to which the "like" is applied for
func getQueryLikeCondition(column string, keywords []string) string {
	var sbQueryLike strings.Builder
	if len(keywords) > 0 {
		sbQueryLike.WriteString("WHERE ")
		for i, word := range keywords {
			if strings.HasPrefix(word, "-") {
				word = strings.TrimPrefix(word, "-")
				sbQueryLike.WriteString(fmt.Sprintf("%s NOT LIKE '%%%s%%' ", column, word))
			} else {
				sbQueryLike.WriteString(fmt.Sprintf("%s LIKE '%%%s%%' ", column, word))
			}
			if i < len(keywords)-1 {
				sbQueryLike.WriteString("AND ")
			}
		}
	}
	return sbQueryLike.String()
}
