# Optimal Load Balancer Design in Go

A comprehensive guide to designing production-ready, extensible, and high-performance load balancers.

---

## Table of Contents

1. [Core Principles](#1-core-principles)
2. [The Interface Design](#2-the-interface-design)
3. [Health-Aware Load Balancing](#3-health-aware-load-balancing)
4. [Load Balancing Algorithms](#4-load-balancing-algorithms)
5. [Advanced Patterns](#5-advanced-patterns)
6. [Production Considerations](#6-production-considerations)

---

## 1. Core Principles

### 1.1 Separation of Concerns

A load balancer should have a single responsibility: **select a backend**. Health checking, connection pooling, and retry logic should be handled by separate components.

```go
// GOOD: Single responsibility
type Balancer interface {
    NextBackend(ctx context.Context) (Backend, error)
}

// BAD: Too many responsibilities
type BadBalancer interface {
    NextBackend() (Backend, error)
    CheckHealth()
    Connect()
    Disconnect()
    Retry()
}
```

### 1.2 Interface Segregation

Define small, focused interfaces that can be composed:

```go
// Core selection interface
type Selector interface {
    Next(ctx context.Context) (Backend, error)
}

// Health-aware components implement this
type HealthAware interface {
    SetHealth(backend string, healthy bool)
    IsHealthy(backend string) bool
}

// For balancers that track statistics
type StatsProvider interface {
    Stats() BalancerStats
}

// Composed interface for full-featured balancers
type LoadBalancer interface {
    Selector
    HealthAware
    StatsProvider
}
```

### 1.3 Fail Fast

Always return errors immediately rather than logging and continuing:

```go
// GOOD: Fail fast
func (lb *LoadBalancer) Next(ctx context.Context) (Backend, error) {
    if len(lb.backends) == 0 {
        return Backend{}, ErrNoBackends
    }
    
    backend, err := lb.selectBackend(ctx)
    if err != nil {
        return Backend{}, fmt.Errorf("select backend: %w", err)
    }
    
    return backend, nil
}
```

---

## 2. The Interface Design

### 2.1 The Core Types

```go
package balancer

import (
    "context"
    "errors"
    "sync"
    "sync/atomic"
    "time"
)

// Backend represents a single backend server
type Backend struct {
    ID       string
    Address  string
    Weight   int
    Metadata map[string]string
}

// BackendState tracks runtime state
type BackendState struct {
    Backend     Backend
    Healthy     atomic.Bool
    LastChecked time.Time
    
    // For weighted algorithms
    CurrentWeight atomic.Int64
    
    // For statistics
    Requests     atomic.Int64
    Failures     atomic.Int64
    Latency      atomic.Int64 // Nanoseconds
}

var (
    ErrNoBackends       = errors.New("no backends available")
    ErrNoHealthyBackends = errors.New("no healthy backends available")
    ErrContextCanceled  = errors.New("context canceled")
)

// Balancer is the core interface
type Balancer interface {
    // Next selects the next available backend
    Next(ctx context.Context) (Backend, error)
    
    // UpdateBackends atomically updates the backend list
    UpdateBackends(backends []Backend)
    
    // GetBackends returns current backend list
    GetBackends() []BackendState
}

// HealthObserver receives health change notifications
type HealthObserver interface {
    OnHealthChanged(backend Backend, healthy bool)
}

// Picker is the strategy interface for selection algorithms
type Picker interface {
    Pick(states []BackendState) (int, error) // Returns index
}
```

### 2.2 The Base Implementation

```go
// BaseBalancer provides common functionality
type BaseBalancer struct {
    mu       sync.RWMutex
    states   []BackendState
    picker   Picker
    
    observers []HealthObserver
}

func NewBaseBalancer(picker Picker) *BaseBalancer {
    return &BaseBalancer{
        picker: picker,
    }
}

func (b *BaseBalancer) Next(ctx context.Context) (Backend, error) {
    if err := ctx.Err(); err != nil {
        return Backend{}, ErrContextCanceled
    }
    
    b.mu.RLock()
    defer b.mu.RUnlock()
    
    if len(b.states) == 0 {
        return Backend{}, ErrNoBackends
    }
    
    // Try to find a healthy backend with limited retries
    for attempts := 0; attempts < len(b.states); attempts++ {
        idx, err := b.picker.Pick(b.states)
        if err != nil {
            return Backend{}, err
        }
        
        state := &b.states[idx]
        state.Requests.Add(1)
        
        if state.Healthy.Load() {
            return state.Backend, nil
        }
    }
    
    return Backend{}, ErrNoHealthyBackends
}

func (b *BaseBalancer) UpdateBackends(backends []Backend) {
    b.mu.Lock()
    defer b.mu.Unlock()
    
    newStates := make([]BackendState, len(backends))
    for i, be := range backends {
        newStates[i] = BackendState{
            Backend: be,
        }
        newStates[i].Healthy.Store(true) // Default to healthy
    }
    
    b.states = newStates
}

func (b *BaseBalancer) GetBackends() []BackendState {
    b.mu.RLock()
    defer b.mu.RUnlock()
    
    result := make([]BackendState, len(b.states))
    copy(result, b.states)
    return result
}

func (b *BaseBalancer) SetHealth(backendID string, healthy bool) {
    b.mu.Lock()
    defer b.mu.Unlock()
    
    for i := range b.states {
        if b.states[i].Backend.ID == backendID {
            oldHealth := b.states[i].Healthy.Swap(healthy)
            if oldHealth != healthy {
                b.notifyObservers(b.states[i].Backend, healthy)
            }
            return
        }
    }
}

func (b *BaseBalancer) notifyObservers(backend Backend, healthy bool) {
    for _, obs := range b.observers {
        obs.OnHealthChanged(backend, healthy)
    }
}
```

---

## 3. Health-Aware Load Balancing

### 3.1 The Problem

Most simple load balancers select backends without considering health status, causing requests to fail against unhealthy backends.

### 3.2 Health-Aware Base Implementation

```go
// HealthAwareBalancer wraps any picker with health checking
type HealthAwareBalancer struct {
    *BaseBalancer
    maxRetries int
}

func NewHealthAwareBalancer(picker Picker, maxRetries int) *HealthAwareBalancer {
    if maxRetries <= 0 {
        maxRetries = 3
    }
    
    return &HealthAwareBalancer{
        BaseBalancer: NewBaseBalancer(picker),
        maxRetries:   maxRetries,
    }
}

func (h *HealthAwareBalancer) Next(ctx context.Context) (Backend, error) {
    h.mu.RLock()
    defer h.mu.RUnlock()
    
    if len(h.states) == 0 {
        return Backend{}, ErrNoBackends
    }
    
    // Quick check: do we have ANY healthy backends?
    hasHealthy := false
    for i := range h.states {
        if h.states[i].Healthy.Load() {
            hasHealthy = true
            break
        }
    }
    
    if !hasHealthy {
        return Backend{}, ErrNoHealthyBackends
    }
    
    // Try to find a healthy backend
    for attempts := 0; attempts < h.maxRetries && attempts < len(h.states); attempts++ {
        idx, err := h.picker.Pick(h.states)
        if err != nil {
            return Backend{}, err
        }
        
        state := &h.states[idx]
        state.Requests.Add(1)
        
        if state.Healthy.Load() {
            return state.Backend, nil
        }
    }
    
    return Backend{}, ErrNoHealthyBackends
}
```

### 3.3 Health Checker Integration

```go
type HealthChecker struct {
    interval   time.Duration
    timeout    time.Duration
    threshold  int  // Failures before marking unhealthy
    
    client     *http.Client
    balancer   HealthAware
    stopCh     chan struct{}
}

type HealthAware interface {
    SetHealth(backendID string, healthy bool)
    GetBackends() []BackendState
}

func NewHealthChecker(balancer HealthAware, interval, timeout time.Duration) *HealthChecker {
    return &HealthChecker{
        interval:  interval,
        timeout:   timeout,
        threshold: 2,
        client: &http.Client{
            Timeout: timeout,
        },
        balancer: balancer,
        stopCh:   make(chan struct{}),
    }
}

func (hc *HealthChecker) Start() {
    ticker := time.NewTicker(hc.interval)
    
    go func() {
        defer ticker.Stop()
        
        for {
            select {
            case <-hc.stopCh:
                return
            case <-ticker.C:
                hc.checkAll()
            }
        }
    }()
}

func (hc *HealthChecker) Stop() {
    close(hc.stopCh)
}

func (hc *HealthChecker) checkAll() {
    backends := hc.balancer.GetBackends()
    
    var wg sync.WaitGroup
    for _, be := range backends {
        wg.Add(1)
        go func(backend Backend) {
            defer wg.Done()
            
            healthy := hc.probe(backend)
            hc.balancer.SetHealth(backend.ID, healthy)
        }(be.Backend)
    }
    
    wg.Wait()
}

func (hc *HealthChecker) probe(backend Backend) bool {
    ctx, cancel := context.WithTimeout(context.Background(), hc.timeout)
    defer cancel()
    
    url := "http://" + backend.Address + "/health"
    req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
    if err != nil {
        return false
    }
    
    resp, err := hc.client.Do(req)
    if err != nil {
        return false
    }
    defer resp.Body.Close()
    
    return resp.StatusCode == http.StatusOK
}
```

---

## 4. Load Balancing Algorithms

### 4.1 Round Robin

```go
// RoundRobin implements simple round-robin selection
type RoundRobin struct {
    counter atomic.Uint64
}

func NewRoundRobin() *RoundRobin {
    return &RoundRobin{}
}

func (rr *RoundRobin) Pick(states []BackendState) (int, error) {
    if len(states) == 0 {
        return 0, ErrNoBackends
    }
    
    // atomic.AddUint64 returns the new value, so subtract 1
    idx := (rr.counter.Add(1) - 1) % uint64(len(states))
    return int(idx), nil
}

// Weighted Round Robin
type WeightedRoundRobin struct {
    counter atomic.Uint64
}

func (wrr *WeightedRoundRobin) Pick(states []BackendState) (int, error) {
    if len(states) == 0 {
        return 0, ErrNoBackends
    }
    
    // Calculate total weight
    var totalWeight int
    for _, s := range states {
        totalWeight += s.Backend.Weight
    }
    
    if totalWeight == 0 {
        return 0, errors.New("total weight is zero")
    }
    
    // Find backend based on weighted selection
    counter := wrr.counter.Add(1) - 1
    target := int(counter % uint64(totalWeight))
    
    current := 0
    for i, s := range states {
        current += s.Backend.Weight
        if target < current {
            return i, nil
        }
    }
    
    return len(states) - 1, nil
}
```

### 4.2 Least Connections

```go
// LeastConnections picks the backend with fewest active connections
type LeastConnections struct{}

func (lc *LeastConnections) Pick(states []BackendState) (int, error) {
    if len(states) == 0 {
        return 0, ErrNoBackends
    }
    
    // Find backend with minimum active connections
    minIdx := 0
    minConns := states[0].Requests.Load() // Simplified - use proper active conns
    
    for i := 1; i < len(states); i++ {
        conns := states[i].Requests.Load()
        if conns < minConns {
            minConns = conns
            minIdx = i
        }
    }
    
    return minIdx, nil
}

// With connection tracking
type ConnectionTrackingBalancer struct {
    *BaseBalancer
    activeConns map[string]*atomic.Int64
}

func (ct *ConnectionTrackingBalancer) Next(ctx context.Context) (Backend, error) {
    backend, err := ct.BaseBalancer.Next(ctx)
    if err != nil {
        return Backend{}, err
    }
    
    // Increment active connections
    ct.activeConns[backend.ID].Add(1)
    
    // Return wrapper that decrements on completion
    return backend, nil
}
```

### 4.3 Consistent Hashing

```go
// ConsistentHash routes requests to same backend based on key
type ConsistentHash struct {
    replicas int // Virtual nodes per backend
    ring     []uint32           // Sorted hash ring
    nodes    map[uint32]string  // Hash -> backend ID
}

func NewConsistentHash(replicas int) *ConsistentHash {
    if replicas <= 0 {
        replicas = 150
    }
    
    return &ConsistentHash{
        replicas: replicas,
        nodes:    make(map[uint32]string),
    }
}

func (ch *ConsistentHash) Add(backend Backend) {
    for i := 0; i < ch.replicas; i++ {
        hash := ch.hash(fmt.Sprintf("%s:%d", backend.ID, i))
        ch.nodes[hash] = backend.ID
        ch.ring = append(ch.ring, hash)
    }
    sort.Slice(ch.ring, func(i, j int) bool {
        return ch.ring[i] < ch.ring[j]
    })
}

func (ch *ConsistentHash) Get(key string) string {
    if len(ch.ring) == 0 {
        return ""
    }
    
    hash := ch.hash(key)
    
    // Binary search for first node >= hash
    idx := sort.Search(len(ch.ring), func(i int) bool {
        return ch.ring[i] >= hash
    })
    
    if idx == len(ch.ring) {
        idx = 0
    }
    
    return ch.nodes[ch.ring[idx]]
}

func (ch *ConsistentHash) hash(key string) uint32 {
    h := fnv.New32a()
    h.Write([]byte(key))
    return h.Sum32()
}
```

### 4.4 Power of Two Choices

```go
// PowerOfTwo provides better load distribution than random
type PowerOfTwo struct {
    rng *rand.Rand
}

func NewPowerOfTwo() *PowerOfTwo {
    return &PowerOfTwo{
        rng: rand.New(rand.NewSource(time.Now().UnixNano())),
    }
}

func (p2 *PowerOfTwo) Pick(states []BackendState) (int, error) {
    if len(states) == 0 {
        return 0, ErrNoBackends
    }
    
    if len(states) == 1 {
        return 0, nil
    }
    
    // Pick two random candidates
    a := p2.rng.Intn(len(states))
    b := p2.rng.Intn(len(states))
    for b == a {
        b = p2.rng.Intn(len(states))
    }
    
    // Select the one with fewer connections
    if states[a].Requests.Load() < states[b].Requests.Load() {
        return a, nil
    }
    return b, nil
}
```

---

## 5. Advanced Patterns

### 5.1 Circuit Breaker Pattern

```go
type CircuitState int

const (
    CircuitClosed CircuitState = iota
    CircuitOpen
    CircuitHalfOpen
)

type CircuitBreaker struct {
    failureThreshold int
    successThreshold int
    timeout          time.Duration
    
    state      atomic.Int32
    failures   atomic.Int32
    successes  atomic.Int32
    lastFailureTime atomic.Int64
}

func (cb *CircuitBreaker) Allow() bool {
    state := CircuitState(cb.state.Load())
    
    switch state {
    case CircuitClosed:
        return true
        
    case CircuitOpen:
        // Check if timeout has passed
        lastFail := cb.lastFailureTime.Load()
        if time.Since(time.Unix(lastFail, 0)) > cb.timeout {
            // Try half-open
            if cb.state.CompareAndSwap(int32(CircuitOpen), int32(CircuitHalfOpen)) {
                cb.failures.Store(0)
                cb.successes.Store(0)
            }
            return true
        }
        return false
        
    case CircuitHalfOpen:
        return true
        
    default:
        return false
    }
}

func (cb *CircuitBreaker) RecordSuccess() {
    state := CircuitState(cb.state.Load())
    
    if state == CircuitHalfOpen {
        successes := cb.successes.Add(1)
        if successes >= int32(cb.successThreshold) {
            cb.state.Store(int32(CircuitClosed))
        }
    }
}

func (cb *CircuitBreaker) RecordFailure() {
    cb.lastFailureTime.Store(time.Now().Unix())
    
    state := CircuitState(cb.state.Load())
    
    if state == CircuitHalfOpen {
        cb.state.Store(int32(CircuitOpen))
        return
    }
    
    failures := cb.failures.Add(1)
    if failures >= int32(cb.failureThreshold) {
        cb.state.Store(int32(CircuitOpen))
    }
}
```

### 5.2 Retry with Backoff

```go
type RetryConfig struct {
    MaxRetries  int
    BaseDelay   time.Duration
    MaxDelay    time.Duration
    Multiplier  float64
}

type RetryBalancer struct {
    balancer Balancer
    config   RetryConfig
}

func (rb *RetryBalancer) Next(ctx context.Context) (Backend, error) {
    var lastErr error
    
    for attempt := 0; attempt <= rb.config.MaxRetries; attempt++ {
        backend, err := rb.balancer.Next(ctx)
        if err == nil {
            return backend, nil
        }
        
        lastErr = err
        
        if attempt < rb.config.MaxRetries {
            // Calculate backoff
            delay := rb.calculateDelay(attempt)
            
            select {
            case <-ctx.Done():
                return Backend{}, ctx.Err()
            case <-time.After(delay):
                // Retry
            }
        }
    }
    
    return Backend{}, fmt.Errorf("max retries exceeded: %w", lastErr)
}

func (rb *RetryBalancer) calculateDelay(attempt int) time.Duration {
    delay := rb.config.BaseDelay
    
    for i := 0; i < attempt; i++ {
        delay = time.Duration(float64(delay) * rb.config.Multiplier)
    }
    
    if delay > rb.config.MaxDelay {
        delay = rb.config.MaxDelay
    }
    
    // Add jitter
    jitter := time.Duration(rand.Float64() * float64(delay))
    return delay + jitter
}
```

### 5.3 Composite Balancer (Primary/Backup)

```go
// PrimaryBackup tries primary first, falls back to backup
type PrimaryBackup struct {
    primary Balancer
    backup  Balancer
}

func (pb *PrimaryBackup) Next(ctx context.Context) (Backend, error) {
    backend, err := pb.primary.Next(ctx)
    if err == nil {
        return backend, nil
    }
    
    // Primary failed, try backup
    return pb.backup.Next(ctx)
}

// Tiered tries multiple tiers in order
type TieredBalancer struct {
    tiers []Balancer
}

func (tb *TieredBalancer) Next(ctx context.Context) (Backend, error) {
    for _, tier := range tb.tiers {
        backend, err := tier.Next(ctx)
        if err == nil {
            return backend, nil
        }
    }
    
    return Backend{}, ErrNoBackends
}
```

---

## 6. Production Considerations

### 6.1 Observability

```go
type InstrumentedBalancer struct {
    Balancer
    metrics MetricsRecorder
    tracer  trace.Tracer
}

func (ib *InstrumentedBalancer) Next(ctx context.Context) (Backend, error) {
    ctx, span := ib.tracer.Start(ctx, "balancer.select")
    defer span.End()
    
    start := time.Now()
    backend, err := ib.Balancer.Next(ctx)
    duration := time.Since(start)
    
    // Record metrics
    if err != nil {
        ib.metrics.IncrementCounter("balancer.errors", 
            "error", err.Error())
        span.RecordError(err)
    } else {
        ib.metrics.IncrementCounter("balancer.success",
            "backend", backend.ID)
        ib.metrics.RecordHistogram("balancer.duration", 
            duration.Milliseconds())
    }
    
    return backend, err
}
```

### 6.2 Hot Reload

```go
type DynamicBalancer struct {
    mu       sync.RWMutex
    balancer Balancer
}

func (db *DynamicBalancer) Next(ctx context.Context) (Backend, error) {
    db.mu.RLock()
    b := db.balancer
    db.mu.RUnlock()
    
    return b.Next(ctx)
}

func (db *DynamicBalancer) Update(balancer Balancer) {
    db.mu.Lock()
    defer db.mu.Unlock()
    
    db.balancer = balancer
}
```

### 6.3 Testing

```go
// Mock balancer for testing
type MockBalancer struct {
    mu        sync.Mutex
    backends  []Backend
    callCount int
}

func (m *MockBalancer) Next(ctx context.Context) (Backend, error) {
    m.mu.Lock()
    defer m.mu.Unlock()
    
    m.callCount++
    
    if len(m.backends) == 0 {
        return Backend{}, ErrNoBackends
    }
    
    idx := (m.callCount - 1) % len(m.backends)
    return m.backends[idx], nil
}

func (m *MockBalancer) SetBackends(backends []Backend) {
    m.mu.Lock()
    defer m.mu.Unlock()
    m.backends = backends
}

// Property-based test
func TestBalancerProperties(t *testing.T) {
    backends := []Backend{
        {ID: "a", Address: "localhost:8001"},
        {ID: "b", Address: "localhost:8002"},
        {ID: "c", Address: "localhost:8003"},
    }
    
    balancers := []struct {
        name     string
        balancer Balancer
    }{
        {"roundrobin", NewBaseBalancer(NewRoundRobin())},
        {"leastconn", NewBaseBalancer(&LeastConnections{})},
        {"power2", NewBaseBalancer(NewPowerOfTwo())},
    }
    
    for _, tc := range balancers {
        t.Run(tc.name, func(t *testing.T) {
            tc.balancer.UpdateBackends(backends)
            
            // Property: Should eventually select all backends
            selected := make(map[string]bool)
            for i := 0; i < len(backends)*10; i++ {
                be, err := tc.balancer.Next(context.Background())
                require.NoError(t, err)
                selected[be.ID] = true
            }
            
            assert.Len(t, selected, len(backends))
        })
    }
}
```

### 6.4 Benchmarks

```go
func BenchmarkRoundRobin(b *testing.B) {
    backends := []Backend{
        {ID: "a", Address: "localhost:8001"},
        {ID: "b", Address: "localhost:8002"},
        {ID: "c", Address: "localhost:8003"},
    }
    
    balancer := NewBaseBalancer(NewRoundRobin())
    balancer.UpdateBackends(backends)
    
    ctx := context.Background()
    
    b.ResetTimer()
    b.RunParallel(func(pb *testing.PB) {
        for pb.Next() {
            _, err := balancer.Next(ctx)
            if err != nil {
                b.Fatal(err)
            }
        }
    })
}

// Benchmark with health checking
func BenchmarkHealthAwareRoundRobin(b *testing.B) {
    backends := []Backend{
        {ID: "a", Address: "localhost:8001"},
        {ID: "b", Address: "localhost:8002"},
        {ID: "c", Address: "localhost:8003"},
    }
    
    base := NewBaseBalancer(NewRoundRobin())
    balancer := NewHealthAwareBalancer(base.picker, 3)
    balancer.UpdateBackends(backends)
    
    ctx := context.Background()
    
    b.ResetTimer()
    b.RunParallel(func(pb *testing.PB) {
        for pb.Next() {
            _, err := balancer.Next(ctx)
            if err != nil {
                b.Fatal(err)
            }
        }
    })
}
```

---

## Summary

An optimal load balancer should:

1. **Have clear interfaces** - Small, composable interfaces (Selector, HealthAware, etc.)
2. **Be health-aware** - Skip unhealthy backends automatically
3. **Support multiple algorithms** - Round-robin, least-connections, consistent hashing
4. **Handle failures gracefully** - Circuit breakers, retries with backoff
5. **Be observable** - Metrics, tracing, structured logging
6. **Be testable** - Interface-based design allows mocking
7. **Be performant** - Lock-free where possible, minimize allocations
8. **Support hot reload** - Update backends without restart

The key insight is that a load balancer is a **composition of strategies**: selection strategy + health checking + retry logic + circuit breaking. Each should be swappable and testable independently.
