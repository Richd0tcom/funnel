package consumer

import (
	"log"
	"time"

	"github.com/richd0tcom/funnel/internal/domain"
)

type MockDataConsumer struct {
	name string
}

func NewMockDataConsumer(name string) *MockDataConsumer {
	return &MockDataConsumer{name: name}
}

func (m *MockDataConsumer) Process(data []domain.SensorData) error {
	log.Printf("[%s] Processing batch of %d sensor data points", m.name, len(data))
	for _, d := range data {
		log.Printf("[%s] Sensor: %s, Type: %s, Value: %.2f, Time: %s", 
			m.name, d.SensorID, d.MetricType, d.Value, d.Timestamp.Format(time.RFC3339))
	}
	return nil
}