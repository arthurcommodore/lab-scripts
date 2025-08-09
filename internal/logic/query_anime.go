package logic

import (
	"context"

	"github.com/gpt-utils/internal/dto"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

func NewQueryAnimeMongo(collection *mongo.Collection) *RepositoryMongo {
	return NewRepositoryMongo(collection, func() dto.Document {
		return &dto.Anime{}
	})
}

func (r *RepositoryMongo) ListPageAnime(ctx context.Context, page int, pageSize int, query bson.M) ([]dto.Anime, error) {
	skip := (page - 1) * pageSize

	pipeline := mongo.Pipeline{
		bson.D{{Key: "$match", Value: query}},
		bson.D{{Key: "$skip", Value: int64(skip)}},
		bson.D{{Key: "$limit", Value: int64(pageSize)}},
	}

	cursor, err := r.Collection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var animes []dto.Anime
	for cursor.Next(ctx) {
		select {
		case <-ctx.Done():
			cursor.Close(ctx)
			return nil, ctx.Err()
		default:
			var anime dto.Anime
			if err := cursor.Decode(&anime); err != nil {
				return nil, err
			}
			animes = append(animes, anime)
		}
	}
	return animes, cursor.Err()
}
