# Gateway Service Architecture

> API gateway providing authentication, routing, and traffic management for the Unkey platform

## Table of Contents
- [Overview](#overview)
- [Architecture](#architecture)
- [Component Details](#component-details)
- [Authentication System](#authentication-system)
- [Request Routing](#request-routing)
- [Rate Limiting](#rate-limiting)
- [Security Features](#security-features)
- [Performance & Scalability](#performance--scalability)
- [Operational Considerations](#operational-considerations)
- [Cross-References](#cross-references)

---

## Overview

The Gateway Service is the central entry point for all API traffic in the Unkey platform, providing authentication, authorization, request routing, and traffic management capabilities.

### Key Responsibilities
- **Authentication & Authorization**: JWT validation and customer context management
- **Request Routing**: Intelligent routing to appropriate metald instances
- **Rate Limiting**: Per-customer and global traffic control
- **Load Balancing**: Traffic distribution across metald instances
- **Security**: DDoS protection, input validation, and threat detection
- **Observability**: Request tracing, metrics collection, and logging

### Technology Stack (Recommended)
- **Language**: Go 1.21+ (for consistency with metald)
- **Framework**: Chi router or Gin (high-performance HTTP routing)
- **Authentication**: JWT libraries (golang-jwt/jwt)
- **Rate Limiting**: Redis-backed sliding window
- **Load Balancing**: Consistent hashing or round-robin
- **Observability**: OpenTelemetry integration

---

## Architecture

### High-Level Gateway Architecture

```mermaid
graph TB
    subgraph "Ingress Layer"
        LB[Load Balancer<br/>- TLS Termination<br/>- Geographic Routing<br/>- Health Checks]
    end
    
    subgraph "Gateway Cluster"
        subgraph "Gateway Instance 1"
            GW1[Gateway Service<br/>- HTTP Server<br/>- Request Processing<br/>- Response Handling]
        end
        
        subgraph "Gateway Instance 2"
            GW2[Gateway Service<br/>- HTTP Server<br/>- Request Processing<br/>- Response Handling]
        end
        
        subgraph "Gateway Instance N"
            GWN[Gateway Service<br/>- HTTP Server<br/>- Request Processing<br/>- Response Handling]
        end
    end
    
    subgraph "Shared Services"
        Redis[(Redis<br/>- Rate Limiting<br/>- Session Storage<br/>- Cache)]
        AuthService[Auth Service<br/>- JWT Validation<br/>- Customer Lookup<br/>- Permissions]
    end
    
    subgraph "Backend Services"
        subgraph "Metald Cluster"
            M1[Metald Instance 1<br/>Customer Pool A]
            M2[Metald Instance 2<br/>Customer Pool B]
            MN[Metald Instance N<br/>Customer Pool N]
        end
    end
    
    LB --> GW1
    LB --> GW2
    LB --> GWN
    
    GW1 --> Redis
    GW2 --> Redis
    GWN --> Redis
    
    GW1 --> AuthService
    GW2 --> AuthService
    GWN --> AuthService
    
    GW1 --> M1
    GW1 --> M2
    GW2 --> M1
    GW2 --> MN
    GWN --> M2
    GWN --> MN
```

### Request Processing Pipeline

```mermaid
graph LR
    subgraph "Ingress"
        Request[Client Request<br/>- HTTP/gRPC<br/>- Headers<br/>- Payload]
    end
    
    subgraph "Pre-Processing"
        Validation[Input Validation<br/>- Schema Validation<br/>- Content-Type<br/>- Size Limits]
        Security[Security Checks<br/>- DDoS Detection<br/>- IP Filtering<br/>- Threat Analysis]
    end
    
    subgraph "Authentication"
        TokenExtract[Token Extraction<br/>- Bearer Header<br/>- Query Parameter<br/>- Cookie]
        TokenValidate[Token Validation<br/>- JWT Signature<br/>- Expiry Check<br/>- Claims Validation]
        CustomerContext[Customer Context<br/>- Customer ID<br/>- Permissions<br/>- Metadata]
    end
    
    subgraph "Authorization"
        RateLimit[Rate Limiting<br/>- Per-Customer Limits<br/>- Global Limits<br/>- Burst Handling]
        PermissionCheck[Permission Check<br/>- Operation Access<br/>- Resource Quotas<br/>- Feature Flags]
    end
    
    subgraph "Routing"
        TargetSelection[Target Selection<br/>- Customer Sharding<br/>- Load Balancing<br/>- Health Checks]
        RequestTransform[Request Transform<br/>- Header Injection<br/>- Protocol Translation<br/>- Context Propagation]
    end
    
    subgraph "Backend"
        MetaldCall[Metald Call<br/>- HTTP/gRPC Call<br/>- Timeout Handling<br/>- Retry Logic]
        ResponseTransform[Response Transform<br/>- Error Mapping<br/>- Format Conversion<br/>- Header Processing]
    end
    
    Request --> Validation
    Validation --> Security
    Security --> TokenExtract
    TokenExtract --> TokenValidate
    TokenValidate --> CustomerContext
    CustomerContext --> RateLimit
    RateLimit --> PermissionCheck
    PermissionCheck --> TargetSelection
    TargetSelection --> RequestTransform
    RequestTransform --> MetaldCall
    MetaldCall --> ResponseTransform
    ResponseTransform --> Response[Client Response]
```

---

## Component Details

### Core Gateway Components

```mermaid
graph TB
    subgraph "HTTP Server Layer"
        HTTPServer[HTTP Server<br/>- TLS Configuration<br/>- Connection Pooling<br/>- Graceful Shutdown]
        Middleware[Middleware Stack<br/>- Logging<br/>- CORS<br/>- Compression<br/>- Recovery]
    end
    
    subgraph "Request Processing"
        Router[Request Router<br/>- Path Matching<br/>- Method Validation<br/>- Version Handling]
        Validator[Request Validator<br/>- Schema Validation<br/>- Input Sanitization<br/>- Type Checking]
    end
    
    subgraph "Authentication Module"
        JWTValidator[JWT Validator<br/>- Signature Verification<br/>- Claims Extraction<br/>- Expiry Validation]
        CustomerExtractor[Customer Extractor<br/>- Customer ID<br/>- Metadata Lookup<br/>- Context Building]
    end
    
    subgraph "Rate Limiting Module"
        RateLimiter[Rate Limiter<br/>- Token Bucket<br/>- Sliding Window<br/>- Redis Backend]
        QuotaManager[Quota Manager<br/>- Customer Limits<br/>- Feature Quotas<br/>- Usage Tracking]
    end
    
    subgraph "Routing Module"
        LoadBalancer[Load Balancer<br/>- Health Checking<br/>- Consistent Hashing<br/>- Failover Logic]
        ServiceDiscovery[Service Discovery<br/>- Metald Instances<br/>- Health Status<br/>- Configuration]
    end
    
    subgraph "Backend Communication"
        HTTPClient[HTTP Client<br/>- Connection Pooling<br/>- Timeout Management<br/>- Retry Logic]
        CircuitBreaker[Circuit Breaker<br/>- Failure Detection<br/>- Automatic Recovery<br/>- Fallback Handling]
    end
    
    HTTPServer --> Middleware
    Middleware --> Router
    Router --> Validator
    
    Validator --> JWTValidator
    JWTValidator --> CustomerExtractor
    CustomerExtractor --> RateLimiter
    RateLimiter --> QuotaManager
    
    QuotaManager --> LoadBalancer
    LoadBalancer --> ServiceDiscovery
    ServiceDiscovery --> HTTPClient
    HTTPClient --> CircuitBreaker
```

---

## Authentication System

### JWT Authentication Flow

```mermaid
sequenceDiagram
    participant Client
    participant Gateway
    participant AuthService
    participant Redis
    participant Metald
    
    Client->>Gateway: API Request + JWT Token
    Gateway->>Gateway: Extract Bearer Token
    
    alt Token in Cache
        Gateway->>Redis: Check Token Cache
        Redis->>Gateway: Cached Customer Context
    else Token Not Cached
        Gateway->>AuthService: Validate JWT
        AuthService->>AuthService: Verify Signature
        AuthService->>AuthService: Check Expiry
        AuthService->>AuthService: Extract Claims
        AuthService->>Gateway: Customer Context
        Gateway->>Redis: Cache Customer Context
    end
    
    Gateway->>Gateway: Build Request Context
    Gateway->>Metald: Forward Request + Context
    Metald->>Gateway: Response
    Gateway->>Client: Final Response
```

### Customer Context Management

```mermaid
graph TB
    subgraph "Token Processing"
        TokenHeader[Authorization Header<br/>Bearer <jwt-token>]
        TokenParsing[Token Parsing<br/>- Header Validation<br/>- Base64 Decoding<br/>- JSON Parsing]
        SignatureVerify[Signature Verification<br/>- Algorithm Check<br/>- Key Validation<br/>- Crypto Verification]
    end
    
    subgraph "Claims Extraction"
        StandardClaims[Standard Claims<br/>- iss (Issuer)<br/>- exp (Expiry)<br/>- iat (Issued At)]
        CustomClaims[Custom Claims<br/>- customer_id<br/>- permissions<br/>- features<br/>- quotas]
    end
    
    subgraph "Context Building"
        CustomerID[Customer ID<br/>- Primary Identifier<br/>- Routing Key<br/>- Ownership Context]
        Permissions[Permissions<br/>- API Access<br/>- Resource Limits<br/>- Feature Flags]
        Metadata[Metadata<br/>- Customer Tier<br/>- Rate Limits<br/>- Preferences]
    end
    
    TokenHeader --> TokenParsing
    TokenParsing --> SignatureVerify
    SignatureVerify --> StandardClaims
    SignatureVerify --> CustomClaims
    
    StandardClaims --> CustomerID
    CustomClaims --> CustomerID
    CustomClaims --> Permissions
    CustomClaims --> Metadata
```

### Development vs Production Authentication

```mermaid
graph LR
    subgraph "Development Mode"
        DevToken[Development Token<br/>dev_customer_123]
        DevValidation[Simple Validation<br/>- Prefix Check<br/>- Customer ID Extract<br/>- No Signature]
        DevContext[Dev Context<br/>- Customer ID<br/>- Full Permissions<br/>- No Limits]
    end
    
    subgraph "Production Mode"
        ProdToken[Production JWT<br/>Signed Token]
        ProdValidation[Full JWT Validation<br/>- Signature Verification<br/>- Expiry Check<br/>- Claims Validation]
        ProdContext[Prod Context<br/>- Customer ID<br/>- Limited Permissions<br/>- Rate Limits]
    end
    
    DevToken --> DevValidation
    DevValidation --> DevContext
    
    ProdToken --> ProdValidation
    ProdValidation --> ProdContext
```

---

## Request Routing

### Customer-Based Routing Strategy

```mermaid
graph TB
    subgraph "Routing Decision"
        CustomerID[Customer ID<br/>from JWT Context]
        HashFunction[Hash Function<br/>Consistent Hashing]
        InstancePool[Metald Instance Pool<br/>Available Instances]
    end
    
    subgraph "Routing Algorithms"
        ConsistentHash[Consistent Hashing<br/>- Customer Affinity<br/>- Even Distribution<br/>- Minimal Reshuffling]
        RoundRobin[Round Robin<br/>- Simple Distribution<br/>- No Affinity<br/>- Load Spreading]
        LeastConnections[Least Connections<br/>- Load-based<br/>- Dynamic Routing<br/>- Performance Optimized]
    end
    
    subgraph "Health Considerations"
        HealthCheck[Health Checking<br/>- Instance Availability<br/>- Response Time<br/>- Error Rate]
        Failover[Failover Logic<br/>- Unhealthy Instance<br/>- Automatic Rerouting<br/>- Traffic Shifting]
    end
    
    CustomerID --> HashFunction
    HashFunction --> InstancePool
    
    InstancePool --> ConsistentHash
    InstancePool --> RoundRobin
    InstancePool --> LeastConnections
    
    ConsistentHash --> HealthCheck
    RoundRobin --> HealthCheck
    LeastConnections --> HealthCheck
    
    HealthCheck --> Failover
```

### Routing Configuration

```yaml
# Gateway Routing Configuration
routing:
  strategy: "consistent_hash"  # consistent_hash, round_robin, least_connections
  health_check_interval: "30s"
  unhealthy_threshold: 3
  recovery_threshold: 2
  
  metald_instances:
    - host: "metald-1.internal"
      port: 8080
      weight: 100
      customer_pools: ["pool_a", "pool_b"]
    - host: "metald-2.internal" 
      port: 8080
      weight: 100
      customer_pools: ["pool_c", "pool_d"]
    - host: "metald-3.internal"
      port: 8080
      weight: 50
      customer_pools: ["overflow"]
```

### Load Balancing Flow

```mermaid
sequenceDiagram
    participant Gateway
    participant HealthChecker
    participant Instance1 as Metald Instance 1
    participant Instance2 as Metald Instance 2
    participant Instance3 as Metald Instance 3
    
    Note over Gateway,Instance3: Health Monitoring
    loop Every 30 seconds
        HealthChecker->>Instance1: Health Check
        Instance1->>HealthChecker: Healthy
        HealthChecker->>Instance2: Health Check
        Instance2->>HealthChecker: Healthy
        HealthChecker->>Instance3: Health Check
        Instance3->>HealthChecker: Unhealthy
    end
    
    Note over Gateway,Instance3: Request Routing
    Gateway->>Gateway: Customer Request (customer_123)
    Gateway->>Gateway: Hash(customer_123) → Instance2
    Gateway->>Instance2: Forward Request
    Instance2->>Gateway: Response
    
    Note over Gateway,Instance3: Failover Scenario
    Gateway->>Gateway: Customer Request (customer_456)
    Gateway->>Gateway: Hash(customer_456) → Instance3
    Gateway->>Gateway: Instance3 Unhealthy
    Gateway->>Gateway: Fallback to Instance1
    Gateway->>Instance1: Forward Request
    Instance1->>Gateway: Response
```

---

## Rate Limiting

### Multi-Level Rate Limiting

```mermaid
graph TB
    subgraph "Global Limits"
        GlobalRPS[Global RPS Limit<br/>- Total Requests/sec<br/>- DDoS Protection<br/>- System Capacity]
        GlobalConcurrent[Global Concurrent<br/>- Active Connections<br/>- Resource Protection<br/>- Memory Limits]
    end
    
    subgraph "Customer Limits"
        CustomerRPS[Customer RPS Limit<br/>- Per-Customer Rate<br/>- Tier-based Limits<br/>- Burst Allowance]
        CustomerQuota[Customer Quota<br/>- Daily/Monthly Limits<br/>- Feature-based Quotas<br/>- Usage Tracking]
    end
    
    subgraph "API Endpoint Limits"
        EndpointRPS[Endpoint RPS Limit<br/>- Per-Endpoint Rate<br/>- Operation-specific<br/>- Resource-based]
        CostBasedLimit[Cost-based Limiting<br/>- Expensive Operations<br/>- Resource Consumption<br/>- Weighted Requests]
    end
    
    subgraph "Implementation"
        TokenBucket[Token Bucket<br/>- Burst Handling<br/>- Smooth Rate Control<br/>- Redis Backend]
        SlidingWindow[Sliding Window<br/>- Precise Counting<br/>- Time-based Limits<br/>- Memory Efficient]
    end
    
    GlobalRPS --> TokenBucket
    GlobalConcurrent --> SlidingWindow
    CustomerRPS --> TokenBucket
    CustomerQuota --> SlidingWindow
    EndpointRPS --> TokenBucket
    CostBasedLimit --> SlidingWindow
```

### Rate Limiting Algorithm

```mermaid
sequenceDiagram
    participant Client
    participant Gateway
    participant Redis
    participant Metald
    
    Client->>Gateway: API Request
    Gateway->>Gateway: Extract Customer ID
    
    Note over Gateway,Redis: Global Rate Check
    Gateway->>Redis: Check Global RPS
    Redis->>Gateway: Current: 450/500 RPS
    
    Note over Gateway,Redis: Customer Rate Check
    Gateway->>Redis: Check Customer RPS (customer_123)
    Redis->>Gateway: Current: 8/10 RPS
    
    Note over Gateway,Redis: Endpoint Rate Check
    Gateway->>Redis: Check CreateVM RPS
    Redis->>Gateway: Current: 2/5 RPS
    
    alt All Limits OK
        Gateway->>Redis: Increment Counters
        Gateway->>Metald: Forward Request
        Metald->>Gateway: Response
        Gateway->>Client: Success Response
    else Rate Limit Exceeded
        Gateway->>Client: 429 Too Many Requests
        Note over Client: Rate limit headers included
    end
```

### Rate Limit Configuration

```yaml
# Rate Limiting Configuration
rate_limits:
  global:
    requests_per_second: 10000
    burst_size: 1000
    window_size: "1m"
  
  customer_tiers:
    free:
      requests_per_second: 10
      requests_per_day: 1000
      burst_size: 20
    
    pro:
      requests_per_second: 100
      requests_per_day: 100000
      burst_size: 200
    
    enterprise:
      requests_per_second: 1000
      requests_per_day: 1000000
      burst_size: 2000
  
  endpoints:
    CreateVM:
      requests_per_second: 5
      cost_weight: 10
    
    BootVM:
      requests_per_second: 20
      cost_weight: 5
    
    ListVMs:
      requests_per_second: 100
      cost_weight: 1
```

---

## Security Features

### DDoS Protection

```mermaid
graph TB
    subgraph "Detection Layer"
        TrafficAnalysis[Traffic Analysis<br/>- Request Patterns<br/>- Source IP Analysis<br/>- Anomaly Detection]
        ThresholdMonitoring[Threshold Monitoring<br/>- RPS Thresholds<br/>- Connection Limits<br/>- Bandwidth Limits]
    end
    
    subgraph "Protection Layer"
        IPBlocking[IP Blocking<br/>- Automatic Blacklisting<br/>- CIDR Blocking<br/>- Time-based Blocks]
        RateLimiting[Aggressive Rate Limiting<br/>- Reduced Limits<br/>- Challenge Responses<br/>- Delay Responses]
    end
    
    subgraph "Response Layer"
        AlertGeneration[Alert Generation<br/>- Security Team Alerts<br/>- Automated Responses<br/>- Incident Tracking]
        TrafficShedding[Traffic Shedding<br/>- Selective Dropping<br/>- Priority Handling<br/>- Circuit Breaking]
    end
    
    TrafficAnalysis --> IPBlocking
    ThresholdMonitoring --> RateLimiting
    IPBlocking --> AlertGeneration
    RateLimiting --> TrafficShedding
```

### Input Validation & Sanitization

```mermaid
graph LR
    subgraph "Request Validation"
        SchemaValidation[Schema Validation<br/>- JSON Schema<br/>- Required Fields<br/>- Type Checking]
        SizeValidation[Size Validation<br/>- Payload Limits<br/>- Header Limits<br/>- Parameter Limits]
        FormatValidation[Format Validation<br/>- Email Formats<br/>- UUID Formats<br/>- URL Validation]
    end
    
    subgraph "Content Sanitization"
        HTMLSanitization[HTML Sanitization<br/>- XSS Prevention<br/>- Script Removal<br/>- Tag Filtering]
        SQLInjectionPrevention[SQL Injection Prevention<br/>- Parameter Binding<br/>- Query Validation<br/>- Escape Sequences]
        CommandInjectionPrevention[Command Injection Prevention<br/>- Input Escaping<br/>- Whitelist Validation<br/>- Sandbox Execution]
    end
    
    subgraph "Security Headers"
        CSP[Content Security Policy<br/>- Script Sources<br/>- Resource Restrictions<br/>- Inline Prevention]
        HSTS[HTTP Strict Transport Security<br/>- HTTPS Enforcement<br/>- Subdomain Inclusion<br/>- Preload Lists]
        CSRF[CSRF Protection<br/>- Token Validation<br/>- Origin Checking<br/>- Referrer Validation]
    end
    
    SchemaValidation --> HTMLSanitization
    SizeValidation --> SQLInjectionPrevention
    FormatValidation --> CommandInjectionPrevention
    
    HTMLSanitization --> CSP
    SQLInjectionPrevention --> HSTS
    CommandInjectionPrevention --> CSRF
```

---

## Performance & Scalability

### Performance Characteristics

```mermaid
graph TB
    subgraph "Latency Profile"
        P50[P50 Latency<br/>< 10ms<br/>Typical Request]
        P95[P95 Latency<br/>< 50ms<br/>Complex Requests]
        P99[P99 Latency<br/>< 100ms<br/>Worst Case]
    end
    
    subgraph "Throughput Profile"
        RPS[Requests/Second<br/>10,000+ RPS<br/>Per Instance]
        Concurrent[Concurrent Connections<br/>10,000+ Connections<br/>Per Instance]
        Bandwidth[Bandwidth<br/>1 Gbps+<br/>Per Instance]
    end
    
    subgraph "Resource Usage"
        CPU[CPU Usage<br/>< 70%<br/>Under Normal Load]
        Memory[Memory Usage<br/>< 2GB<br/>Per Instance]
        Network[Network Usage<br/>< 500 Mbps<br/>Typical Load]
    end
    
    P50 --> RPS
    P95 --> Concurrent
    P99 --> Bandwidth
    
    RPS --> CPU
    Concurrent --> Memory
    Bandwidth --> Network
```

### Horizontal Scaling

```mermaid
graph TB
    subgraph "Load Balancer"
        LB[Load Balancer<br/>- Health Checks<br/>- Traffic Distribution<br/>- SSL Termination]
    end
    
    subgraph "Gateway Cluster"
        GW1[Gateway 1<br/>- Stateless<br/>- Auto-scaling<br/>- Redis Backend]
        GW2[Gateway 2<br/>- Stateless<br/>- Auto-scaling<br/>- Redis Backend]
        GW3[Gateway 3<br/>- Stateless<br/>- Auto-scaling<br/>- Redis Backend]
        GWN[Gateway N<br/>- Stateless<br/>- Auto-scaling<br/>- Redis Backend]
    end
    
    subgraph "Shared State"
        Redis[(Redis Cluster<br/>- Rate Limit State<br/>- Session Storage<br/>- Cache)]
    end
    
    subgraph "Auto-scaling Triggers"
        CPUTrigger[CPU > 70%<br/>Scale Up]
        LatencyTrigger[P95 > 100ms<br/>Scale Up]
        ConnectionsTrigger[Connections > 8000<br/>Scale Up]
    end
    
    LB --> GW1
    LB --> GW2
    LB --> GW3
    LB --> GWN
    
    GW1 --> Redis
    GW2 --> Redis
    GW3 --> Redis
    GWN --> Redis
    
    CPUTrigger --> GWN
    LatencyTrigger --> GWN
    ConnectionsTrigger --> GWN
```

---

## Operational Considerations

### Health Monitoring

```yaml
# Health Check Endpoints
health_checks:
  # Basic health check
  - path: "/health"
    method: "GET"
    response: {"status": "healthy", "timestamp": "2025-06-12T10:30:00Z"}
  
  # Detailed health check
  - path: "/health/detailed"
    method: "GET"
    response:
      status: "healthy"
      components:
        - name: "http_server"
          status: "healthy"
          latency_ms: 1.2
        - name: "redis_connection"
          status: "healthy"
          latency_ms: 2.1
        - name: "metald_backend"
          status: "healthy"
          latency_ms: 15.3
      
  # Readiness check
  - path: "/ready"
    method: "GET"
    response: {"ready": true, "dependencies": ["redis", "metald"]}
```

### Metrics Collection

```mermaid
graph TB
    subgraph "Request Metrics"
        RequestCount[Request Count<br/>- Total Requests<br/>- Per-Customer<br/>- Per-Endpoint]
        RequestLatency[Request Latency<br/>- P50, P95, P99<br/>- Per-Operation<br/>- Success/Error]
        RequestSize[Request Size<br/>- Payload Size<br/>- Response Size<br/>- Bandwidth Usage]
    end
    
    subgraph "Authentication Metrics"
        AuthSuccess[Auth Success Rate<br/>- JWT Validation<br/>- Customer Context<br/>- Token Types]
        AuthFailures[Auth Failures<br/>- Invalid Tokens<br/>- Expired Tokens<br/>- Missing Tokens]
        AuthLatency[Auth Latency<br/>- Token Validation<br/>- Context Building<br/>- Cache Hit Rate]
    end
    
    subgraph "Rate Limiting Metrics"
        RateLimitHits[Rate Limit Hits<br/>- Global Limits<br/>- Customer Limits<br/>- Endpoint Limits]
        RateLimitUtilization[Rate Limit Utilization<br/>- Percentage Used<br/>- Burst Usage<br/>- Quota Consumption]
    end
    
    subgraph "Backend Metrics"
        BackendLatency[Backend Latency<br/>- Metald Response Time<br/>- Connection Time<br/>- Queue Time]
        BackendErrors[Backend Errors<br/>- HTTP Errors<br/>- Timeouts<br/>- Circuit Breaker]
        BackendHealth[Backend Health<br/>- Instance Status<br/>- Response Rate<br/>- Error Rate]
    end
    
    RequestCount --> AuthSuccess
    RequestLatency --> AuthFailures
    RequestSize --> AuthLatency
    
    AuthSuccess --> RateLimitHits
    AuthFailures --> RateLimitUtilization
    AuthLatency --> RateLimitUtilization
    
    RateLimitHits --> BackendLatency
    RateLimitUtilization --> BackendErrors
    RateLimitUtilization --> BackendHealth
```

### Error Handling & Circuit Breaking

```mermaid
stateDiagram-v2
    [*] --> Closed : Normal Operation
    
    Closed --> Open : Error Threshold Exceeded
    Open --> HalfOpen : Timeout Expired
    HalfOpen --> Closed : Success Threshold Met
    HalfOpen --> Open : Failure Detected
    
    note right of Closed
        Requests flow normally
        Monitor error rate
        Track response times
    end note
    
    note right of Open
        Fail fast responses
        No backend calls
        Return cached/fallback
    end note
    
    note right of HalfOpen
        Limited requests allowed
        Test backend health
        Quick failure detection
    end note
```

### Configuration Management

```yaml
# Gateway Configuration
gateway:
  server:
    host: "0.0.0.0"
    port: 8080
    tls:
      enabled: true
      cert_file: "/etc/ssl/gateway.crt"
      key_file: "/etc/ssl/gateway.key"
    
    timeouts:
      read: "30s"
      write: "30s"
      idle: "120s"
    
    limits:
      max_header_size: "1MB"
      max_body_size: "10MB"
      max_connections: 10000
  
  authentication:
    jwt:
      public_key_file: "/etc/keys/jwt-public.pem"
      algorithm: "RS256"
      cache_ttl: "5m"
    
    development:
      enabled: false
      token_prefix: "dev_customer_"
  
  rate_limiting:
    backend: "redis"
    redis:
      address: "redis-cluster:6379"
      password: ""
      db: 0
      pool_size: 100
  
  routing:
    strategy: "consistent_hash"
    health_check_interval: "30s"
    backend_timeout: "30s"
    retry_attempts: 3
    retry_backoff: "100ms"
  
  observability:
    metrics:
      enabled: true
      port: 9090
      path: "/metrics"
    
    tracing:
      enabled: true
      endpoint: "http://jaeger:14268/api/traces"
      sample_rate: 0.1
    
    logging:
      level: "info"
      format: "json"
      output: "stdout"
```

---

## Cross-References

### Architecture Documentation
- **[System Architecture Overview](../overview.md)** - Complete system design
- **[Metald Architecture](metald.md)** - Backend service integration
- **[Security Architecture](../security/overview.md)** - Security design

### API Documentation
- **[API Reference](../../api/reference.md)** - API endpoints and usage
- **[Configuration Guide](../../api/configuration.md)** - Configuration options

### Operational Documentation
- **[Production Deployment](../../deployment/production.md)** - Deployment procedures
- **[Security Hardening](../../deployment/security-hardening.md)** - Security configuration
- **[Monitoring Setup](../../deployment/monitoring-setup.md)** - Observability setup

### Development Documentation
- **[Testing Guide](../../development/testing/stress-testing.md)** - Load testing procedures
- **[Contribution Guide](../../development/contribution-guide.md)** - Development setup

---

*Last updated: 2025-06-12 | Next review: Gateway Design Review*