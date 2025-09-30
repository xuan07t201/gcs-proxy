# Cloud Run Pricing Calculator - Chi tiết và Ví dụ thực tế

## 💰 **Cách tính tiền Cloud Run**

### **4 yếu tố chính:**
1. **CPU allocation** - Tính theo vCPU-seconds
2. **Memory allocation** - Tính theo GB-seconds  
3. **Request count** - Tính theo số lượng requests
4. **Network egress** - Data transfer ra ngoài GCP

### **Pricing (US pricing, Sept 2025):**
```
CPU: $0.00002400 per vCPU-second
Memory: $0.00000250 per GB-second
Requests: $0.40 per 1 million requests
Network egress: $0.12 per GB (to internet)
```

---

## 📊 **Trường hợp: 1 request/giây**

### **Assumptions:**
- **Traffic**: 1 request/giây = 2,592,000 requests/tháng (30 days)
- **Response time**: 200ms average (bao gồm cả cold start)
- **Instance allocation**: CPU + Memory được allocate trong suốt thời gian xử lý

---

## 🧮 **Scenario 1: Node.js Version**

### **Configuration:**
```yaml
resources:
  cpu: "1"          # 1 full vCPU
  memory: "512Mi"   # 0.5 GB
concurrency: 80
minInstances: 0
```

### **Calculation:**
```
Monthly stats:
- Total requests: 2,592,000
- Processing time: 200ms per request
- Total CPU-seconds: 2,592,000 × 0.2s × 1 vCPU = 518,400 vCPU-seconds
- Total memory-seconds: 2,592,000 × 0.2s × 0.5 GB = 259,200 GB-seconds
- Response size: ~10KB average (with headers)
- Egress: 2,592,000 × 10KB = ~25GB

Cost breakdown:
1. CPU: 518,400 × $0.000024 = $12.44
2. Memory: 259,200 × $0.0000025 = $0.65  
3. Requests: 2,592,000 × $0.40/1M = $1.04
4. Egress: 25GB × $0.12 = $3.00

Total: $17.13/month
```

---

## 🚀 **Scenario 2: Go Version (Optimized)**

### **Configuration:**
```yaml
resources:
  cpu: "0.5"        # Half vCPU (Go efficiency)
  memory: "128Mi"   # 0.125 GB
concurrency: 1000   # Higher concurrency
minInstances: 0
```

### **Calculation:**
```
Monthly stats:
- Total requests: 2,592,000
- Processing time: 100ms per request (Go faster)
- Total CPU-seconds: 2,592,000 × 0.1s × 0.5 vCPU = 129,600 vCPU-seconds
- Total memory-seconds: 2,592,000 × 0.1s × 0.125 GB = 32,400 GB-seconds
- Response size: ~8KB average (more efficient)
- Egress: 2,592,000 × 8KB = ~20GB

Cost breakdown:
1. CPU: 129,600 × $0.000024 = $3.11
2. Memory: 32,400 × $0.0000025 = $0.08
3. Requests: 2,592,000 × $0.40/1M = $1.04  
4. Egress: 20GB × $0.12 = $2.40

Total: $6.63/month
```

---

## ⚡ **Scenario 3: Go Version (Ultra Optimized)**

### **Configuration:**
```yaml
resources:
  cpu: "0.25"       # Quarter vCPU
  memory: "64Mi"    # 0.0625 GB  
concurrency: 2000  # Very high concurrency
minInstances: 0
```

### **Calculation:**
```
Monthly stats:
- Total requests: 2,592,000
- Processing time: 80ms per request (optimal Go)
- Total CPU-seconds: 2,592,000 × 0.08s × 0.25 vCPU = 51,840 vCPU-seconds
- Total memory-seconds: 2,592,000 × 0.08s × 0.0625 GB = 12,960 GB-seconds
- Response size: ~6KB average
- Egress: 2,592,000 × 6KB = ~15GB

Cost breakdown:
1. CPU: 51,840 × $0.000024 = $1.24
2. Memory: 12,960 × $0.0000025 = $0.03
3. Requests: 2,592,000 × $0.40/1M = $1.04
4. Egress: 15GB × $0.12 = $1.80

Total: $4.11/month
```

---

## 📈 **Scaling Analysis**

### **Cost theo traffic level:**

| Requests/second | Node.js | Go Optimized | Go Ultra | Savings |
|----------------|---------|--------------|----------|---------|
| **0.1** (8,640/month) | $0.57 | $0.22 | $0.14 | **75%** |
| **1** (2.6M/month) | $17.13 | $6.63 | $4.11 | **76%** |
| **10** (26M/month) | $171.30 | $66.30 | $41.10 | **76%** |
| **100** (260M/month) | $1,713 | $663 | $411 | **76%** |

---

## 💡 **Optimization Tips**

### **1. Reduce processing time:**
```go
// Faster Go code
func handler(c *gin.Context) {
    // Use connection pooling
    client := getPooledGCSClient()
    
    // Stream directly, no buffering
    reader, _ := obj.NewReader(ctx)
    defer reader.Close()
    
    // Set headers first
    c.Header("Cache-Control", "public, max-age=31536000")
    
    // Stream directly to response
    io.Copy(c.Writer, reader)
}
```

### **2. Optimize concurrency:**
```yaml
# Handle more requests per instance
containerConcurrency: 2000  # Go can handle this
minInstances: 0             # Scale to zero when idle
```

### **3. Memory optimization:**
```yaml
# Use minimal memory
memory: "64Mi"    # Go runs fine with 64MB
cpu: "0.25"       # Quarter vCPU sufficient
```

---

## 🔍 **Real-world Impact Analysis**

### **With Cloudflare CDN (95% cache hit rate):**
```
Actual backend requests: 1/sec × 5% = 0.05/sec
Monthly requests to Cloud Run: ~130,000

Go Ultra cost: $0.26/month
Node.js cost: $0.86/month

Savings: 70% cost reduction + much better performance!
```

### **Cold start impact:**
```
Node.js: 2s cold start × occasional = ~$0.50 extra/month
Go: 0.2s cold start × occasional = ~$0.05 extra/month

Additional savings: $0.45/month
```

---

## 📊 **Free Tier Considerations**

### **Cloud Run free tier (per month):**
- **CPU**: 180,000 vCPU-seconds
- **Memory**: 360,000 GB-seconds  
- **Requests**: 2 million requests

### **Will 1 req/sec fit in free tier?**

**Go Ultra version:**
- CPU usage: 51,840 vCPU-seconds ✅ (fits in 180k)
- Memory usage: 12,960 GB-seconds ✅ (fits in 360k)
- Requests: 2,592,000 ❌ (exceeds 2M)

**Cost with free tier:**
- CPU: $0 (within free tier)
- Memory: $0 (within free tier)  
- Requests: 592,000 excess × $0.40/1M = $0.24
- Egress: $1.80

**Total: $2.04/month (50% savings from free tier!)**

---

## 🎯 **Summary cho 1 request/giây:**

| Version | Monthly Cost | vs Node.js | Free Tier Compatible |
|---------|-------------|-----------|-------------------|
| **Node.js** | $17.13 | baseline | ❌ No |
| **Go Optimized** | $6.63 | -61% | ❌ No |  
| **Go Ultra** | $4.11 | -76% | Partially (CPU/Memory) |
| **Go + Free Tier** | $2.04 | -88% | ✅ Mostly |

**Kết luận: Go version có thể tiết kiệm 75-90% chi phí so với Node.js!** 🚀💰