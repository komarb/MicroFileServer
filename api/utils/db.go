package utils

import (
	"strings"
)

func GetDbName(dburi string) string {
	l := 0
	i := 0
	for i != 3 {
		l += strings.Index(dburi[l:], "/")+1
		i++
	}
	r := strings.Index(dburi, "?")
	if r == -1 {
		r = len(dburi)
	}
	return dburi[l:r]
}