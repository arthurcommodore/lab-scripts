package scripts

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/gpt-utils/internal/dto"
	"github.com/gpt-utils/internal/logic"
	"github.com/joho/godotenv"
	"go.mongodb.org/mongo-driver/bson"
)

type CharacterPrompt struct {
	Name            string `json:"name"`
	Bio             string `json:"bio"`
	Characteristics dto.Characteristics
	Tags            []string `json:"tags"`
}

// cleanGPTJSON deixa o JSON do GPT “parseável” mesmo com caracteres estranhos
func cleanGPTJSON(raw string) string {
	clean := strings.TrimSpace(raw)

	// Remove blocos de código Markdown de forma mais agressiva
	reCodeBlock := regexp.MustCompile("(?s)```(?:json)?(.*?)```")
	matches := reCodeBlock.FindStringSubmatch(clean)
	if len(matches) > 1 {
		clean = matches[1]
	}

	clean = strings.TrimSpace(clean)

	// Substitui aspas e apóstrofos “curly” e outros símbolos comuns
	replacements := map[string]string{
		"“": "\"",
		"”": "\"",
		"„": "\"",
		"‟": "\"",
		"‘": "'",
		"’": "'",
		"`": "'",
		"´": "'",
		"–": "-", // traço longo
		"—": "-", // traço médio
		"…": "...",
	}

	for old, new := range replacements {
		clean = strings.ReplaceAll(clean, old, new)
	}

	// Remove caracteres invisíveis ou de controle
	reInvisible := regexp.MustCompile(`[\x00-\x1F\x7F]`)
	clean = reInvisible.ReplaceAllString(clean, "")

	// Remove vírgulas finais antes de fechar objetos ou arrays
	reTrailingComma := regexp.MustCompile(`,(\s*[\]}])`)
	clean = reTrailingComma.ReplaceAllString(clean, "$1")

	// Remove comentários do GPT tipo // ou /* */
	reComments := regexp.MustCompile(`(?m)//.*$|/\*.*?\*/`)
	clean = reComments.ReplaceAllString(clean, "")

	// Remove espaços extras no início/fim de cada linha
	reExtraSpaces := regexp.MustCompile(`(?m)^\s+|\s+$`)
	clean = reExtraSpaces.ReplaceAllString(clean, "")

	return strings.TrimSpace(clean)
}

// Garante que animalEars seja "Yes" ou "No"
func normalizeAnimalEars(c *CharacterPrompt) {
	if strings.ToLower(c.Characteristics.AnimalEars) == "yes" {
		c.Characteristics.AnimalEars = "Yes"
	} else {
		c.Characteristics.AnimalEars = "No"
	}
}

func UpdateAnimeGptOptimized() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)

	if err := godotenv.Load(); err != nil {
		log.Println("Erro ao carregar .env:", err)
	}

	uri := os.Getenv("DB_URI")
	logic.Connect(uri)

	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		log.Fatal("OPENAI_API_KEY não definido")
	}

	client := logic.GetDB()
	collection := client.Database("animeSearch").Collection("animes")
	rep := logic.NewQueryAnimeMongo(collection)

	animes, err := rep.ListPageAnime(ctx, 1, 5, bson.M{"characters": bson.M{"$exists": true, "$ne": bson.A{}}})
	if err != nil {
		log.Fatal("Erro ao listar animes:", err)
	}

	tags, err := rep.ListTagsCharacters(ctx)
	if err != nil {
		log.Fatal("Erro ao listar tags:", err)
	}
	cancel()

	tagsJSON, _ := json.Marshal(tags)
	model := "gpt-4.1-mini"
	batchSize := 2

	for i := 0; i < len(animes); i += batchSize {
		end := i + batchSize
		if end > len(animes) {
			end = len(animes)
		}
		batch := animes[i:end]

		var wg sync.WaitGroup
		for _, anime := range batch {
			wg.Add(1)
			go func(anime dto.Anime) {
				defer wg.Done()

				if len(anime.Characters) == 0 {
					log.Printf("Anime %s não tem personagens, pulando...", anime.Title)
				}

				var charactersPrompt []CharacterPrompt
				for _, c := range anime.Characters {
					charactersPrompt = append(charactersPrompt, CharacterPrompt{
						Name:            c.Name,
						Bio:             c.Bio,
						Characteristics: c.Characteristics,
						Tags:            c.Tags,
					})
				}

				for idx := range charactersPrompt {
					normalizeAnimalEars(&charactersPrompt[idx])
				}

				charactersJSON, _ := json.Marshal(charactersPrompt)

				prompt := fmt.Sprintf(
					"Por favor, processe o array 'characters' com o seguinte formato:\n"+
						"{\n"+
						"  \"name\": \"string\",\n"+
						"  \"bio\": \"string\",\n"+
						"  \"characteristics\": {\n"+
						"    \"gender\": \"string\",\n"+
						"    \"eyeColor\": \"string\",\n"+
						"    \"hairColor\": \"string\",\n"+
						"    \"hairLength\": \"string\",\n"+
						"    \"apparentAge\": \"string\",\n"+
						"    \"animalEars\": \"string\"\n"+
						"  },\n"+
						"  \"tags\": [\"string\"]\n"+
						"}\n"+
						"- 'animalEars' deve ser sempre 'Yes' ou 'No'.\n"+
						"- Preencha 'tags' apenas se estiverem vazias, usando entre 4 e 10 tags de %s.\n"+
						"- Não adicione personagens novos.\n"+
						"- Apenas corrija ou preencha campos ausentes.\n"+
						"- Melhore 'bio' ou 'name' se necessário.\n"+
						"Personagens atuais: %s\n"+
						"Título: %s\nSinopse: %s\nStatus: %s\nEpisódios: %d",
					tagsJSON, charactersJSON, anime.Title, anime.Synopsis, anime.Status, anime.Episodes,
				)

				// timeout por anime
				ctx, cancel := context.WithTimeout(context.Background(), 20*time.Minute)
				defer cancel()

				response, err := logic.CallOpenAI(ctx, apiKey, model, prompt)
				if err != nil {
					log.Printf("Erro ao chamar GPT para anime %s: %v", anime.Title, err)
				}

				var apiResp struct {
					Output []struct {
						Content []struct {
							Text string `json:"text"`
						} `json:"content"`
					} `json:"output"`
				}

				if err := json.Unmarshal([]byte(response), &apiResp); err != nil {
					log.Printf("Erro ao parsear resposta para anime %s: %v\nResposta bruta: %s", anime.Title, err, response)
				}

				if len(apiResp.Output) == 0 || len(apiResp.Output[0].Content) == 0 {
					log.Printf("Resposta GPT vazia para anime %s", anime.Title)
				}

				clean := cleanGPTJSON(apiResp.Output[0].Content[0].Text)

				type GPTCharacterResponse struct {
					Name            string `json:"name"`
					Bio             string `json:"bio"`
					Characteristics struct {
						AnimalEars  string `json:"animalEars"`
						ApparentAge string `json:"apparentAge"`
						EyeColor    string `json:"eyeColor"`
						Gender      string `json:"gender"`
						HairColor   string `json:"hairColor"`
						HairLength  string `json:"hairLength"`
					} `json:"characteristics"`
					Tags []string `json:"tags"`
				}

				var generic GPTCharacterResponse
				if err := json.Unmarshal([]byte(clean), &generic); err != nil {
					log.Printf("JSON inválido para anime %s: %v\nConteúdo: %s", anime.Title, err, clean)
				}
			}(anime)
		}
		wg.Wait()
	}
}
