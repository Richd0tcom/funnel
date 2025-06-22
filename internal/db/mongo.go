package db

import (
	"context"
	"fmt"
	"time"

	"github.com/richd0tcom/funnel/internal/domain"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"go.mongodb.org/mongo-driver/v2/mongo/readpref"
)

// TODO: decouple storage and storage implemtation
type MongoTimeSeriesStore struct {
	client     *mongo.Client
	db         *mongo.Database
	collection *mongo.Collection
}


func NewMongoConnection(uri string) (*mongo.Client, error){
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := mongo.Connect(options.Client().ApplyURI(uri))
	if err != nil {
		return nil, fmt.Errorf("failed to connect to MongoDB: %w", err)
	}

	// defer func() {
	// 	if err = client.Disconnect(ctx); err != nil {
	// 		panic(err)
	// 	}
	// }()

	ctx, cancel = context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	_ = client.Ping(ctx, readpref.Primary())
	
	return client , err
}

func NewMongoTimeSeriesStore(client *mongo.Client, database string) (*MongoTimeSeriesStore, error) {

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	
	db := client.Database(database)
	
	
	tsOptions := options.CreateCollection().SetTimeSeriesOptions(
		options.TimeSeries().
			SetTimeField("timestamp").
			SetMetaField("sensor_id").
			SetGranularity("seconds"),
	)

	
	db.CreateCollection(ctx, "sensor_metrics", tsOptions)
	collection := db.Collection("sensor_metrics")

	
	indexModels := []mongo.IndexModel{
		{
			Keys: bson.D{
				{Key: "sensor_id", Value: 1},
				{Key: "timestamp", Value: 1},
			},
		},
		{
			Keys: bson.D{
				{Key: "metric_type", Value: 1},
				{Key: "timestamp", Value: 1},
			},
		},
	}
	collection.Indexes().CreateMany(ctx, indexModels)

	return &MongoTimeSeriesStore{
		client:     client,
		db:         db,
		collection: collection,
	}, nil
}

func (m *MongoTimeSeriesStore) InsertBatch(ctx context.Context, data []domain.SensorData) error {//use to insert sensor data
	// data:= dat.([]any) //omo check here for possible error
	
	if len(data) == 0 {
		return nil
	}

	docs := make([]interface{}, len(data))
	for i, d := range data {
		docs[i] = d
	}

	opts := options.InsertMany().SetOrdered(false) 
	_, err := m.collection.InsertMany(ctx, docs, opts)
	return err
}

func (m *MongoTimeSeriesStore) GetAggregates(ctx context.Context, query domain.AggregateQuery) ([]domain.AggregateResult, error) {
	pipeline := m.buildAggregationPipeline(query)
	
	cursor, err := m.collection.Aggregate(ctx, pipeline)
	if err != nil {
		fmt.Println("---------AGGREGATE ERROR!!!!", err)
		return nil, err
	}
	defer cursor.Close(ctx)

	var results []domain.AggregateResult
	if err := cursor.All(ctx, &results); err != nil {
		fmt.Println("---------AGGREGATE PARSE ERROR!!!!", err)


		return nil, err
	}

	return results, nil
}

func (m *MongoTimeSeriesStore) buildAggregationPipeline(query domain.AggregateQuery) []bson.M {
	
	matchStage := bson.M{
		"timestamp": bson.M{
			"$gte": query.StartTime,
			"$lte": query.EndTime,
		},
	}

	if query.SensorID != "" {
		matchStage["sensor_id"] = query.SensorID
	}
	if query.MetricType != "" {
		matchStage["metric_type"] = query.MetricType
	}

	
	var dateFormat string
	switch query.Granularity {
	case "minute":
		dateFormat = "%Y-%m-%d-%H-%M"
	case "hour":
		dateFormat = "%Y-%m-%d-%H"
	case "day":
		dateFormat = "%Y-%m-%d"
	default:
		dateFormat = "%Y-%m-%d-%H"
	}

	groupStage := bson.M{
		"_id": bson.M{
			"sensor_id":   "$sensor_id",
			"metric_type": "$metric_type",
			"time_window": bson.M{
				"$dateToString": bson.M{
					"format": dateFormat,
					"date":   "$timestamp",
					"timezone":   "UTC",
				},
			},
		},
		"count": bson.M{"$sum": 1},
		"sum":   bson.M{"$sum": "$value"},
		"avg":   bson.M{"$avg": "$value"},
		"min":   bson.M{"$min": "$value"},
		"max":   bson.M{"$max": "$value"},
		"tags":  bson.M{"$first": "$tags"},
	}

	
	projectStage := bson.M{
		"sensor_id":   "$_id.sensor_id",
		"metric_type": "$_id.metric_type",
		"time_window": bson.M{
			"$dateFromString": bson.M{
				"dateString": "$_id.time_window",
				"format": dateFormat,
				"timezone":   "UTC",
			},
		},
		"count": 1,
		"sum":   1,
		"avg":   1,
		"min":   1,
		"max":   1,
		"tags":  1,
	}

	return []bson.M{
		{"$match": matchStage},
		{"$group": groupStage},
		{"$project": projectStage},
		{"$sort": bson.M{"time_window": 1}},
	}
}

func (m *MongoTimeSeriesStore) Close() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return m.client.Disconnect(ctx)
}