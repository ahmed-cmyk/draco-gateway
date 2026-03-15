# 🚀 GopherGate: A Go-Powered API Gateway

A lightweight, high-performance API Gateway built from scratch in Go. This project is a deep dive into Go's `net/http` package, concurrency patterns, and distributed systems architecture.

## 📖 Overview

**GopherGate** acts as the single entry point for your microservices. It handles the "cross-cutting concerns" like authentication, rate limiting, and request routing so your backend services don't have to.

### Why build this?

* To master the `httputil.ReverseProxy` standard library.
* To implement common distributed system patterns (Circuit Breakers, Retries).
* To practice writing high-performance middleware.

---

## 🛠 Feature Roadmap

I am building this incrementally. Here is the current status of the gateway:

### Phase 1: Core Routing

* [x] **Reverse Proxy:** Forwarding traffic to backend targets.
* [x] **Dynamic Path Matching:** Routing based on URL prefixes (e.g., `/users/*`).
* [ ] **Method Filtering:** Restricting routes to specific HTTP verbs (GET, POST, etc.).
* [ ] **Header Transformation:** Adding/Removing headers before forwarding.

### Phase 2: Traffic Control

* [ ] **Load Balancing:** Round-robin distribution across multiple service instances.
* [x] **Rate Limiting:** Protect backends using the Token Bucket algorithm.
* [ ] **Health Checks:** Automatically removing unhealthy backends from the pool.

### Phase 3: Security & Ops

* [ ] **JWT Validation:** Centralized authentication at the edge.
* [ ] **CORS Middleware:** Global configuration for cross-origin requests.
* [ ] **Structured Logging:** Zap or Logrus integration for JSON logging.
* [ ] **Prometheus Metrics:** Tracking request latency and 5xx errors.

---

## 🚦 Getting Started

### Prerequisites

* Go 1.21+
* Make (optional)

### Installation

```bash
git clone https://github.com/ahmed-cmyk/gophergate.git
cd gophergate
go mod download

```

### Running the Gateway

```bash
go run main.go --config config.yaml

```

---

## ⚙️ Configuration Example

The gateway is configured via a simple YAML file:

```yaml
server:
  port: 8080

routes:
  - path: /api/v1/users
    target: "http://user-service:8081"
    strip_prefix: true
    middlewares:
      - rate_limit
      - auth

```

---

## 🧪 Testing

To run the suite of unit and integration tests:

```bash
go test ./... -v

```

---