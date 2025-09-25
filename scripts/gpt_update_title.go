package scripts

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/gpt-utils/internal/dto"
	"github.com/gpt-utils/internal/logic"
	"github.com/gpt-utils/internal/logic/gpt"
	"github.com/joho/godotenv"
	"go.mongodb.org/mongo-driver/bson"
)

type CharacterPrompt struct {
	Name            string `json:"name"`
	Bio             string `json:"bio"`
	Characteristics dto.Characteristics
	Tags            []string `json:"tags"`
}

// Garante que animalEars seja "Yes" ou "No"
func normalizeAnimalEars(c *CharacterPrompt) {
	if strings.ToLower(c.Characteristics.AnimalEars) == "yes" {
		c.Characteristics.AnimalEars = "Yes"
	} else {
		c.Characteristics.AnimalEars = "No"
	}
}

func updateAnime(animes []dto.Anime) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

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

				prompt := fmt.Sprintf(
					"I need you to improve the synopsis: %s, for this anime: %s.\n"+
						"Only generate a valid JSON, exactly in this format:\n"+
						"{\n  \"synopsis\": \"string\"\n}\n"+
						"Do not include any other fields or information.",
					anime.Synopsis, anime.Title,
				)

				messages := []gpt.Message{
					{Role: "system", Content: "You are an anime expert and skilled writer. Your task is to improve and enhance anime synopses, making them engaging, clear, and accurate. Focus only on rewriting the synopsis without adding any unrelated information or extra fields."},
					{Role: "user", Content: prompt},
				}

				jsonFormat := gpt.ResponseFormat{
					Type: "json_schema",
					Json_schema: gpt.JSONSchema{
						Name: "Synopsis",
						Schema: gpt.Schema{
							Type: "object",
							Properties: map[string]interface{}{
								"synopsis": map[string]string{
									"type": "string",
								},
							},
							Required:             []string{"synopsis"},
							AdditionalProperties: false,
						},
					},
				}

				// "gpt-4o-2024-08-06"
				response, err := gpt.CallOpenAIStructOutPut(ctx, apiKey, model, messages, jsonFormat)
				if err != nil {
					log.Fatal(err)
				}

				if len(response.Choices) == 0 || len(response.Choices[0].Message.Content) == 0 {
					log.Printf("Resposta GPT vazia para anime %s", anime.Title)
				}

				var content struct {
					Synopsis string `json:"synopsis"`
				}
				if err := json.Unmarshal([]byte(response.Choices[0].Message.Content), &content); err != nil {
					log.Fatalf("error parser json in gpt.go %v", err)
				}

				rep.UpdateOne(ctx, bson.M{"_id": anime.ID}, bson.M{"$set": bson.M{"synopsis": content.Synopsis}})

			}(anime)
		}
		wg.Wait()
	}
}

