package netlog

import (
	"fmt"
	"strings"
)

func getQueryLikeCondition(keywords []string) string {
	var sbQueryLike strings.Builder
	if len(keywords) > 0 {
		sbQueryLike.WriteString("WHERE ")
		for i, word := range keywords {
			sbQueryLike.WriteString(fmt.Sprintf("url LIKE '%%%s%%' ", word))
			if i < len(keywords)-1 {
				sbQueryLike.WriteString("AND ")
			}
		}
	}
	return sbQueryLike.String()
}
