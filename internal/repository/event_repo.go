package repository

import (
	"context"
	"errors"

	"github.com/neus/bet-kafka-system/internal/domain"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

type MongoEventRepo struct {
	col *mongo.Collection
}

func NewMongoEventRepo(db *mongo.Database) *MongoEventRepo {
	return &MongoEventRepo{col: db.Collection("events")}
}

func (r *MongoEventRepo) FindByID(ctx context.Context, id string) (*domain.Event, error) {
	var event domain.Event
	err := r.col.FindOne(ctx, bson.M{"_id": id}).Decode(&event)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, nil
	}
	return &event, err
}

func (r *MongoEventRepo) FindAll(ctx context.Context) ([]domain.Event, error) {
	cursor, err := r.col.Find(ctx, bson.M{})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)
	var events []domain.Event
	return events, cursor.All(ctx, &events)
}

func (r *MongoEventRepo) FindByStatus(ctx context.Context, status domain.EventStatus) ([]domain.Event, error) {
	cursor, err := r.col.Find(ctx, bson.M{"status": status})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)
	var events []domain.Event
	return events, cursor.All(ctx, &events)
}

func (r *MongoEventRepo) Save(ctx context.Context, event *domain.Event) error {
	_, err := r.col.InsertOne(ctx, event)
	return err
}

func (r *MongoEventRepo) UpdateStatus(ctx context.Context, id string, status domain.EventStatus) error {
	_, err := r.col.UpdateOne(ctx, bson.M{"_id": id}, bson.M{"$set": bson.M{"status": status}})
	return err
}

func (r *MongoEventRepo) UpdateMarketStatus(ctx context.Context, eventID, marketID string, status domain.MarketStatus) error {
	filter := bson.M{"_id": eventID, "markets.market_id": marketID}
	update := bson.M{"$set": bson.M{"markets.$.status": status}}
	_, err := r.col.UpdateOne(ctx, filter, update)
	return err
}

func (r *MongoEventRepo) UpdateOutcomeResult(ctx context.Context, eventID, marketID, outcomeID string, result domain.OutcomeResult) error {
	filter := bson.M{"_id": eventID}
	update := bson.M{
		"$set": bson.M{
			"markets.$[m].outcomes.$[o].result": result,
		},
	}
	opts := mongoArrayFilters(
		bson.M{"m.market_id": marketID},
		bson.M{"o.outcome_id": outcomeID},
	)
	_, err := r.col.UpdateOne(ctx, filter, update, opts)
	return err
}