func updateCharacters(animes []dto.Anime) {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Minute)
	defer cancel()
	tags, err := rep.ListTagsCharacters(ctx)
	if err != nil {
		log.Fatal("Erro ao listar tags:", err)
	}

	tagsJSON, _ := json.Marshal(tags)
	model := "gpt-4.1-mini"
	batchSize := 2

	type characterGpt struct {
		Type            string
		Name            string
		Bio             string
		Characteristics struct {
			gender      string
			eyeColor    string
			hairColor   string
			hairLength  string
			apparentAge string
			animalEars  string
		}
		Tags []string
	}

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
					return
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
						"- Preencha 'characteristics' apenas se estiverem vazias.\n"+
						"- Não adicione personagens novos.\n"+
						"- Apenas corrija ou preencha campos ausentes.\n"+
						"- O campo 'name' **não deve ser alterado em hipótese alguma**.\n"+
						"- Melhore apenas a 'bio' se necessário.\n"+
						"Personagens atuais: %s\n"+
						"Título: %s\nSinopse: %s\nStatus: %s\n",
					tagsJSON, charactersJSON, anime.Title, anime.Synopsis, anime.Status,
				)

				// timeout por anime
				ctx, cancel := context.WithTimeout(context.Background(), 20*time.Minute)
				defer cancel()

				messages := []gpt.Message{
					{Role: "system", Content: "You are a skilled editor and anime expert. Your role is to review and refine the information about these characters, improving clarity, accuracy, and engagement while keeping the content faithful to the original source."},
					{Role: "user", Content: prompt},
				}

				type Prop struct {
					Type string `json:"type"`
				}

				responseFormat := gpt.ResponseFormat{
					Type: "json_schema",
					Json_schema: gpt.JSONSchema{
						Name: "characters_updated",
						Schema: gpt.Schema{
							Type: "object",
							Properties: map[string]interface{}{
								"characters": map[string]interface{}{
									"type": "array",
									"items": map[string]interface{}{
										"type": "object",
										"properties": map[string]interface{}{
											"name": Prop{Type: "string"},
											"bio":  Prop{Type: "string"},
											"characteristics": map[string]interface{}{
												"gender":      Prop{Type: "string"},
												"eyeColor":    Prop{Type: "string"},
												"hairColor":   Prop{Type: "string"},
												"hairLength":  Prop{Type: "string"},
												"apparentAge": Prop{Type: "string"},
												"animalEars":  Prop{Type: "string"},
											},
											"tags": struct {
												Type  string `json:"type"`
												Items Prop   `json:"items"`
											}{
												Type:  "array",
												Items: Prop{Type: "string"},
											},
										},
										"required": []string{"name", "bio", "characteristics", "tags"},
									},
								},
							},
							Required:             []string{"characters"},
							AdditionalProperties: false,
						},
					},
				}

				response, err := gpt.CallOpenAIStructOutPut(ctx, apiKey, model, messages, responseFormat)
				if err != nil {
					log.Fatal(err)
				}

				if len(response.Choices) == 0 || len(response.Choices[0].Message.Content) == 0 {
					log.Printf("Resposta GPT vazia para anime %s", anime.Title)
				}

				type contentGpt struct {
					Name            string
					Bio             string
					Characteristics struct {
						Gender      string
						EyeColor    string
						HairColor   string
						HairLength  string
						ApparentAge string
						AnimalEars  string
					}
					Tags []string
				}
				// ...existing code...
				type CharactersResponse struct {
					Characters []contentGpt `json:"characters"`
				}
				// ...existing code...

				var resp CharactersResponse
				if err := json.Unmarshal([]byte(response.Choices[0].Message.Content), &resp); err != nil {
					log.Fatalf("error parser json in gpt.go %v", err)
				}
				charactersGpt := resp.Characters

				var charactersUpdated []dto.Character
				for _, characterGpt := range charactersGpt {
					for _, character := range anime.Characters {
						if characterGpt.Name == character.Name {
							var tags []string
							if len(characterGpt.Tags) > 0 {
								tags = characterGpt.Tags
							} else {
								tags = character.Tags
							}
							doc := dto.Character{
								ID:              character.ID,
								Tags:            tags,
								Name:            character.Name,
								PathImage:       character.PathImage,
								Link:            character.Link,
								Bio:             characterGpt.Bio,
								Age:             character.Age,
								DateOfBirth:     character.DateOfBirth,
								AniListApi:      true,
								VoiceActors:     character.VoiceActors,
								Characteristics: characterGpt.Characteristics,
							}
							charactersUpdated = append(charactersUpdated, doc)
						}
					}
				}

				if len(charactersUpdated) > 0 {
					rep.UpdateOne(ctx, bson.M{"_id": anime.ID}, bson.M{"$set": bson.M{"characters": charactersUpdated, "chatGpt": true}})
					fmt.Printf("Atualizado %s \n", anime.ID.Hex())
				}

			}(anime)
		}
		wg.Wait()
	}
}

func load() {

	if err := godotenv.Load(); err != nil {
		log.Println("Erro ao carregar .env:", err)
	}

	uri = os.Getenv("DB_URI")
	logic.Connect(uri)

	apiKey = os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		log.Fatal("OPENAI_API_KEY não definido")
	}

	client = logic.GetDB()
	collection = client.Database("animeSearch").Collection("animes")
	rep = logic.NewQueryAnimeMongo(collection)
}

func UpdateAnimeGptOptimized() {

	load()

	var page int = 1
	for {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()
		animes, err := rep.ListPageAnime(ctx, page, 10, bson.M{"characters": bson.M{"$exists": true, "$ne": bson.A{}}, "chatGpt": bson.M{"$ne": true}})
		if err != nil {
			log.Fatal("Erro ao listar animes:", err)
		}

		if len(animes) < 1 {
			break
		}

		updateAnime(animes)
		updateCharacters(animes)
		page++
	}

}
