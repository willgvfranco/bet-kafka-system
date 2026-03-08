package repository

import (
	"context"
	"errors"
	"time"

	"github.com/neus/bet-kafka-system/internal/domain"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

type MongoBetRepo struct {
	col *mongo.Collection
}

func NewMongoBetRepo(db *mongo.Database) *MongoBetRepo {
	return &MongoBetRepo{col: db.Collection("bets")}
}

func (r *MongoBetRepo) Save(ctx context.Context, bet *domain.Bet) error {
	_, err := r.col.InsertOne(ctx, bet)
	return err
}

func (r *MongoBetRepo) FindByID(ctx context.Context, id string) (*domain.Bet, error) {
	var bet domain.Bet
	err := r.col.FindOne(ctx, bson.M{"_id": id}).Decode(&bet)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, nil
	}
	return &bet, err
}

func (r *MongoBetRepo) FindByUser(ctx context.Context, userID string) ([]domain.Bet, error) {
	cursor, err := r.col.Find(ctx, bson.M{"user_id": userID})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)
	var bets []domain.Bet
	return bets, cursor.All(ctx, &bets)
}

func (r *MongoBetRepo) FindByMarket(ctx context.Context, marketID string) ([]domain.Bet, error) {
	cursor, err := r.col.Find(ctx, bson.M{"market_id": marketID, "status": domain.BetStatusPending})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)
	var bets []domain.Bet
	return bets, cursor.All(ctx, &bets)
}

func (r *MongoBetRepo) UpdateStatus(ctx context.Context, id string, status domain.BetStatus, settledAt time.Time) error {
	_, err := r.col.UpdateOne(ctx, bson.M{"_id": id}, bson.M{
		"$set": bson.M{"status": status, "settled_at": settledAt},
	})
	return err
}
