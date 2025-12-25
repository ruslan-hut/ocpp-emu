package storage

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// ChangeEventType represents the type of change event
type ChangeEventType string

const (
	ChangeEventInsert  ChangeEventType = "insert"
	ChangeEventUpdate  ChangeEventType = "update"
	ChangeEventReplace ChangeEventType = "replace"
	ChangeEventDelete  ChangeEventType = "delete"
)

// ChangeEvent represents a change event from MongoDB Change Streams
type ChangeEvent struct {
	Type          ChangeEventType        `json:"type"`
	Collection    string                 `json:"collection"`
	DocumentID    string                 `json:"documentId,omitempty"`
	FullDocument  map[string]interface{} `json:"fullDocument,omitempty"`
	UpdatedFields map[string]interface{} `json:"updatedFields,omitempty"`
	Timestamp     time.Time              `json:"timestamp"`
}

// ChangeEventHandler is a function that handles change events
type ChangeEventHandler func(event ChangeEvent)

// ChangeStreamWatcher watches MongoDB collections for changes
type ChangeStreamWatcher struct {
	db     *MongoDBClient
	logger *slog.Logger
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup

	// Handlers for different collections
	handlers   map[string][]ChangeEventHandler
	handlersMu sync.RWMutex

	// Track active streams
	streams   map[string]*mongo.ChangeStream
	streamsMu sync.Mutex
}

// NewChangeStreamWatcher creates a new change stream watcher
func NewChangeStreamWatcher(db *MongoDBClient, logger *slog.Logger) *ChangeStreamWatcher {
	ctx, cancel := context.WithCancel(context.Background())

	return &ChangeStreamWatcher{
		db:       db,
		logger:   logger,
		ctx:      ctx,
		cancel:   cancel,
		handlers: make(map[string][]ChangeEventHandler),
		streams:  make(map[string]*mongo.ChangeStream),
	}
}

// RegisterHandler registers a handler for a specific collection
func (w *ChangeStreamWatcher) RegisterHandler(collection string, handler ChangeEventHandler) {
	w.handlersMu.Lock()
	defer w.handlersMu.Unlock()

	w.handlers[collection] = append(w.handlers[collection], handler)
	w.logger.Debug("Registered change stream handler", "collection", collection)
}

// Start begins watching all collections with registered handlers
func (w *ChangeStreamWatcher) Start() error {
	w.handlersMu.RLock()
	collections := make([]string, 0, len(w.handlers))
	for collection := range w.handlers {
		collections = append(collections, collection)
	}
	w.handlersMu.RUnlock()

	for _, collection := range collections {
		if err := w.watchCollection(collection); err != nil {
			w.logger.Error("Failed to start watching collection",
				"collection", collection,
				"error", err,
			)
			// Continue with other collections even if one fails
		}
	}

	w.logger.Info("Change stream watcher started",
		"collections", collections,
	)

	return nil
}

// watchCollection starts watching a specific collection
func (w *ChangeStreamWatcher) watchCollection(collectionName string) error {
	var collection *mongo.Collection

	switch collectionName {
	case "stations":
		collection = w.db.StationsCollection
	case "messages":
		collection = w.db.MessagesCollection
	case "transactions":
		collection = w.db.TransactionsCollection
	case "sessions":
		collection = w.db.SessionsCollection
	case "meter_values":
		collection = w.db.MeterValuesCollection
	default:
		collection = w.db.GetDatabase().Collection(collectionName)
	}

	// Configure change stream options
	opts := options.ChangeStream().
		SetFullDocument(options.UpdateLookup). // Get full document on updates
		SetMaxAwaitTime(5 * time.Second)       // Max time to wait for changes

	// Create pipeline to filter events
	pipeline := mongo.Pipeline{
		bson.D{{Key: "$match", Value: bson.M{
			"operationType": bson.M{
				"$in": []string{"insert", "update", "replace", "delete"},
			},
		}}},
	}

	// Open change stream
	stream, err := collection.Watch(w.ctx, pipeline, opts)
	if err != nil {
		return err
	}

	// Store stream reference
	w.streamsMu.Lock()
	w.streams[collectionName] = stream
	w.streamsMu.Unlock()

	// Start goroutine to process changes
	w.wg.Add(1)
	go w.processChanges(collectionName, stream)

	w.logger.Info("Started watching collection", "collection", collectionName)

	return nil
}

