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

func 	NewDBWorker(rdb *redis.Client, mongoColl *mongo.Collection) *DBWorker {
	return &DBWorker{
		RDB:       rdb,
		MongoColl: mongoColl,
	}
}

func (w *DBWorker) Start() {
	ctx:=context.Background()
	log.Println("Starting DB Worker")
	const Batchsize = 100
	const flushInterval = 10 * time.Second
	var buffer []interface{}
	ticker := time.NewTicker(flushInterval)
	defer ticker.Stop()
	log.Println("DB Worker Started")
	for {
		select {
		case <-ticker.C:
			if len(buffer) > 0 {
				log.Println("Timeout Reached 10 seconds flushing buffer to DB")
				w.flush(ctx, &buffer)
			}
		default:
			results, err := w.RDB.LPopCount(ctx, "chat_msgs", Batchsize-len(buffer)).Result()
			if err == redis.Nil || len(results) == 0 {
				time.Sleep(200 * time.Millisecond)
				continue
			} else if err != nil {
				log.Println("Redis LPOP Error: ", err)
				time.Sleep(2 * time.Second)
				continue
			}
			for _, msgJSON := range results {
				var msg models.Message
				err := json.Unmarshal([]byte(msgJSON), &msg)
				if err != nil {
					log.Println("Failed to unmarshal message in worker ", err)
					continue
				}
				buffer = append(buffer, msg)
			}
			if len(buffer) >= Batchsize {
				log.Println("Buffer limit reached flushing to DB", len(buffer))
				w.flush(ctx, &buffer)
				ticker.Reset(flushInterval)
			}
		}
	}
}

func (w *DBWorker) flush(ctx context.Context, buffer *[]interface{}) {
	if len(*buffer) == 0 {
		return
	}
	_, err := w.MongoColl.InsertMany(ctx, *buffer)
	if err != nil {
		log.Println("Failed to bulk write to MongoDB ", err)
	} else {
		log.Println("Successfully flushed logs to MongoDB ", len(*buffer))
	}
	*buffer = (*buffer)[:0]
}
