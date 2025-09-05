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
	"go.mongodb.org/mongo-driver/bson/primitive"
)

const max = 1000000

func sanitizeFileName(name string) string {
	invalid := []string{"/", "\\", ":", "*", "?", "\"", "<", ">", "|", ".", "^", "$", "-"}
	for _, c := range invalid {
		name = strings.ReplaceAll(name, c, "_")
	}
	return name
}

func UpdateAnimes() {

	var uploadsCharacters []struct {
		URL  string
		Path string
	}

	var uploadEpisodes []struct {
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

	animes, err := rep.ListPageAnime(ctx, 1, 15, bson.M{"aniListApi": bson.M{"$ne": true}})
	if err != nil {
		log.Fatalf("Falha ao listar Anime: %v", err)
		return
	}

	for _, anime := range animes {
		if len(anime.Title) < 5 {
			rep.UpdateOne(ctx, bson.M{"_id": anime.ID}, bson.M{"$set": bson.M{"aniListNotFound": true, "aniListApi": true}})
			continue
		}

		allEdges, fullResponse, err := logic.FetchAllAnimeCharacters(anime.Title, 25)
		if err != nil {
			log.Fatalf("Falha FetchAllAnimeCharacters: %v", err)
			continue
		}

		if allEdges == nil && fullResponse.Data.Media.Description == "" {
			fmt.Println("Not Found")
			continue
		}

		combined := logic.CombinedResultAniList{
			FullResponse: fullResponse,
			AllEdges:     allEdges,
		}

		var docStreamingEpisodes []dto.StreamingEpisode
		for _, ep := range combined.FullResponse.Data.Media.StreamingEpisodes {
			uploadEpisodes = append(uploadEpisodes, struct {
				URL  string
				Path string
			}{
				URL:  ep.Thumbnail,
				Path: fmt.Sprintf("%s.jpg", sanitizeFileName(ep.Title)),
			})

			doc := dto.StreamingEpisode{
				ID:        primitive.NewObjectID(),
				Site:      ep.Site,
				PathImage: fmt.Sprintf("%s.jpg", sanitizeFileName(ep.Title)),
				Title:     ep.Title,
			}
			docStreamingEpisodes = append(docStreamingEpisodes, doc)
		}

		jsonData, err := json.MarshalIndent(combined, "", "  ")
		if err != nil {
			log.Fatalf("Erro ao converter para JSON: %v", err)
			return
		}

		safeTitle := sanitizeFileName(anime.Title)
		_, err = utils.SaveJSONToFile(jsonData, safeTitle, "output")
		if err != nil {
			log.Fatalf("Erro ao salvar arquivo: %v", err)
		}

		_, err = rep.UpdateOne(
			ctx, bson.M{"_id": anime.ID},
			bson.M{"$set": bson.M{
				"synopsis":          combined.FullResponse.Data.Media.Description,
				"countryOfOrigin":   combined.FullResponse.Data.Media.CountryOfOrigin,
				"isAdult":           combined.FullResponse.Data.Media.IsAdult,
				"episodes":          combined.FullResponse.Data.Media.Episodes,
				"averageScore":      combined.FullResponse.Data.Media.AverageScore,
				"type":              combined.FullResponse.Data.Media.Format,
				"startDate":         combined.FullResponse.Data.Media.StartDate,
				"endDate":           combined.FullResponse.Data.Media.EndDate,
				"status":            combined.FullResponse.Data.Media.Status,
				"source":            combined.FullResponse.Data.Media.Source,
				"duration":          combined.FullResponse.Data.Media.Duration,
				"streamingEpisodes": docStreamingEpisodes,
				"studios":           combined.FullResponse.Data.Media.Studios.Nodes,
				"format":            combined.FullResponse.Data.Media.Format,
				"relations":         combined.FullResponse.Data.Media.Relations.Nodes,
				"aniListApi":        true,
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
					"characters.$.voiceActors": edge.VoiceActors,
					"characters.$.aniListApi":  true,
				}})

				if err != nil {
					log.Fatalf("update character erro if CompareFirstWords: %v", err)
					continue
				}

				if matchedCharacter.PathImage == "" {

					uploadsCharacters = append(uploadsCharacters, struct {
						URL  string
						Path string
					}{
						URL:  edge.Node.Image.Large,
						Path: fmt.Sprintf("%s.jpg", utils.SanitizeFilename(edge.Node.Name.Full, "_")),
					})

					_, err = rep.UpdateOne(ctx, bson.M{"characters.name": bson.M{"$regex": matchedCharacter.Name, "$options": "i"}}, bson.M{"$set": bson.M{
						"characters.$.PathImage": fmt.Sprintf("%s.jpg", utils.SanitizeFilename(edge.Node.Name.Full, "_")),
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
							PathImage:   fmt.Sprintf("%s.jpg", utils.SanitizeFilename(edge.Node.Name.Full, "_")),
							Link:        edge.Node.SiteURL,
							AniListApi:  true,
						},
					},
				})

				if err != nil {
					log.Fatalf("Falha ao atualizar characters.PathImage|Link: %v", err)
					continue
				}

				uploadsCharacters = append(uploadsCharacters, struct {
					URL  string
					Path string
				}{
					URL:  edge.Node.Image.Large,
					Path: fmt.Sprintf("%s.jpg", utils.SanitizeFilename(edge.Node.Name.Full, "_")),
				})
			}
		}
	}

	uploadImages(uploadsCharacters)
	uploadImages(uploadEpisodes)

	time.Sleep(3 * time.Second)
}

func uploadImages(uploads []struct {
	URL  string
	Path string
}) {
	addr := os.Getenv("FTP_ADDR")
	user := os.Getenv("FTP_USER")
	password := os.Getenv("FTP_PASSWORD")

	ftpClient, err := logic.NewFtpClient(addr, user, password)
	if err != nil {
		log.Fatal(err)
	}

	defer ftpClient.Close()

	for _, up := range uploads {
		err := logic.DownloadImage(up.URL, fmt.Sprintf("/home/daym/Documentos/aniListImages/%s", up.Path))
		if err != nil {
			log.Fatal(err)
			return
		}

		fmt.Printf("Download %s\n", up.Path)
		if err := ftpClient.UploadFile(fmt.Sprintf("/home/daym/Documentos/aniListImages/%s", up.Path)); err != nil {
			log.Fatal(err)
			return
		}
		fmt.Println("send to ftp")

		time.Sleep(1 * time.Second)
	}
}
