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

func (r *RepositoryMongo) ListTagsCharacters(ctx context.Context) ([]string, error) {
	pipeline := mongo.Pipeline{
		// 1. Project apenas o campo characters.tags
		{{Key: "$project", Value: bson.D{
			{Key: "characters.tags", Value: 1},
		}}},
		// 2. Unwind do array characters
		{{Key: "$unwind", Value: bson.D{
			{Key: "path", Value: "$characters"},
			{Key: "preserveNullAndEmptyArrays", Value: false},
		}}},
		// 3. Filtrar tags não nulas
		{{Key: "$match", Value: bson.D{
			{Key: "characters.tags", Value: bson.D{{Key: "$ne", Value: nil}}},
		}}},
		// 4. Unwind do array tags
		{{Key: "$unwind", Value: bson.D{
			{Key: "path", Value: "$characters.tags"},
			{Key: "preserveNullAndEmptyArrays", Value: false},
		}}},
		// 5. Agrupar tudo em um único array com todos os tags distintos
		{{Key: "$group", Value: bson.D{
			{Key: "_id", Value: nil},
			{Key: "allTags", Value: bson.D{{Key: "$addToSet", Value: "$characters.tags"}}},
		}}},
		// 6. Project final para retornar apenas allTags
		{{Key: "$project", Value: bson.D{
			{Key: "_id", Value: 0},
			{Key: "allTags", Value: 1},
		}}},
	}

	cursor, err := r.Collection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var result struct {
		AllTags []string `bson:"allTags"`
	}

	if cursor.Next(ctx) {
		if err := cursor.Decode(&result); err != nil {
			return nil, err
		}
	}

	return result.AllTags, nil
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
