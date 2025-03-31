package main

import (
	"container/list"
	"fmt"
	"strings"
	"sync"
)

// LRUCache represents our cache with a doubly linked list for recency tracking
// and a map for O(1) lookups. This helps us maintain both speed and memory efficiency
// it is similar to a linkedhashmap in java
type LRUCache struct {
	capacity   int                      // max number of items before eviction
	items      map[string]*list.Element // for O(1) lookups
	evictionQ  *list.List               // tracks usage order for eviction
	mutex      sync.RWMutex             // protects concurrent access
	hitCount   int                      // tracks cache hits for stats
	missCount  int                      // tracks cache misses for stats
	totalGets  int                      // tracks total get operations
	totalPuts  int                      // tracks total put operations
	evictions  int                      // tracks how many items we've kicked out
	shardCount int                      // number of shards for the cache
	shards     []*cacheShard            // array of shards
	shardMask  uint32                   // bitmask used for shard selection
}

// cacheShard represents a portion of the cache to reduce lock contention. lock contention can cause performance issues
// when multiple goroutines are trying to access the same data structure at the same time
type cacheShard struct {
	items     map[string]*list.Element
	evictionQ *list.List
	mutex     sync.RWMutex
}

// cacheEntry represents a key-value pair in our cache
type cacheEntry struct {
	key   string
	value string
}

// NewLRUCache creates a new cache with the given capacity
// We're using 256 shards by default which is a good balance for most workloads
func NewLRUCache(capacity int) *LRUCache {
	// Default to 256 shards - power of 2 for efficient modulo with bitwise AND
	shardCount := 256
	cache := &LRUCache{
		capacity:   capacity,
		shardCount: shardCount,
		shardMask:  uint32(shardCount - 1),
		shards:     make([]*cacheShard, shardCount),
	}

	// initialize each shard
	for i := 0; i < shardCount; i++ {
		cache.shards[i] = &cacheShard{
			items:     make(map[string]*list.Element),
			evictionQ: list.New(),
			mutex:     sync.RWMutex{},
		}
	}

	return cache
}

// getShard returns the appropriate shard for a given key
// We use a simple hash function to distribute keys evenly
func (c *LRUCache) getShard(key string) *cacheShard {
	// FNV-1a hash for good distribution
	h := uint32(2166136261)
	for i := 0; i < len(key); i++ {
		h ^= uint32(key[i])
		h *= 16777619
	}
	// Use bitmask for efficient modulo with power of 2
	return c.shards[h&c.shardMask]
}

// put adds a key-value pair to the cache
func (c *LRUCache) Put(key, value string) {
	// count this operation
	c.mutex.Lock()
	c.totalPuts++
	c.mutex.Unlock()

	shard := c.getShard(key)
	shard.mutex.Lock()
	defer shard.mutex.Unlock()

	// check if the key exists
	if _, ok := shard.items[key]; ok {
		// update existing entry
		shard.evictionQ.MoveToFront(shard.items[key])
		shard.items[key].Value.(*cacheEntry).value = value
		return
	}

	// adding new entry
	entry := &cacheEntry{key: key, value: value}
	elem := shard.evictionQ.PushFront(entry)
	shard.items[key] = elem
	// checking if we need to evict
	if shard.evictionQ.Len() > c.capacity/c.shardCount {
		c.evictFromShard(shard)
	}
}

// evictFromShard removes the least recently used item from a shard
func (c *LRUCache) evictFromShard(shard *cacheShard) {
	// get the oldest element
	elem := shard.evictionQ.Back()
	if elem != nil {
		// removing from list and map
		shard.evictionQ.Remove(elem)
		entry := elem.Value.(*cacheEntry)
		delete(shard.items, entry.key)

		// Update eviction stats
		c.mutex.Lock()
		c.evictions++
		c.mutex.Unlock()
	}
}

// get retrieves a value for the given key
func (c *LRUCache) Get(key string) (string, bool) {
	// count this operation
	c.mutex.Lock()
	c.totalGets++
	c.mutex.Unlock()

	shard := c.getShard(key)
	shard.mutex.RLock()
	elem, ok := shard.items[key]

	// If key doesn't exist, return not found
	if !ok {
		shard.mutex.RUnlock()
		c.mutex.Lock()
		c.missCount++
		c.mutex.Unlock()
		return "", false
	}

	// Get value before upgrading lock
	value := elem.Value.(*cacheEntry).value
	shard.mutex.RUnlock()

	// Move to front - requires write lock
	shard.mutex.Lock()
	shard.evictionQ.MoveToFront(elem)
	shard.mutex.Unlock()

	// Update hit stats
	c.mutex.Lock()
	c.hitCount++
	c.mutex.Unlock()

	return value, true
}

