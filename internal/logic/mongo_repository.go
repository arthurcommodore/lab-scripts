package logic

import (
	"context"

	"github.com/gpt-utils/internal/dto"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type RepositoryMongo struct {
	Collection *mongo.Collection
	factory    func() dto.Document
}

func NewRepositoryMongo(collection *mongo.Collection, factory func() dto.Document) *RepositoryMongo {
	return &RepositoryMongo{
		Collection: collection,
		factory:    factory,
	}
}

func (r *RepositoryMongo) InsertOne(ctx context.Context, doc dto.Document) error {
	_, err := r.Collection.InsertOne(ctx, doc)
	return err
}

func (r *RepositoryMongo) List(
	ctx context.Context,
	query interface{},
	projection interface{},
	limit *int,
) ([]dto.Document, error) {
	if projection == nil {
		projection = bson.M{}
	}
	findOptions := options.Find().SetProjection(projection)
	if limit != nil {
		findOptions.SetLimit(int64(*limit))
	}

	cursor, err := r.Collection.Find(ctx, query, findOptions)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var docs []dto.Document
	for cursor.Next(ctx) {
		doc := r.factory() // cria nova instância
		if err := cursor.Decode(doc); err != nil {
			return nil, err
		}
		docs = append(docs, doc)
	}
	return docs, cursor.Err()
}

func (r *RepositoryMongo) UpdateOne(
	ctx context.Context,
	filter interface{},
	update interface{},
) (modifiedCount int64, err error) {

	result, err := r.Collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return 0, err
	}
	return result.ModifiedCount, nil
}

func (r *RepositoryMongo) UpdateMany(
	ctx context.Context,
	filter interface{},
	update interface{},
) (matched int64, modified int64, err error) {

	result, err := r.Collection.UpdateMany(ctx, filter, update)
	if err != nil {
		return 0, 0, err
	}
	return result.MatchedCount, result.ModifiedCount, nil
}

func (r *RepositoryMongo) Count(ctx context.Context, filter bson.M) (int, error) {
	count, err := r.Collection.CountDocuments(ctx, filter)
	return int(count), err
}
