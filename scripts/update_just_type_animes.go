package scripts

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/gpt-utils/internal/logic"
	"github.com/joho/godotenv"
	"go.mongodb.org/mongo-driver/bson"
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

func UpdateJustTypeAnimes() {

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	animes, err := rep.ListPageAnime(ctx, 1, max, bson.M{"type": ""})
	if err != nil {
		log.Fatalf("Falha ao listar Anime: %v", err)
		return
	}

	ctx = context.Background()
	for _, anime := range animes {

		if len(anime.Title) < 5 {
			continue
		}

		resp, err := logic.FetchJustType(anime.Title)
		if err != nil {
			log.Fatal(err)
		}
		rep.UpdateOne(ctx, bson.M{"_id": anime.ID}, bson.M{"$set": bson.M{"type": resp.Data.Media.Type}})
		fmt.Printf("update %v with status: %v \n", anime.ID.Hex(), resp.Data.Media.Type)

		time.Sleep(time.Second * 3)
	}
}
