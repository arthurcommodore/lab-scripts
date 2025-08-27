package utils

import (
	"regexp"
	"strings"
)

func CompareFirstWords(a, b string) bool {
	re := regexp.MustCompile(`^\s*(\w+)`) // pega a primeira palavra ignorando espaços

	ma := re.FindStringSubmatch(a)
	mb := re.FindStringSubmatch(b)

	if len(ma) < 2 || len(mb) < 2 {
		return false
	}

	// compara ignorando maiúsculas/minúsculas
	return strings.EqualFold(ma[1], mb[1])
}
