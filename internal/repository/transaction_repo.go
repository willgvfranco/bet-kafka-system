package repository

import (
	"context"

	"github.com/neus/bet-kafka-system/internal/domain"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type MongoTransactionRepo struct {
	col *mongo.Collection
}

func NewMongoTransactionRepo(db *mongo.Database) *MongoTransactionRepo {
	return &MongoTransactionRepo{col: db.Collection("transactions")}
}

func (r *MongoTransactionRepo) Save(ctx context.Context, tx *domain.Transaction) error {
	_, err := r.col.InsertOne(ctx, tx)
	return err
}

func (r *MongoTransactionRepo) FindByUser(ctx context.Context, userID string) ([]domain.Transaction, error) {
	opts := options.Find().SetSort(bson.M{"created_at": -1}).SetLimit(50)
	cursor, err := r.col.Find(ctx, bson.M{"user_id": userID}, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)
	var txs []domain.Transaction
	return txs, cursor.All(ctx, &txs)
}
