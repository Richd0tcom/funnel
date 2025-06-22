# Funnel - High Volume Data Aggregator (PoC)

Funnel is a proof-of-concept metric ingestion service designed to efficiently handle and aggregate large volumes of time series data. It leverages modern, scalable technologies to provide reliable ingestion, storage, and real-time aggregation of sensor metrics.

---

## Table of Contents

- [Funnel - High Volume Data Aggregator (PoC)](#funnel---high-volume-data-aggregator-poc)
  - [Table of Contents](#table-of-contents)
  - [Overview](#overview)
  - [Architecture](#architecture)
  - [Features](#features)
  - [Tech Stack](#tech-stack)
  - [Setup \& Usage](#setup--usage)
    - [Prerequisites](#prerequisites)
    - [Steps](#steps)
    - [Project Structure](#project-structure)
    - [Testing \& Load Generation](#testing--load-generation)

---

## Overview

Funnel simulates a high-throughput metric ingestion pipeline. Sensor data is published to Kafka, batched, and stored in a MongoDB time series collection. Aggregation queries can be run on demand to summarize data over configurable time windows.

---

## Architecture

![aggregator](https://github.com/user-attachments/assets/62f0e228-411c-4085-94b6-57b5686a4c4b)

<!--
[Architecture Diagram Placeholder]
Insert architecture diagram here.
-->

**Description:**
- **Producers** send sensor data to **Kafka**.
- **Go-based consumers** batch and flush data to **MongoDB** every few seconds.
- **MongoDB** stores data in a time series collection, optimized for fast inserts and aggregations.
- **REST API** endpoints allow for data ingestion and aggregate queries.
- **Load testing tool** simulates high-volume data ingestion and validates aggregation performance.

---

## Features

- High-throughput ingestion using Kafka
- Efficient batch processing and storage in MongoDB time series collections
- Flexible aggregation queries (minute, hour, day granularity)
- Scalable, concurrent Go workers
- Load testing utility for benchmarking
- Easy deployment with Docker Compose

---

## Tech Stack

- **Golang**: Concurrency, batch processing, REST API
- **Kafka**: Message queue for decoupled, scalable ingestion
- **MongoDB**: Time series storage and aggregation
- **Docker Compose**: Simplified multi-service orchestration
- **Prometheus**: (Coming soon) Real-time metrics and monitoring

---

## Setup & Usage

### Prerequisites

- [Docker](https://www.docker.com/) & [Docker Compose](https://docs.docker.com/compose/)

### Steps

1. **Clone the repository**
   ```sh
   git clone https://github.com/yourusername/funnel.git
   cd funnel
   ```

2. **Start core services**
   ```sh
   docker compose up -d
   ```
3. **Start the main application (with Kafka profile)**
    ```sh
    docker compose --profile kafka up -d
    ```

4. **Start the load testing tool (optional)**
    ```sh
    docker compose --profile loadtest up -d
    ```

### Project Structure

```ascii
.
├── internal/
│   ├── broker/         # Kafka integration
│   ├── db/             # MongoDB storage and aggregation logic
│   ├── domain/         # Core domain models and interfaces
│   └── worker/         # Batch processing workers
├── load-testing/       # Load test tool (Go)
├── scripts/            # MongoDB initialization scripts
├── Dockerfile          # Main app Dockerfile
├──   # Multi-service orchestration
└── 
```

### Testing & Load Generation 
- The load-testing service simulates concurrent users sending sensor data and running aggregate queries.
- Configuration (users, duration, etc.) can be set via environment variables in docker-compose.yml.
