package utils

import (
	"regexp"
	"strings"
)

func SanitizeFilename(name string, replacement string) string {
	// Caracteres inválidos no Windows: \ / : * ? " < > |
	invalidChars := `[\\/:*?"<>|]`

	// Substitui inválidos pelo replacement
	re := regexp.MustCompile(invalidChars)
	safe := re.ReplaceAllString(name, replacement)

	// Remove caracteres de controle ASCII (0-31)
	safe = strings.Map(func(r rune) rune {
		if r < 32 {
			return -1
		}
		return r
	}, safe)

	// Trim espaços extras no começo e fim
	safe = strings.TrimSpace(safe)

	// Se ficar vazio, usa um nome padrão
	if safe == "" {
		safe = "arquivo"
	}

	return safe
}
