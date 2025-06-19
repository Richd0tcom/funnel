package domain

import (
	"context"
	"time"
)

type DataStore interface {
	InsertBatch(ctx context.Context, data []SensorData) error
	GetAggregates(ctx context.Context, query AggregateQuery) ([]AggregateResult, error)
	Close() error
}


type AggregateQuery struct {
	SensorID   string    `json:"sensor_id,omitempty"`
	MetricType string    `json:"metric_type,omitempty"`
	StartTime  time.Time `json:"start_time"`
	EndTime    time.Time `json:"end_time"`
	Granularity string   `json:"granularity"` // minute, hour, day
}

type AggregateResult struct {
	SensorID    string            `json:"sensor_id" bson:"_id"`
	MetricType  string            `json:"metric_type" bson:"metric_type"`
	TimeWindow  time.Time         `json:"time_window" bson:"time_window"`
	Count       int64             `json:"count" bson:"count"`
	Sum         float64           `json:"sum" bson:"sum"`
	Avg         float64           `json:"avg" bson:"avg"`
	Min         float64           `json:"min" bson:"min"`
	Max         float64           `json:"max" bson:"max"`
	Tags        map[string]string `json:"tags,omitempty" bson:"tags,omitempty"`
}