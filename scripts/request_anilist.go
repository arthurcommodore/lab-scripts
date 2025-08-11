package scripts

import (
	"github.com/gpt-utils/internal/logic"
)

func RequestAnimeList() {
	logic.FetchAllAnimeCharacters("Naruto", 1500)
}
