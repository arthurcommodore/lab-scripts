package scripts

import (
	"github.com/gpt-utils/internal/logic"
	"go.mongodb.org/mongo-driver/mongo"
)

var (
	apiKey     string
	uri        string
	client     *mongo.Client
	collection *mongo.Collection
	rep        *logic.RepositoryMongo
)
