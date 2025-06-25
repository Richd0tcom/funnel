package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/richd0tcom/funnel/internal/broker"
	"github.com/richd0tcom/funnel/internal/domain"
)


type Worker struct {
	store domain.DataStore
	consumer domain.DataConsumer
	workerCount int
	batchSize int

	// metrics     *Metrics
}

func NewWorker(store domain.DataStore, consumer domain.DataConsumer, workerCount, batchSize int) *Worker {
	return &Worker{
		store: store,
		consumer: consumer,
		workerCount: workerCount,
		batchSize: batchSize,
	}
} 

func (w *Worker) Start(c context.Context, mq broker.MessageQueue) error {
	var wg sync.WaitGroup

	err:= mq.Subscribe()
	if err != nil {
		return err
	}

	for i:=range w.workerCount {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			w.worker(c, workerID, mq)
		}(i)
	}

	wg.Wait()
	return nil
}
func (w *Worker) worker(ctx context.Context, workerID int, mq broker.MessageQueue) {
	log.Printf("Worker %d started", workerID)

	defer log.Printf("Worker %d stopped", workerID)

	batch:= make([]domain.SensorData, 0, w.batchSize)
	ticker:= time.NewTicker(5 * time.Second)

	defer ticker.Stop()


	handler := func(data []byte) error {
		var bulkData domain.BulkSensorData
		if err := json.Unmarshal(data, &bulkData); err != nil {
			
			return fmt.Errorf("failed to unmarshal data: %w", err)
		}

		batch = append(batch, bulkData.Data...)
		

		if len(batch) >= w.batchSize {
			w.processBatch(ctx, batch)
			batch = batch[:0] // Reset batch
		}


		return nil
	}

	go func() {
		if err:= mq.Consume(ctx, handler); err != nil {
			log.Printf("Worker %d subscription error: %v", workerID, err)
		}
	}()


	for {
		select {
		case <-ctx.Done():
			if len(batch) > 0 {
				w.processBatch(ctx, batch)
			}
			return
		case <-ticker.C:
			if len(batch) > 0 {
				w.processBatch(ctx, batch)
				batch = batch[:0]
			}
		}
	}

}
func (w *Worker) processBatch(ctx context.Context, batch []domain.SensorData) {
	start := time.Now()


	if err := w.store.InsertBatch(ctx, batch); err != nil {
		log.Printf("Failed to store batch: %v", err)
		return
	}

	if err := w.consumer.Process(batch); err != nil {
		log.Printf("Failed to process batch in consumer: %v", err)
		return
	}

	duration := time.Since(start)

		log.Printf("Processed batch of %d items in %v", len(batch), duration)

}