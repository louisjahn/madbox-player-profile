package mongorepo

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"madbox-player-profile/internal/event"
)

type Repo struct{ coll *mongo.Collection }

func NewRepo(db *mongo.Database) *Repo {
	return &Repo{coll: db.Collection("events")}
}

// EnsureIndexes is idempotent; call at startup. Both read paths
// (recent list, windowed count) filter on player_id and sort/range on ts.
func (r *Repo) EnsureIndexes(ctx context.Context) error {
	_, err := r.coll.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: bson.D{{Key: "player_id", Value: 1}, {Key: "ts", Value: -1}},
	})
	if err != nil {
		return fmt.Errorf("create events index: %w", err)
	}
	return nil
}

func (r *Repo) Insert(ctx context.Context, e event.Event) (event.Event, error) {
	res, err := r.coll.InsertOne(ctx, e)
	if err != nil {
		return event.Event{}, fmt.Errorf("insert event: %w", err)
	}
	e.ID = res.InsertedID.(bson.ObjectID)
	return e, nil
}

func (r *Repo) ListRecent(ctx context.Context, playerID string, f event.Filter) ([]event.Event, error) {
	filter := bson.M{"player_id": playerID}
	if f.Type != "" {
		filter["type"] = f.Type
	}
	// _id as a tiebreaker keeps ordering stable for equal ts.
	opts := options.Find().
		SetSort(bson.D{{Key: "ts", Value: -1}, {Key: "_id", Value: -1}}).
		SetLimit(int64(f.Limit))

	cur, err := r.coll.Find(ctx, filter, opts)
	if err != nil {
		return nil, fmt.Errorf("find events: %w", err)
	}
	defer cur.Close(ctx)

	events := make([]event.Event, 0, f.Limit)
	if err := cur.All(ctx, &events); err != nil {
		return nil, fmt.Errorf("decode events: %w", err)
	}
	if events == nil {
		events = []event.Event{}
	}
	return events, nil
}

func (r *Repo) CountByType(ctx context.Context, playerID string, since time.Time) (map[string]int64, error) {
	pipeline := mongo.Pipeline{
		{{Key: "$match", Value: bson.D{
			{Key: "player_id", Value: playerID},
			{Key: "ts", Value: bson.D{{Key: "$gte", Value: since}}},
		}}},
		{{Key: "$group", Value: bson.D{
			{Key: "_id", Value: "$type"},
			{Key: "count", Value: bson.D{{Key: "$sum", Value: 1}}},
		}}},
	}

	cur, err := r.coll.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, fmt.Errorf("aggregate event counts: %w", err)
	}
	defer cur.Close(ctx)

	var rows []struct {
		Type  string `bson:"_id"`
		Count int64  `bson:"count"`
	}
	if err := cur.All(ctx, &rows); err != nil {
		return nil, fmt.Errorf("decode event counts: %w", err)
	}

	counts := make(map[string]int64, len(rows))
	for _, row := range rows {
		counts[row.Type] = row.Count
	}
	return counts, nil
}
