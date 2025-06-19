package domain

import "time"

type SensorData struct {
	SensorID    string                 `json:"sensor_id" bson:"sensor_id"`
	Timestamp   time.Time              `json:"timestamp" bson:"timestamp"`
	MetricType  string                 `json:"metric_type" bson:"metric_type"`
	Value       float64                `json:"value" bson:"value"`
	Tags        map[string]string      `json:"tags,omitempty" bson:"tags,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty" bson:"metadata,omitempty"`
}

type BulkSensorData struct {
	Data []SensorData `json:"data"`
}

type DataConsumer interface {
	Process(data []SensorData) error
}