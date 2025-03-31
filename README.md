# Gored

Gored is a high-performance, in-memory key-value store designed for low-latency, high-throughput workloads. It follows the RESP (Redis Serialization Protocol) and provides a minimal yet efficient caching solution with a built-in LRU eviction strategy.

## Features

- Optimized for high request throughput with minimal latency
- Uses an in-memory storage engine with an LRU eviction policy
- Implements RESP for seamless Redis compatibility
- Multi-threaded architecture with sharded cache design
- Efficient memory management to fit within limited RAM constraints
- Deployable as a standalone binary or in a Docker container

## Architecture

### Cache Design

- **Sharded LRU Cache**: Cache is divided into multiple shards to reduce lock contention and improve concurrency.
- **Efficient Hashing**: Uses FNV-1a hashing for distributing keys across shards.
- **Optimized Data Structures**: Uses a combination of hash maps and doubly linked lists to achieve O(1) lookups and O(1) evictions.
- **Memory-Conscious**: Carefully managed memory to prevent unnecessary allocations and reduce garbage collection overhead.

### Performance Optimizations

- **Direct RESP Protocol**: Eliminates HTTP overhead by using the RESP protocol directly over TCP.
- **Connection Reuse**: Supports persistent client connections for better performance.
- **Read-Write Optimization**: Uses fine-grained locking to allow concurrent reads while limiting write contention.
- **Buffer Pooling**: Reduces memory allocations by reusing buffers.
- **Automatic CPU Scaling**: Uses all available CPU cores for better efficiency.

## Getting Started

### Running with Docker

To quickly run Gored using Docker, follow these steps:

```sh
# Pull the latest Gored image from Docker Hub
docker pull your-dockerhub-username/gored:latest

# Run the container
docker run -p 7171:7171 your-dockerhub-username/gored:latest
```

### Running Locally

If you prefer to run Gored without Docker, you can build and run it manually:

```sh
# Clone the repository
git clone https://github.com/your-repo/gored.git && cd gored

# Initialize Go modules
go mod init gored

# Download dependencies
go mod tidy

# Build the binary
go build -o gored

# Start the server
./gored
```

## Using the Key-Value Store

Gored communicates over TCP using the RESP protocol, making it compatible with Redis clients.

### Basic Commands

- `SET key value` - Stores a key-value pair
- `GET key` - Retrieves the value of a given key
- `PING` - Returns a PONG response to test connectivity
- `STATS` - Returns cache statistics

### Using Redis CLI

If you have Redis installed, you can use the Redis CLI to interact with Gored:

```sh
redis-cli -p 7171
SET mykey "Hello, World!"
GET mykey
```

### Using Raw RESP Commands

You can also send raw RESP commands manually:

```sh
# Send a SET command
printf "*3\r\n$3\r\nSET\r\n$5\r\nmykey\r\n$13\r\nHello, World!\r\n" | nc localhost 7171

# Send a GET command
printf "*2\r\n$3\r\nGET\r\n$5\r\nmykey\r\n" | nc localhost 7171
```

## Redis Benchmark Results

The following table summarizes the performance of the Redis-like cache under a high-concurrency workload using `redis-benchmark`.

| Operation | Requests | Time Taken (s) | Clients | Avg Latency (ms) | P50 Latency (ms) | P95 Latency (ms) | P99 Latency (ms) | Max Latency (ms) | Throughput (req/sec) |
|-----------|----------|---------------|---------|------------------|------------------|------------------|------------------|------------------|----------------------|
| **SET**   | 1,000,000 | 15.98         | 10,000  | 79.174           | 78.655           | 104.703          | 113.151          | 161.535          | 62,566.48           |
| **GET**   | 1,000,000 | 15.94         | 10,000  | 79.167           | 79.167           | 100.159          | 112.639          | 130.239          | 62,801.51           |

### Notes:
- Benchmark was executed using `redis-benchmark -t SET,GET -n 1000000 -c 10000`.
- Latency metrics represent the time taken per request.
- Throughput is the number of requests served per second.


## Load Testing

You can use Locust to benchmark Gored's performance.

```sh
pip install locust
locust -f locustfile.py --host=tcp://localhost:7171
```

Then open `http://localhost:8089` in a web browser to start the test.

## Performance Results

Benchmarking on an AWS `t3.small` instance (2 vCPUs, 2GB RAM) shows:

- Over 11,000 requests per second
- Average latency under 5ms
- 0% cache miss rate for actively used keys
- Memory usage remains within safe limits under high load

## Error Handling

- If a key does not exist, the server returns a null bulk string (`$-1\r\n`).
- If the cache exceeds its memory limit, the LRU eviction policy removes the least recently used keys.
- Errors are returned in RESP format and follow Redis-like conventions.
- The server handles client disconnections and network errors gracefully.

## Contributing

If you'd like to contribute to Gored, feel free to submit pull requests or report issues on the GitHub repository.

## License

Gored is open-source and released under the MIT License.

