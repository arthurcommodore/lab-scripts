package scripts

import (
	"log"
	"os"

	"github.com/gpt-utils/internal/logic"
	"github.com/joho/godotenv"
	"go.mongodb.org/mongo-driver/mongo"
)

var (
	apiKey     string
	uri        string
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
