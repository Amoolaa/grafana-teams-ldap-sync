package sync

import (
	"fmt"
	"strings"
)

func parens(filters []string) []string {
	c := make([]string, len(filters))
	for i, f := range filters {
		c[i] = paren(f)
	}
	return c
}

func paren(q string) string {
	if strings.HasPrefix(q, "(") && strings.HasSuffix(q, ")") {
		return q
	}
	return fmt.Sprintf("(%s)", q)
}

func op(operation string, filters ...string) string {
	return paren(operation + strings.Join(parens(filters), ""))
}

func or(filters ...string) string {
	return op("|", filters...)
}

func and(filters ...string) string {
	return op("&", filters...)
}
