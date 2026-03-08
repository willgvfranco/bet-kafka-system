package repository

import (
	"context"
	"errors"
	"time"

	"github.com/neus/bet-kafka-system/internal/domain"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

type MongoUserRepo struct {
	col *mongo.Collection
}

func NewMongoUserRepo(db *mongo.Database) *MongoUserRepo {
	return &MongoUserRepo{col: db.Collection("users")}
}

func (r *MongoUserRepo) FindByID(ctx context.Context, id string) (*domain.User, error) {
	var user domain.User
	err := r.col.FindOne(ctx, bson.M{"_id": id}).Decode(&user)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, nil
	}
	return &user, err
}

func (r *MongoUserRepo) UpdateBalance(ctx context.Context, id string, amount float64) (*domain.User, error) {
	filter := bson.M{"_id": id}
	if amount < 0 {
		filter["balance"] = bson.M{"$gte": -amount}
	}
	update := bson.M{
		"$inc": bson.M{"balance": amount},
		"$set": bson.M{"updated_at": time.Now()},
	}
	var user domain.User
	err := r.col.FindOneAndUpdate(ctx, filter, update).Decode(&user)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, errors.New("insufficient balance or user not found")
	}
	if err != nil {
		return nil, err
	}
	user.Balance += amount
	user.UpdatedAt = time.Now()
	return &user, nil
}

func (r *MongoUserRepo) Save(ctx context.Context, user *domain.User) error {
	_, err := r.col.InsertOne(ctx, user)
	return err
}

func (r *MongoUserRepo) FindAll(ctx context.Context) ([]domain.User, error) {
	cursor, err := r.col.Find(ctx, bson.M{})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)
	var users []domain.User
	return users, cursor.All(ctx, &users)
}
