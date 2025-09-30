# Cloud Run Configuration for Go GCS Proxy (Low Traffic)

## 🎯 **Optimized for Low Traffic + Cost Efficiency**

### **service.yaml - Go Version**
```yaml
apiVersion: serving.knative.dev/v1
kind: Service
metadata:
  name: gcs-proxy-go
  annotations:
    run.googleapis.com/ingress: all
    run.googleapis.com/execution-environment: gen2
spec:
  template:
    metadata:
      annotations:
        # Ultra aggressive scaling for cost optimization
        autoscaling.knative.dev/maxScale: "10"      # Low max for cost control
        autoscaling.knative.dev/minScale: "0"       # Scale to zero when idle
        autoscaling.knative.dev/scaleDownDelay: "30s"  # Quick scale down
        
        # CPU allocation
        run.googleapis.com/cpu-throttling: "true"   # Throttle CPU when idle
        run.googleapis.com/startup-cpu-boost: "true" # Boost during cold start
        
    spec:
      # High concurrency since Go handles it well
      containerConcurrency: 1000                    # Go can handle many concurrent requests
      timeoutSeconds: 60                            # Short timeout for cost efficiency
      
      containers:
      - image: gcr.io/PROJECT_ID/gcs-proxy-go:latest
        ports:
        - containerPort: 8080
          name: http1
        
        env:
        - name: GOOGLE_CLOUD_PROJECT_ID
          value: "PROJECT_ID"
        - name: GCS_BUCKET_NAME
          value: "BUCKET_NAME"
        - name: PORT
          value: "8080"
        - name: GIN_MODE
          value: "release"                          # Production mode
        
        resources:
          limits:
            # Minimal resources for Go efficiency
            cpu: "0.5"                              # 0.5 vCPU sufficient for low traffic
            memory: "128Mi"                         # Go uses very little memory
          requests:
            cpu: "0.1"                              # Minimal baseline
            memory: "64Mi"                          # Go can run in 64MB
        
        # Health checks optimized for Go speed
        livenessProbe:
          httpGet:
            path: /health
            port: 8080
          initialDelaySeconds: 2                    # Go starts fast
          periodSeconds: 30                         # Less frequent checks
          timeoutSeconds: 2
          failureThreshold: 3
          
        readinessProbe:
          httpGet:
            path: /health
            port: 8080
          initialDelaySeconds: 1                    # Go ready very quickly  
          periodSeconds: 10
          timeoutSeconds: 2
          
  traffic:
  - percent: 100
    latestRevision: true
```

### **cloudbuild.yaml - Go Version**
```yaml
steps:
  # Build Go binary
  - name: 'golang:1.21-alpine'
    env: 
    - 'CGO_ENABLED=0'
    - 'GOOS=linux'
    - 'GOARCH=amd64'
    script: |
      go mod tidy
      go build -ldflags="-w -s" -o gcs-proxy .

  # Build minimal Docker image
  - name: 'gcr.io/cloud-builders/docker'
    args: [
      'build',
      '-t', 'gcr.io/$PROJECT_ID/gcs-proxy-go:$COMMIT_SHA',
      '-t', 'gcr.io/$PROJECT_ID/gcs-proxy-go:latest',
      '-f', 'Dockerfile.go',
      '.'
    ]

  # Push image
  - name: 'gcr.io/cloud-builders/docker'
    args: ['push', 'gcr.io/$PROJECT_ID/gcs-proxy-go:$COMMIT_SHA']

  - name: 'gcr.io/cloud-builders/docker'
    args: ['push', 'gcr.io/$PROJECT_ID/gcs-proxy-go:latest']

  # Deploy to Cloud Run
  - name: 'gcr.io/google.com/cloudsdktool/cloud-sdk'
    entrypoint: gcloud
    args: [
      'run', 'deploy', 'gcs-proxy-go',
      '--image', 'gcr.io/$PROJECT_ID/gcs-proxy-go:$COMMIT_SHA',
      '--region', 'us-central1',
      '--platform', 'managed',
      '--allow-unauthenticated',
      '--set-env-vars', 'GOOGLE_CLOUD_PROJECT_ID=$PROJECT_ID,GCS_BUCKET_NAME=${_BUCKET_NAME},GIN_MODE=release',
      '--memory', '128Mi',                          # Minimal memory
      '--cpu', '0.5',                               # Half vCPU
      '--concurrency', '1000',                      # High concurrency
      '--max-instances', '10',                      # Cost control
      '--min-instances', '0',                       # Scale to zero
      '--timeout', '60s'                            # Short timeout
    ]

substitutions:
  _BUCKET_NAME: 'your-bucket-name'

options:
  logging: CLOUD_LOGGING_ONLY
```

