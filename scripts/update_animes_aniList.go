package scripts

import (
	"context"
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
	"go.mongodb.org/mongo-driver/mongo"
)

const max = 1000000

func sanitizeFileName(name string) string {
	invalid := []string{"/", "\\", ":", "*", "?", "\"", "<", ">", "|", ".", "^", "$", "-"}
	for _, c := range invalid {
		name = strings.ReplaceAll(name, c, "_")
	}
	return name
}

var (
	client     *mongo.Client
	collection *mongo.Collection
	rep        *logic.RepositoryMongo
)

func init() {

	err := godotenv.Load()
	if err != nil {
		log.Println("Erro iniciar updateAnime")
		return
	}

	// conecta e inicializa o client só uma vez
	uri := os.Getenv("DB_URI")

	logic.Connect(uri)

	client = logic.GetDB()
	if client == nil {
		log.Fatal("Mongo client retornou nil em GetDB()")
	}

	collection = client.Database("animeSearch").Collection("animes")
	rep = logic.NewQueryAnimeMongo(collection)
}

type Upload struct {
	URL  string
	Path string
}

func UpdateAnimes() {

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	animes, err := rep.ListPageAnime(ctx, 1, max, bson.M{"aniListApi": bson.M{"$ne": true}})
	if err != nil {
		log.Fatalf("Falha ao listar Anime: %v", err)
		return
	}

	ctx = context.Background()
	for _, anime := range animes {

		var (
			uploadsCharacters []struct {
				URL  string
				Path string
			}
			uploadEpisodes []struct {
				URL  string
				Path string
			}
		)

		if len(anime.Title) < 5 {
			rep.UpdateOne(ctx, bson.M{"_id": anime.ID}, bson.M{"$set": bson.M{"aniListNotFound": true, "aniListApi": true}})
			continue
		}

		allEdges, fullResponse, err := logic.FetchAllAnimeCharacters(anime.Title, 25)
		time.Sleep(3 * time.Second)

		if allEdges == nil && fullResponse.Data.Media.Description == "" {
			fmt.Println("Not Found")
			rep.UpdateOne(ctx, bson.M{"_id": anime.ID}, bson.M{"$set": bson.M{"aniListApi": true}})
			continue
		}

		/*
			combined := struct {
				AllEdges    []logic.CharacterEdge
				FullReponse *logic.ResponseAnilist
			}{
				allEdges,
				fullResponse,
			}

			data, err := json.Marshal(combined)
			if err != nil {
				panic(err)
			}

			_, err = utils.SaveJSONToFile(data, "anilist", "output")

			if err != nil {
				log.Fatalf("Erro ao salvar arquivo: %v", err)
			}
		*/

		var docStreamingEpisodes []dto.StreamingEpisode
		for _, ep := range fullResponse.Data.Media.StreamingEpisodes {
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

		_, err = rep.UpdateOne(
			ctx, bson.M{"_id": anime.ID},
			bson.M{"$set": bson.M{
				"synopsis":          fullResponse.Data.Media.Description,
				"countryOfOrigin":   fullResponse.Data.Media.CountryOfOrigin,
				"isAdult":           fullResponse.Data.Media.IsAdult,
				"episodes":          fullResponse.Data.Media.Episodes,
				"averageScore":      fullResponse.Data.Media.AverageScore,
				"type":              fullResponse.Data.Media.Format,
				"startDate":         fullResponse.Data.Media.StartDate,
				"endDate":           fullResponse.Data.Media.EndDate,
				"status":            fullResponse.Data.Media.Status,
				"source":            fullResponse.Data.Media.Source,
				"duration":          fullResponse.Data.Media.Duration,
				"streamingEpisodes": docStreamingEpisodes,
				"studios":           fullResponse.Data.Media.Studios.Nodes,
				"format":            fullResponse.Data.Media.Format,
				"aniListApi":        true,
			}})

		if err != nil {
			log.Fatalf("Erro ao atualizar anime:%v", err)
			continue
		}

		updateCharacters(ctx, allEdges, anime, &uploadsCharacters)
		uploadImages(uploadsCharacters)
		uploadImages(uploadEpisodes)
	}
}

func updateCharacters(ctx context.Context, edges []logic.CharacterEdge, anime dto.Anime, uploadsCharacters *[]struct {
	URL  string
	Path string
}) {
	for _, edge := range edges {

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

				*uploadsCharacters = append(*uploadsCharacters, struct {
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
			_, err := rep.UpdateOne(ctx, bson.M{"_id": anime.ID}, bson.M{
				"$push": bson.M{
					"characters": dto.Character{
						ID:          primitive.NewObjectID(),
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

			*uploadsCharacters = append(*uploadsCharacters, struct {
				URL  string
				Path string
			}{
				URL:  edge.Node.Image.Large,
				Path: fmt.Sprintf("%s.jpg", utils.SanitizeFilename(edge.Node.Name.Full, "_")),
			})
		}
	}
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
		err := logic.DownloadImage(up.URL, fmt.Sprintf("output/%s", up.Path))
		if err != nil {
			log.Fatal(err)
			return
		}

		fmt.Printf("Download %s\n", up.Path)
		if err := ftpClient.UploadFile(fmt.Sprintf("output/%s", up.Path)); err != nil {
			log.Fatal(err)
			return
		}
		fmt.Println("send to ftp")

		time.Sleep(1 * time.Second)
		os.Remove(fmt.Sprintf("output/%s", up.Path))
	}
}
