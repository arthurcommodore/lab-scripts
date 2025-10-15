package scripts

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/gpt-utils/internal/logic"
	"go.mongodb.org/mongo-driver/bson"
)

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

		resp, err := logic.FetchJustFormat(anime.Title)
		if err != nil {
			log.Fatal(err)
		}
		rep.UpdateOne(ctx, bson.M{"_id": anime.ID}, bson.M{"$set": bson.M{"type": resp.Data.Media.Format}})
		fmt.Printf("update %v with status: %v \n", anime.ID.Hex(), resp.Data.Media.Format)

		time.Sleep(time.Second * 3)
	}
}
