package workers

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"github.com/coDing-Knight99/GoChat/models"
	"github.com/redis/go-redis/v9"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

type DBWorker struct {
	RDB       *redis.Client
	MongoColl *mongo.Collection
}

type pendingMessage struct {
	raw       string
	doc       models.Message
	persisted bool
}

func NewDBWorker(rdb *redis.Client, mongoColl *mongo.Collection) *DBWorker {
	return &DBWorker{
		RDB:       rdb,
		MongoColl: mongoColl,
	}
}

func (w *DBWorker) Start() {
	ctx := context.Background()
	log.Println("Starting DB Worker")
	const batchSize = 100
	const flushInterval = 4 * time.Second
	const queueName = "chat_msgs"
	const processingQueueName = "chat_msgs:processing"

	// Messages already claimed when the process stopped were never acknowledged.
	// Put them back before taking new work so they are retried instead of lost.
	w.requeueProcessing(ctx, queueName, processingQueueName)

	var buffer []pendingMessage
	ticker := time.NewTicker(flushInterval)
	poll := time.NewTicker(200 * time.Millisecond)
	defer ticker.Stop()
	defer poll.Stop()
	log.Println("DB Worker Started")
	for {
		select {
		case <-ticker.C:
			if len(buffer) > 0 {
				log.Println("Timeout Reached 10 seconds flushing buffer to DB")
				w.flush(ctx, processingQueueName, &buffer)
			}
		case <-poll.C:
			if len(buffer) >= batchSize {
				log.Println("Buffer limit reached flushing to DB", len(buffer))
				w.flush(ctx, processingQueueName, &buffer)
				continue
			}

			msgJSON, err := w.RDB.LMove(ctx, queueName, processingQueueName, "RIGHT", "LEFT").Result()
			if err == redis.Nil {
				continue
			}
			if err != nil {
				log.Println("Redis LMOVE error:", err)
				continue
			}

			var msg models.Message
			if err := json.Unmarshal([]byte(msgJSON), &msg); err != nil {
				// Invalid data cannot be retried successfully; move it aside for inspection.
				log.Println("Invalid message moved to dead-letter queue:", err)
				if err := w.RDB.LMove(ctx, processingQueueName, "chat_msgs:dead_letter", "LEFT", "LEFT").Err(); err != nil {
					log.Println("Failed to move invalid message to dead-letter queue:", err)
				}
				continue
			}
			buffer = append(buffer, pendingMessage{raw: msgJSON, doc: msg})
		}
	}
}

func (w *DBWorker) requeueProcessing(ctx context.Context, queueName, processingQueueName string) {
	for {
		_, err := w.RDB.LMove(ctx, processingQueueName, queueName, "RIGHT", "LEFT").Result()
		if err == redis.Nil {
			return
		}
		if err != nil {
			log.Println("Failed to recover processing queue:", err)
			return
		}
	}
}

func (w *DBWorker) flush(ctx context.Context, processingQueueName string, buffer *[]pendingMessage) {
	if len(*buffer) == 0 {
		return
	}

	if !(*buffer)[0].persisted {
		docs := make([]interface{}, len(*buffer))
		for i, message := range *buffer {
			docs[i] = message.doc
		}

		writeCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
		_, err := w.MongoColl.InsertMany(writeCtx, docs)
		cancel()
		if err != nil {
			// Keep the claimed messages in the processing queue and in memory for retry.
			log.Println("Failed to write batch to MongoDB; it will be retried:", err)
			return
		}
		for i := range *buffer {
			(*buffer)[i].persisted = true
		}
	}

	for len(*buffer) > 0 {
		message := (*buffer)[0]
		if err := w.RDB.LRem(ctx, processingQueueName, 1, message.raw).Err(); err != nil {
			// It is safer to retry a Mongo write than to discard a message that was not acknowledged.
			log.Println("MongoDB write succeeded but Redis acknowledgement failed:", err)
			return
		}
		*buffer = (*buffer)[1:]
	}
	log.Println("Successfully flushed logs to MongoDB")
}
