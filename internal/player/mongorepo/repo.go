package mongorepo

import (
	"context"
	"errors"
	"fmt"
	"time"

	"madbox-player-profile/internal/player"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

type Repo struct {
	coll *mongo.Collection
}

func NewRepo(db *mongo.Database) *Repo {
	return &Repo{coll: db.Collection("players")}
}

// Create calls UpdateOne to avoid querying the entire document.
// It must be fetched later manually when needed.
// Returns true if the new player was inserted for the first time.
func (r *Repo) Create(ctx context.Context, p player.Player) (bool, error) {
	// can we avoid hardcoded column names
	filter := bson.M{"_id": p.ID}

	p.CreatedAt = p.CreatedAt.Truncate(time.Millisecond)
	p.UpdatedAt = p.UpdatedAt.Truncate(time.Millisecond)
	update := bson.M{"$setOnInsert": p}

	opts := options.UpdateOne().SetUpsert(true)

	res, err := r.coll.UpdateOne(ctx, filter, update, opts)
	if err != nil {
		return false, fmt.Errorf("upsert player error: %v", err)
	}

	created := res.UpsertedCount == 1
	return created, nil
}

func (r *Repo) Get(ctx context.Context, ID string) (player.Player, error) {
	var p player.Player
	filter := bson.M{"_id": ID}

	err := r.coll.FindOne(ctx, filter).Decode(&p)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return player.Player{}, player.ErrNotFound
	}
	if err != nil {
		return player.Player{}, fmt.Errorf("find player failed for id: %v", ID)
	}
	return p, nil
}

func (r *Repo) Exists(ctx context.Context, id string) (bool, error) {
	// Don't query the document, only if it exists
	err := r.coll.FindOne(ctx, bson.M{"_id": id},
		options.FindOne().SetProjection(bson.M{"_id": 1})).Err()
	if errors.Is(err, mongo.ErrNoDocuments) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("player exists: %w", err)
	}
	return true, nil
}

func (r *Repo) Update(ctx context.Context, id string, patch player.Patch, updatedAt time.Time) (player.Player, error) {
	set := bson.M{"updated_at": updatedAt}
	if patch.DisplayName != nil {
		set["display_name"] = *patch.DisplayName
	}
	if patch.Country != nil {
		set["country"] = *patch.Country
	}

	var p player.Player
	err := r.coll.FindOneAndUpdate(ctx,
		bson.M{"_id": id},
		bson.M{"$set": set},
		options.FindOneAndUpdate().SetReturnDocument(options.After),
	).Decode(&p)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return player.Player{}, player.ErrNotFound
	}
	if err != nil {
		return player.Player{}, fmt.Errorf("update player: %w", err)
	}
	return p, nil
}