// stats returns cache statistics
func (c *LRUCache) Stats() map[string]interface{} {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	totalItems := 0
	for _, shard := range c.shards {
		shard.mutex.RLock()
		totalItems += len(shard.items)
		shard.mutex.RUnlock()
	}

	hitRate := 0.0
	if c.totalGets > 0 {
		hitRate = float64(c.hitCount) / float64(c.totalGets) * 100.0
	}

	return map[string]interface{}{
		"capacity":    c.capacity,
		"size":        totalItems,
		"get_ops":     c.totalGets,
		"put_ops":     c.totalPuts,
		"hits":        c.hitCount,
		"misses":      c.missCount,
		"hit_rate":    hitRate,
		"evictions":   c.evictions,
		"shard_count": c.shardCount,
	}
}

// We use a single global cache instance with a large capacity
// The capacity is set to handle roughly 1 million entries with 256 chars each
// which should fit within the 2GB RAM constraint while leaving room for the application
var cache = NewLRUCache(1000000)

// processCommand handles incoming RESP commands
func processCommand(value Value) Value {
	if value.typ != "array" {
		return Value{typ: "error", str: "ERR invalid command format"}
	}

	// we need to make sure if the command is present or not
	if len(value.array) == 0 {
		return Value{typ: "error", str: "ERR empty command"}
	}

	// here's the point where we extract the command form the array
	cmdValue := value.array[0]

	// making sure the command is of the right type
	if cmdValue.typ == "" || (cmdValue.typ != "bulk" && cmdValue.typ != "string") {
		return Value{typ: "error", str: "ERR invalid command type"}
	}

	// converting the command to uppercase
	var cmd string
	if cmdValue.typ == "bulk" {
		cmd = strings.ToUpper(cmdValue.bulk)
	} else {
		cmd = strings.ToUpper(cmdValue.str)
	}

	// Process the command based on what's received
	switch cmd {
	case "PING":
		// ping can have 0 or 1 argument only
		if len(value.array) > 2 {
			return Value{typ: "error", str: "ERR wrong number of arguments for 'PING' command"}
		}

		// if there's no args, returning PONG
		if len(value.array) == 1 {
			return Value{typ: "string", str: "PONG"}
		}

		// if an argument is provided return that argument
		arg := value.array[1]
		if arg.typ == "bulk" {
			return Value{typ: "bulk", bulk: arg.bulk}
		}
		return Value{typ: "string", str: arg.str}

	case "SET", "PUT":
		// vaalidate args
		if len(value.array) != 3 {
			return Value{typ: "error", str: fmt.Sprintf("ERR wrong number of arguments for '%s' command", cmd)}
		}

		// check validity of key/value
		var key, val string
		if value.array[1].typ == "bulk" {
			key = value.array[1].bulk
		} else {
			key = value.array[1].str
		}

		if value.array[2].typ == "bulk" {
			val = value.array[2].bulk
		} else {
			val = value.array[2].str
		}

		// validate key and value length constraints
		if len(key) > 256 || len(val) > 256 {
			return Value{typ: "error", str: "ERR key or value too long (max 256 chars)"}
		}

		// add to cache using our optimized LRU
		cache.Put(key, val)

		// reeturn success
		if cmd == "PUT" {
			return Value{typ: "bulk", bulk: `{"status":"OK","message":"Key inserted/updated successfully."}`}
		}
		return Value{typ: "string", str: "OK"}

	case "GET":
		// validate args
		if len(value.array) != 2 {
			return Value{typ: "error", str: "ERR wrong number of arguments for 'GET' command"}
		}

		// Get key
		var key string
		if value.array[1].typ == "bulk" {
			key = value.array[1].bulk
		} else {
			key = value.array[1].str
		}

		// get value from our optimized LRU
		val, exists := cache.Get(key)
		if !exists {
			if cmd == "GET" && strings.HasPrefix(key, "http") {
				// Handle special HTTP-like GET requests
				return Value{typ: "bulk", bulk: `{"status":"ERROR","message":"Key not found."}`}
			}
			return Value{typ: "null"}
		}

		return Value{typ: "bulk", bulk: val}

	case "STATS":
		// get cache statistics
		stats := cache.Stats()

		// format as a simple string
		statsStr := fmt.Sprintf(
			"Capacity: %d, Size: %d, Get Ops: %d, Put Ops: %d, Hits: %d, Misses: %d, Hit Rate: %.2f%%, Evictions: %d",
			stats["capacity"], stats["size"], stats["get_ops"], stats["put_ops"],
			stats["hits"], stats["misses"], stats["hit_rate"], stats["evictions"],
		)

		return Value{typ: "string", str: statsStr}

	default:
		return Value{typ: "error", str: fmt.Sprintf("ERR unknown command '%s'", cmd)}
	}
}
