# GoQueue

**A distributed background job processing system built with Go and PostgreSQL.**

GoQueue is a backend job queue designed to reliably execute asynchronous tasks using concurrent workers. It uses PostgreSQL for persistent job storage and atomic job claiming, allowing multiple workers to process jobs concurrently without accidentally executing the same job at the same time.

The project explores the engineering challenges behind building a reliable job processing system, including concurrency control, worker coordination, job ownership, lease management, automatic retries, and graceful shutdown.

> Built to understand and implement the core engineering principles behind production-grade background job processing systems.

---

## Overview

Background processing is an essential part of modern backend systems. Tasks such as sending emails, processing payments, generating reports, and handling notifications should not block the main request-response cycle.

GoQueue provides a mechanism to enqueue these tasks and process them asynchronously through independent workers.

Instead of relying on an in-memory queue, GoQueue uses PostgreSQL as its persistent job store. This allows jobs to survive application restarts and enables multiple worker instances to coordinate through the database.

### Key Features

- **Persistent Job Storage:** Stores jobs in PostgreSQL, ensuring jobs survive application restarts.
- **Concurrent Workers:** Uses Go goroutines to process multiple jobs concurrently.
- **Atomic Job Claiming:** Uses PostgreSQL transactions and `FOR UPDATE SKIP LOCKED` to prevent multiple workers from claiming the same job.
- **Job Ownership:** Uses unique claim tokens to ensure only the worker that owns a job can complete or fail it.
- **Lease-Based Processing:** Assigns expiration-based leases to claimed jobs so abandoned jobs can eventually become available again.
- **Heartbeat Mechanism:** Allows workers to renew their leases while processing long-running jobs.
- **Automatic Retries:** Retries failed jobs using exponential backoff.
- **Maximum Retry Attempts:** Moves jobs into a terminal failed state after exhausting their retry attempts.
- **Job Handler Registry:** Maps job types to their respective handlers, making the system extensible.
- **Graceful Shutdown:** Stops workers cleanly and coordinates ongoing processing during application shutdown.
- **Database Integration Testing:** Tests repository behavior against real PostgreSQL instances using Testcontainers.

---

## Architecture

GoQueue follows a layered architecture that separates HTTP handling, business logic, database operations, and background processing.

```text
                 ┌─────────────────────┐
                 │      Client         │
                 └──────────┬──────────┘
                            │
                            ▼
                 ┌─────────────────────┐
                 │     HTTP API        │
                 │  Handlers / Routes  │
                 └──────────┬──────────┘
                            │
                            ▼
                 ┌─────────────────────┐
                 │      Service        │
                 │   Business Logic    │
                 └──────────┬──────────┘
                            │
                            ▼
                 ┌─────────────────────┐
                 │    Repository       │
                 │  Database Queries   │
                 └──────────┬──────────┘
                            │
                            ▼
                 ┌─────────────────────┐
                 │     PostgreSQL      │
                 │    Persistent Jobs  │
                 └──────────┬──────────┘
                            │
                            ▼
                 ┌─────────────────────┐
                 │    Worker Pool      │
                 │  Concurrent Workers │
                 └──────────┬──────────┘
                            │
                            ▼
                 ┌─────────────────────┐
                 │   Handler Registry  │
                 │  Job Type → Handler │
                 └─────────────────────┘
```

### Job Processing Lifecycle

Every job moves through a defined lifecycle.

```text
          ┌─────────┐
          │ PENDING │
          └────┬────┘
               │
               ▼
         ┌────────────┐
         │ PROCESSING │
         └─────┬──────┘
               │
         ┌─────┴──────┐
         │            │
         ▼            ▼
    ┌──────────┐  ┌──────────┐
    │COMPLETED │  │  FAILED  │
    └──────────┘  └────┬─────┘
                       │
                 Retry Available?
                   /       \
                 Yes        No
                  │          │
                  ▼          ▼
              PENDING    TERMINAL
                         FAILURE
```

A failed job can be scheduled for another attempt based on its retry policy. Jobs that exhaust their maximum attempts are moved into a terminal failure state.

---

## Core Engineering Concepts

One of the primary goals of GoQueue is to implement the mechanisms that make background job processing reliable under concurrent execution.

### 1. Atomic Job Claiming

A major challenge in a concurrent job queue is preventing multiple workers from processing the same job.

Imagine two workers attempting to fetch the next available job simultaneously. Without proper coordination, both workers could select the same pending job.

GoQueue addresses this using PostgreSQL transactions and:

`SELECT ... FOR UPDATE SKIP LOCKED`

When a worker claims a job, the database locks the selected row within a transaction. Other workers skip that locked row and can claim different jobs.

This provides safe coordination between concurrent workers without requiring a separate distributed locking service.

### 2. Claim Tokens and Job Ownership

Claiming a job is only part of the problem. The system must also ensure that only the worker that currently owns a job can modify its processing state.

GoQueue assigns a unique claim token whenever a job is claimed.

The token is checked when a worker attempts to complete or fail a job.

Database updates are guarded by conditions such as:

- The job must currently be in the `processing` state.
- The supplied claim token must match the job's current claim token.

This prevents stale workers from modifying jobs that have already been reclaimed by another worker.

### 3. Lease-Based Job Processing

Workers may crash, become unresponsive, or lose connectivity while processing a job.

Without a recovery mechanism, such jobs could remain stuck in the processing state indefinitely.

GoQueue uses time-based leases to address this problem.

