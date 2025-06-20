# Funnel - High Volume Data Aggregator (PoC)

This project simulates a metric injestion service that handles large amount of time series data points per day.



## Tools
- Golang
- Kafka 
- Zookeeper
- MongoDB
- Prometheus for logging realtime (comming soon)

## Architecture
- Golang was chosen for concurrncy and ease of development
- Events are published to a Kafka topic which which is consumed by other services for batch processing and real time logging. Events a
  are batched and flushed every 5 seconds (can be changed) to a a mongodb time series collection.
  The batch events are then aggregated on demand
  



## Setup Requirements

- Docker / Docker compose
  
## Setup

1. clone the repository
2. run `docker compose up -d` to start services
3. run `docker-compose --profile kafka up -d` to start the main application
4. run `docker-compose --profile loadtest up -d` to start up a dummy test service to start producing data