// processChanges processes changes from a change stream
func (w *ChangeStreamWatcher) processChanges(collectionName string, stream *mongo.ChangeStream) {
	defer w.wg.Done()
	defer stream.Close(context.Background())

	for {
		select {
		case <-w.ctx.Done():
			w.logger.Info("Stopping change stream", "collection", collectionName)
			return
		default:
			if stream.Next(w.ctx) {
				var changeDoc struct {
					OperationType string                 `bson:"operationType"`
					FullDocument  map[string]interface{} `bson:"fullDocument"`
					DocumentKey   struct {
						ID interface{} `bson:"_id"`
					} `bson:"documentKey"`
					UpdateDescription struct {
						UpdatedFields map[string]interface{} `bson:"updatedFields"`
						RemovedFields []string               `bson:"removedFields"`
					} `bson:"updateDescription"`
					ClusterTime interface{} `bson:"clusterTime"`
				}

				if err := stream.Decode(&changeDoc); err != nil {
					w.logger.Error("Failed to decode change event",
						"collection", collectionName,
						"error", err,
					)
					continue
				}

				// Create change event
				event := ChangeEvent{
					Type:          ChangeEventType(changeDoc.OperationType),
					Collection:    collectionName,
					FullDocument:  changeDoc.FullDocument,
					UpdatedFields: changeDoc.UpdateDescription.UpdatedFields,
					Timestamp:     time.Now(),
				}

				// Extract document ID
				if changeDoc.DocumentKey.ID != nil {
					if idStr, ok := changeDoc.DocumentKey.ID.(string); ok {
						event.DocumentID = idStr
					}
				}

				// Notify handlers
				w.notifyHandlers(collectionName, event)
			}

			// Check for errors
			if err := stream.Err(); err != nil {
				if w.ctx.Err() != nil {
					// Context cancelled, exit gracefully
					return
				}
				w.logger.Error("Change stream error",
					"collection", collectionName,
					"error", err,
				)
				// Try to reconnect after a delay
				time.Sleep(5 * time.Second)
				if err := w.watchCollection(collectionName); err != nil {
					w.logger.Error("Failed to reconnect change stream",
						"collection", collectionName,
						"error", err,
					)
				}
				return
			}
		}
	}
}

// notifyHandlers notifies all registered handlers for a collection
func (w *ChangeStreamWatcher) notifyHandlers(collection string, event ChangeEvent) {
	w.handlersMu.RLock()
	handlers := w.handlers[collection]
	w.handlersMu.RUnlock()

	for _, handler := range handlers {
		// Call handler in a goroutine to prevent blocking
		go func(h ChangeEventHandler) {
			defer func() {
				if r := recover(); r != nil {
					w.logger.Error("Handler panic recovered",
						"collection", collection,
						"panic", r,
					)
				}
			}()
			h(event)
		}(handler)
	}
}

// Stop stops watching all collections
func (w *ChangeStreamWatcher) Stop() error {
	w.logger.Info("Stopping change stream watcher")

	// Cancel context to signal all goroutines to stop
	w.cancel()

	// Close all streams
	w.streamsMu.Lock()
	for name, stream := range w.streams {
		if err := stream.Close(context.Background()); err != nil {
			w.logger.Error("Failed to close stream",
				"collection", name,
				"error", err,
			)
		}
	}
	w.streams = make(map[string]*mongo.ChangeStream)
	w.streamsMu.Unlock()

	// Wait for all goroutines to finish
	w.wg.Wait()

	w.logger.Info("Change stream watcher stopped")

	return nil
}

// WatchStations starts watching the stations collection for status changes
func (w *ChangeStreamWatcher) WatchStations(handler ChangeEventHandler) {
	w.RegisterHandler("stations", handler)
}

// WatchTransactions starts watching the transactions collection
func (w *ChangeStreamWatcher) WatchTransactions(handler ChangeEventHandler) {
	w.RegisterHandler("transactions", handler)
}

// WatchMessages starts watching the messages collection
func (w *ChangeStreamWatcher) WatchMessages(handler ChangeEventHandler) {
	w.RegisterHandler("messages", handler)
}