- Each claimed job receives a lease expiration time.
- Workers periodically renew their leases through heartbeats.
- If a worker stops renewing its lease, the lease eventually expires.
- Expired jobs can be recovered and made available for processing again.

This provides a recovery mechanism for abandoned jobs.

### 4. Automatic Retries and Exponential Backoff

Temporary failures are common in distributed systems. An external API might become unavailable, a network request might time out, or a dependent service might temporarily fail.

Immediately retrying a failed job can create unnecessary load.

GoQueue implements exponential backoff to gradually increase the delay between retry attempts.

The general formula is:

`delay = base_delay × 2^(attempt - 1)`

For example, with a base delay of 5 seconds:

| Attempt | Retry Delay |
| ------- | ----------: |
| 1       |   5 seconds |
| 2       |  10 seconds |
| 3       |  20 seconds |
| 4       |  40 seconds |
| 5       |  80 seconds |

Once the maximum number of attempts is reached, the job is moved to a terminal failure state.

### 5. Handler Registry

GoQueue separates the job execution mechanism from the actual business logic.

A handler registry maps job types to their corresponding execution functions.

For example:

```go
registry.Register("email.send", emailHandler)
```

When a worker claims a job, it looks up the appropriate handler based on the job type and executes it.

This makes the worker system extensible without coupling the queue implementation to individual business operations.

---

## Technology Stack

| Technology     | Purpose                                    |
| -------------- | ------------------------------------------ |
| Go             | Core application and worker implementation |
| PostgreSQL     | Persistent job storage                     |
| pgx            | PostgreSQL driver                          |
| database/sql   | Database connection management             |
| chi            | HTTP routing                               |
| Docker         | Database containerization                  |
| Testcontainers | PostgreSQL integration testing             |
| golang-migrate | Database schema migrations                 |

---

## Project Structure

The project follows a layered structure to keep responsibilities separated.

```text
goqueue/
├── cmd/
│   └── ...
├── internal/
│   ├── jobs/
│   │   ├── handler/
│   │   ├── repository/
│   │   └── service/
│   ├── worker/
│   └── ...
├── migrations/
├── tests/
├── docker-compose.yml
├── Makefile
├── go.mod
└── README.md
```

_Note: Update this tree to match the final repository structure._

---

## Getting Started

### Prerequisites

Make sure you have the following installed:

- Go
- Docker and Docker Compose
- PostgreSQL (optional if using Docker)
- Git

### 1. Clone the repository

```bash
git clone https://github.com/prabej7/goqueue.git

cd goqueue
```

### 2. Start PostgreSQL

GoQueue uses PostgreSQL for persistent job storage.

Start the database using Docker Compose:

```bash
docker compose up -d
```

The development database is configured to expose PostgreSQL on port `5433` to avoid conflicts with an existing local PostgreSQL instance.

### 3. Run database migrations

GoQueue uses `golang-migrate` to manage database schema changes.

Configure your database connection and execute the migration commands defined in the Makefile.

### 4. Run the application

Configure the database connection using the application's configuration and start the Go application from its server entry point.

---

## Testing

GoQueue uses different testing strategies to validate application behavior.

### Unit Tests

Service-layer tests use mock repositories to verify business logic independently of the database.

### Integration Tests

Repository integration tests use Testcontainers to spin up real PostgreSQL instances.

This allows database behavior to be tested against an actual PostgreSQL engine rather than relying entirely on mocked database interactions.

The tests cover scenarios including:

- Creating and retrieving jobs.
- Handling jobs that do not exist.
- Handling duplicate job identifiers.
- Job state transitions.
- Invalid state transitions.
- Concurrent job claiming.
- Claim ownership validation.
- Lease renewal and expiration.
- Retry scheduling and maximum attempts.
- Worker execution and graceful shutdown.

Run the test suite:

```bash
go test ./...
```
or,
```bash
make test
```

---

## Design Goals

GoQueue is being developed around a few important engineering principles:

- **Reliability:** Jobs should not silently disappear because an application instance restarts.
- **Concurrency Safety:** Multiple workers should be able to process jobs without conflicting claims.
- **Fault Recovery:** Failed workers and expired leases should not leave jobs permanently stuck.
- **Extensibility:** New job types should be registered without changing the underlying worker implementation.
- **Separation of Concerns:** HTTP handling, business logic, persistence, and job execution should remain independent.
- **Testability:** Critical behavior should be verified through unit and integration tests.

---

## Future Improvements

Potential areas for further development include:

- Job listing, filtering, and pagination APIs.
- Job cancellation and scheduled execution APIs.
- Dead-letter queue and failed-job inspection.
- Operational metrics and monitoring.
- Worker and queue performance benchmarks.
- A dashboard for monitoring job execution.
- Improved observability through structured logging and tracing.

---

## Learning Objectives

GoQueue is also an exploration of several important backend engineering concepts:

- Concurrent programming with Go goroutines and contexts.
- PostgreSQL transaction management and row-level locking.
- Distributed coordination through database primitives.
- Lease-based ownership and heartbeat mechanisms.
- Retry strategies and exponential backoff.
- Graceful shutdown and resource lifecycle management.
- Unit testing, integration testing, and Testcontainers.
- Designing maintainable backend systems.

---

## Author

**Mahammad Prabej Hussain**

Backend Engineer | Go | TypeScript | PostgreSQL

- GitHub: [@prabej7](https://github.com/prabej7)
- Portfolio: [prabejhussain.work](https://www.mdprabejhussain.com.np)

---

_GoQueue is a learning-driven engineering project focused on understanding and implementing the core mechanisms behind reliable background job processing systems._