### **Dockerfile.go - Ultra Minimal**
```dockerfile
# Multi-stage build for minimal size
FROM golang:1.21-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags='-w -s -extldflags "-static"' \
    -a -installsuffix cgo \
    -o gcs-proxy .

# Final stage - scratch for minimal size
FROM gcr.io/distroless/static-debian11

# Copy CA certificates for HTTPS
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

# Copy binary
COPY --from=builder /app/gcs-proxy /gcs-proxy

# Non-root user
USER 1001

EXPOSE 8080

ENTRYPOINT ["/gcs-proxy"]
```

## 💰 **Cost Optimization Analysis**

### **Monthly Cost Estimate (Low Traffic)**

#### **Go Version:**
```
Assumptions: 
- 1,000 requests/day (30K/month)
- Average 100ms response time
- 95% cache hit ratio at Cloudflare
- 50 actual backend requests/day

Resources:
- CPU: 0.1 vCPU baseline, 0.5 vCPU max
- Memory: 64Mi baseline, 128Mi max
- Instances: 0 min, 10 max

Cost Calculation:
- vCPU-seconds: 50 requests × 0.1s × 0.5 vCPU × 30 days = 75 vCPU-seconds
- Memory: 75 × 128Mi = ~0.01 GB-seconds  
- Requests: 1,500 requests/month

Monthly Cost: ~$0.50 - $2.00 USD
```

#### **Node.js Version (comparison):**
```
Same traffic, but:
- Memory: 256Mi baseline, 512Mi max
- Slower cold starts = more CPU time

Monthly Cost: ~$2.00 - $5.00 USD
```

### **Performance Benefits:**

| Metric | Go | Node.js | Improvement |
|--------|-----|---------|-------------|
| **Image Size** | ~8MB | ~140MB | **17x smaller** |
| **Cold Start** | ~200ms | ~2000ms | **10x faster** |
| **Memory Usage** | ~10MB | ~60MB | **6x less** |
| **Concurrent Requests** | 1000+ | ~80 | **12x more** |
| **Monthly Cost** | ~$1 | ~$3 | **3x cheaper** |

## ⚙️ **Deployment Commands**

### **Deploy with minimal config:**
```bash
# Set environment
export GOOGLE_CLOUD_PROJECT="your-project"
export GCS_BUCKET_NAME="your-bucket"

# Deploy minimal instance
gcloud run deploy gcs-proxy-go \
  --source . \
  --region us-central1 \
  --allow-unauthenticated \
  --memory 128Mi \
  --cpu 0.5 \
  --concurrency 1000 \
  --max-instances 10 \
  --min-instances 0 \
  --timeout 60s \
  --set-env-vars="GCS_BUCKET_NAME=${GCS_BUCKET_NAME},GIN_MODE=release"
```

### **Update scaling (if traffic increases):**
```bash
gcloud run services update gcs-proxy-go \
  --region us-central1 \
  --max-instances 50 \
  --memory 256Mi \
  --cpu 1
```

## 📊 **Monitoring & Alerts**

### **Key metrics to watch:**
```yaml
# Cloud Monitoring alerts
alerts:
  - name: "High Error Rate"
    condition: "error_rate > 5%"
    
  - name: "High Latency" 
    condition: "latency_p99 > 500ms"
    
  - name: "Cost Spike"
    condition: "monthly_cost > $10"
```

## 🎯 **Summary for Low Traffic Go Setup:**

### **Optimal Configuration:**
- **Memory**: 128Mi (Go efficiency)
- **CPU**: 0.5 vCPU (sufficient for low traffic)
- **Concurrency**: 1000 (Go handles it well)
- **Min instances**: 0 (cost optimization)
- **Max instances**: 10 (traffic control)
- **Timeout**: 60s (quick responses)

### **Expected Performance:**
- **Cold start**: <300ms
- **Response time**: <50ms (cached), <200ms (GCS fetch)
- **Monthly cost**: <$2 for low traffic
- **Scale**: Handle traffic spikes up to 10,000 concurrent users

**Perfect for low-traffic scenarios with maximum cost efficiency!** 🚀💰