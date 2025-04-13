# Raft3D

Raft3D is a distributed 3D printer management system with data persistence implemented via the Raft Consensus Algorithm rather than traditional centralized databases.

## Features

- Distributed 3D printer management with Raft consensus
- REST API for managing printers, filaments, and print jobs
- Fault tolerance with leader election and data replication
- Support for snapshotting and log compaction

## Requirements

- Go 1.21 or higher
- Internet connection to download dependencies

## Installation

1. Clone the repository:

```bash
git clone https://github.com/yourusername/raft3d.git
cd raft3d
```

2. Build the application:

```bash
go build -o raft3d ./cmd/server
```

## Running the Cluster

To demonstrate Raft3D, you need to run at least 3 nodes. Here's how to start a 3-node cluster:

### Node 1 (Bootstrap node)

```bash
mkdir -p data/node1
./raft3d -id node1 -raft-addr localhost:7000 -raft-dir data/node1 -http-addr localhost:8000 -bootstrap -peers localhost:7000,localhost:7001,localhost:7002
```

### Node 2

```bash
mkdir -p data/node2
./raft3d -id node2 -raft-addr localhost:7001 -raft-dir data/node2 -http-addr localhost:8001 -peers localhost:7000,localhost:7001,localhost:7002
```

### Node 3

```bash
mkdir -p data/node3
./raft3d -id node3 -raft-addr localhost:7002 -raft-dir data/node3 -http-addr localhost:8002 -peers localhost:7000,localhost:7001,localhost:7002
```

## Testing the API

You can use curl or a tool like Postman to test the API endpoints. Here are some examples:

### Create a Printer

```bash
curl -X POST http://localhost:8000/api/v1/printers -H "Content-Type: application/json" -d '{"company": "Creality", "model": "Ender 3"}'
```

### List Printers

```bash
curl -X GET http://localhost:8000/api/v1/printers
```

### Create a Filament

```bash
curl -X POST http://localhost:8000/api/v1/filaments -H "Content-Type: application/json" -d '{"type": "PLA", "color": "red", "total_weight_in_grams": 1000, "remaining_weight_in_grams": 1000}'
```

### List Filaments

```bash
curl -X GET http://localhost:8000/api/v1/filaments
```

### Create a Print Job

```bash
curl -X POST http://localhost:8000/api/v1/print_jobs -H "Content-Type: application/json" -d '{"printer_id": "PRINTER_ID", "filament_id": "FILAMENT_ID", "filepath": "prints/model.gcode", "print_weight_in_grams": 100}'
```

Replace `PRINTER_ID` and `FILAMENT_ID` with the actual IDs returned from the create operations.

### List Print Jobs

```bash
curl -X GET http://localhost:8000/api/v1/print_jobs
```

### Update Print Job Status

```bash
curl -X POST "http://localhost:8000/api/v1/print_jobs/JOB_ID/status?status=Running"
```

Replace `JOB_ID` with the actual ID of the print job.

## Testing Raft Functionality

To test the Raft consensus functionality, you can:

1. Send a request to create a printer to the leader node
2. Verify that the printer exists by querying any node
3. Kill the leader node (Ctrl+C on the leader's terminal)
4. Observe that a new leader is elected
5. Create a new printer by sending a request to the new leader
6. Restart the old leader node
7. Verify that the old leader node has received all the data by querying it

## Checking Node Status

You can check the status of any node by sending a GET request to the `/status` endpoint:

```bash
curl -X GET http://localhost:8000/status
```

This will return information about the node, including whether it's the leader and the current state of the Raft consensus.

## Architecture

Raft3D is built using the HashiCorp Raft library and follows the Raft consensus algorithm. The application consists of:

1. **Raft Node**: Responsible for maintaining the distributed state and consensus
2. **Finite State Machine (FSM)**: Handles the application state (printers, filaments, print jobs)
3. **HTTP API**: Provides RESTful endpoints for interacting with the system

## License

This project is licensed under the MIT License.