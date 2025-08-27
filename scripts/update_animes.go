package scripts

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/gpt-utils/internal/dto"
	"github.com/gpt-utils/internal/logic"
	"github.com/gpt-utils/internal/logic/utils"
	"github.com/joho/godotenv"
	"go.mongodb.org/mongo-driver/bson"
)

const max = 1000000

func UpdateAnimes() {

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

	animes, err := rep.ListPageAnime(ctx, 1, max, nil)
	if err != nil {
		log.Fatalf("Falha ao listar Anime: %v", err)
		return
	}

	for _, anime := range animes {
		if len(anime.Title) == 0 {
			continue
		}

		allEdges, fullResponse, err := logic.FetchAllAnimeCharacters(anime.Title, 25)
		if err != nil {
			log.Fatalf("Falha FetchAllAnimeCharacters: %v", err)
			return
		}

		combined := logic.CombinedResult{
			FullResponse: fullResponse,
			AllEdges:     allEdges,
		}

		rep.UpdateOne(
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
			}})

		for _, edge := range combined.AllEdges {
			for _, character := range anime.Characters {
				if utils.CompareFirstWords(character.Name, edge.Node.Name.Full) {

					_, err := rep.UpdateOne(ctx, bson.M{"characters.name": bson.M{"$regex": character.Name, "$options": "i"}}, bson.M{"$set": bson.M{
						"characters.$.bio":         edge.Node.Description,
						"characters.$.Link":        edge.Node.SiteURL,
						"characters.$.Age":         edge.Node.Age,
						"characters.$.DateOfBirth": edge.Node.DateOfBirth,
					}})

					if err != nil {
						log.Fatalf("Falha ao atualizar characters.bio: %v", err)
						continue
					}

					if character.PathImage == "" {

						err = uploadAndDownload(edge.Node.Image.Large, edge.Node.Name.Full)
						if err != nil {
							log.Fatalf("Falha download image update character: %v", err)
							continue
						}
						_, err = rep.UpdateOne(ctx, bson.M{"characters.name": bson.M{"$regex": character.Name, "$options": "i"}}, bson.M{"$set": bson.M{
							"characters.$.PathImage": edge.Node.Image.Large,
						}})
						if err != nil {
							log.Fatalf("Falha ao atualizar characters.PathImage|Link: %v", err)
							continue
						}
					}

				} else {

					_, err = rep.UpdateOne(ctx, bson.M{"_id": anime.ID}, bson.M{"$set": bson.M{
						"characters": append(anime.Characters, dto.Character{
							Name:        edge.Node.Name.Full,
							Age:         edge.Node.Age,
							DateOfBirth: edge.Node.DateOfBirth,
							Bio:         edge.Node.Description,
							PathImage:   edge.Node.Image.Large,
							Link:        edge.Node.SiteURL,
						}),
					}})

					if err != nil {
						log.Fatalf("Falha ao atualizar characters.PathImage|Link: %v", err)
						continue
					}

					err = uploadAndDownload(edge.Node.Image.Large, edge.Node.Name.Full)

					if err != nil {
						log.Fatalf("Falha download image create character: %v", err)
					} else {
						fmt.Println("Upload concluído com sucesso!")
					}
				}
				fmt.Printf("Atualizado: %s\n", character.Name)
			}
		}
	}
}
