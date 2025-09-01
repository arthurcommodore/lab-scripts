package scripts

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/gpt-utils/internal/dto"
	"github.com/gpt-utils/internal/logic"
	"github.com/gpt-utils/internal/logic/utils"
	"github.com/joho/godotenv"
	"go.mongodb.org/mongo-driver/bson"
)

const max = 1000000

func sanitizeFileName(name string) string {
	invalid := []string{"/", "\\", ":", "*", "?", "\"", "<", ">", "|"}
	for _, c := range invalid {
		name = strings.ReplaceAll(name, c, "_")
	}
	return name
}

func UpdateAnimes() {

	var uploads []struct {
		URL  string
		Path string
	}

	ctx := context.TODO()

	err := godotenv.Load()
	if err != nil {
		log.Fatalf("Falha ao carregar env: %v", err)
		return
	}

	logic.Connect("mongodb://localhost:27017/animeSearch")

	client := logic.GetDB() // deve retornar *mongo.Client
	collection := client.Database("animeSearch").Collection("animes")
	rep := logic.NewQueryAnimeMongo(collection)

	uploadAndDownload := func(url, path string) error {

		err = logic.DownloadImage(url, fmt.Sprintf("output/%s.jpg", path))
		if err != nil {
			log.Fatalf("Falha fazer download da imagem: %v", err)
		}

		addr := os.Getenv("FTP_ADDR")
		user := os.Getenv("FTP_USER")
		password := os.Getenv("FTP_PASSWORD")
		err = logic.SendFileFtpPasv(addr, user, password, fmt.Sprintf("output/%s.jpg", path))

		return err
	}

	animes, err := rep.ListPageAnime(ctx, 1, 5, bson.M{"aniListApi": bson.M{"$ne": true}})
	if err != nil {
		log.Fatalf("Falha ao listar Anime: %v", err)
		return
	}

	for _, anime := range animes {
		if len(anime.Title) == 0 {
			continue
		}

		time.Sleep(3500 * time.Millisecond)

		allEdges, fullResponse, err := logic.FetchAllAnimeCharacters(anime.Title, 25)
		if err != nil {
			log.Fatalf("Falha FetchAllAnimeCharacters: %v", err)
			continue
		}

		combined := logic.CombinedResult{
			FullResponse: fullResponse,
			AllEdges:     allEdges,
		}

		jsonData, err := json.MarshalIndent(combined, "", "  ")
		if err != nil {
			log.Fatalf("Erro ao converter para JSON: %v", err)
			return
		}

		safeTitle := sanitizeFileName(anime.Title)
		_, err = utils.SaveJSONToFile(jsonData, safeTitle, "/home/daym/Documentos/gpt-utils/output")
		if err != nil {
			log.Fatalf("Erro ao salvar arquivo: %v", err)
		}

		_, err = rep.UpdateOne(
			ctx, bson.M{"_id": anime.ID},
			bson.M{"$set": bson.M{
				"synopsis":        combined.FullResponse.Data.Media.Description,
				"countryOfOrigin": combined.FullResponse.Data.Media.CountryOfOrigin,
				"isAdult":         combined.FullResponse.Data.Media.IsAdult,
				"episodes":        combined.FullResponse.Data.Media.Episodes,
				"averageScore":    combined.FullResponse.Data.Media.AverageScore,
				"type":            combined.FullResponse.Data.Media.Type,
				"startDate":       combined.FullResponse.Data.Media.StartDate,
				"endDate":         combined.FullResponse.Data.Media.EndDate,
				"status":          combined.FullResponse.Data.Media.Status,
				"aniListApi":      true,
			}})

		if err != nil {
			log.Fatalf("Erro ao atualizar anime:%v", err)
			continue
		}

		for _, edge := range combined.AllEdges {

			var matchedCharacter dto.Character
			for _, character := range anime.Characters {
				if utils.CompareFirstWords(character.Name, edge.Node.Name.Full) {
					matchedCharacter = character
					break
				}
			}

			if matchedCharacter.Name != "" {
				_, err := rep.UpdateOne(ctx, bson.M{"characters.name": bson.M{"$regex": matchedCharacter.Name, "$options": "i"}}, bson.M{"$set": bson.M{
					"characters.$.bio":         edge.Node.Description,
					"characters.$.link":        edge.Node.SiteURL,
					"characters.$.age":         edge.Node.Age,
					"characters.$.dateOfBirth": edge.Node.DateOfBirth,
					"characters.$.aniListApi":  true,
				}})

				if err != nil {
					log.Fatalf("update character erro if CompareFirstWords: %v", err)
					continue
				}

				if matchedCharacter.PathImage == "" {

					uploads = append(uploads, struct {
						URL  string
						Path string
					}{
						URL:  edge.Node.Image.Large,
						Path: edge.Node.Name.Full,
					})

					_, err = rep.UpdateOne(ctx, bson.M{"characters.name": bson.M{"$regex": matchedCharacter.Name, "$options": "i"}}, bson.M{"$set": bson.M{
						"characters.$.PathImage": edge.Node.Image.Large,
					}})
					if err != nil {
						log.Fatalf("Falha ao atualizar characters.PathImage|Link: %v", err)
						continue
					}
				}
			} else {
				_, err = rep.UpdateOne(ctx, bson.M{"_id": anime.ID}, bson.M{
					"$push": bson.M{
						"characters": dto.Character{
							Name:        edge.Node.Name.Full,
							Age:         edge.Node.Age,
							DateOfBirth: edge.Node.DateOfBirth,
							Bio:         edge.Node.Description,
							PathImage:   edge.Node.Image.Large,
							Link:        edge.Node.SiteURL,
							AniListApi:  true,
						},
					},
				})

				if err != nil {
					log.Fatalf("Falha ao atualizar characters.PathImage|Link: %v", err)
					continue
				}

				uploads = append(uploads, struct {
					URL  string
					Path string
				}{
					URL:  edge.Node.Image.Large,
					Path: edge.Node.Name.Full,
				})
			}
		}
	}

	for _, up := range uploads {
		err := uploadAndDownload(up.URL, up.Path)
		if err != nil {
			log.Printf("Falha download image: %v", err)
			continue
		}
		fmt.Println("Upload concluído com sucesso!")
	}
}
