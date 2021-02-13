package netlog

import (
	"fmt"
	"strings"
)

// getQueryWhereCondition will make a SQL WHERE condition
// keywords starting with "-" will be filtered out with `url NOT LIKE ...`
// column - the name of the column to which the "like" is applied for
// source - the source of the netlog visit
func getQueryWhereCondition(column, source string, keywords []string) string {
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

	if source != "all" && len(keywords) == 0 {
		sbQueryLike.WriteString(fmt.Sprintf("WHERE source = '%s'", source))
	} else if source != "all" {
		sbQueryLike.WriteString(fmt.Sprintf("AND source = '%s'", source))
	}

	return sbQueryLike.String()
}
