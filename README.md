# GCS Proxy for Cloudflare CDN

Một proxy server Node.js để phục vụ files từ Google Cloud Storage (GCS) làm backend cho Cloudflare CDN. Được thiết kế để chạy trên Google Cloud Run với hiệu suất cao và tối ưu hóa cho caching.

## Tính năng

- ✅ **Proxy GCS Files**: Serve files trực tiếp từ GCS buckets
- ✅ **CDN Optimized**: Headers tối ưu cho Cloudflare caching (Cache-Control, ETag, Last-Modified)
- ✅ **Conditional Requests**: Hỗ trợ 304 Not Modified để giảm bandwidth
- ✅ **Streaming**: Streaming files để xử lý files lớn hiệu quả
- ✅ **Health Checks**: Built-in health check endpoint
- ✅ **Structured Logging**: Winston logging cho monitoring và debugging
- ✅ **Security**: Helmet.js và CORS configuration
- ✅ **Compression**: Gzip compression cho responses
- ✅ **Cloud Run Ready**: Dockerfile và deployment configs sẵn sàng

## Cấu trúc thư mục

```
gcs-proxy/
├── src/
│   └── server.js          # Main Express server
├── scripts/
│   └── deploy.sh          # Deployment script
├── package.json           # Dependencies và scripts
├── Dockerfile             # Container configuration
├── .dockerignore          # Docker ignore patterns  
├── cloudbuild.yaml        # Cloud Build configuration
├── service.yaml           # Cloud Run service definition
├── .env.example           # Environment variables template
├── AUTHENTICATION.md      # Authentication setup guide
└── README.md             # This file
```

## Cài đặt và Setup

### 1. Clone và Install Dependencies

```bash
git clone <repository-url>
cd gcs-proxy
npm install
```

### 2. Cấu hình Environment Variables

Copy `.env.example` thành `.env` và cấu hình:

```bash
cp .env.example .env
```

Chỉnh sửa `.env`:

```env
GOOGLE_CLOUD_PROJECT_ID=your-gcp-project-id
GCS_BUCKET_NAME=your-bucket-name
GOOGLE_CLOUD_KEY_FILE=./service-account-key.json  # Cho development
NODE_ENV=development
PORT=8080
```

### 3. Setup Authentication

Tham khảo [AUTHENTICATION.md](./AUTHENTICATION.md) để setup authentication với GCS.

### 4. Chạy Development Server

```bash
npm run dev
```

Server sẽ chạy tại `http://localhost:8080`

## Deployment lên Google Cloud Run

### Tự động với Cloud Build

```bash
# Set environment variables
export GOOGLE_CLOUD_PROJECT="your-project-id"
export GCS_BUCKET_NAME="your-bucket-name"

# Deploy
./scripts/deploy.sh
```

### Manual Deployment

```bash
# Build Docker image
docker build -t gcr.io/your-project/gcs-proxy .

# Push to Container Registry
docker push gcr.io/your-project/gcs-proxy

# Deploy to Cloud Run
gcloud run deploy gcs-proxy \\
  --image gcr.io/your-project/gcs-proxy \\
  --region us-central1 \\
  --allow-unauthenticated \\
  --set-env-vars GOOGLE_CLOUD_PROJECT_ID=your-project,GCS_BUCKET_NAME=your-bucket
```

## Usage

### API Endpoints

#### `GET /health`
Health check endpoint
```json
{
  "status": "healthy",
  "timestamp": "2024-01-01T00:00:00.000Z",
  "version": "1.0.0"
}
```

#### `GET /*`
Proxy endpoint để serve files từ GCS

**Examples:**
- `GET /images/logo.png` → Serves `gs://your-bucket/images/logo.png`
- `GET /css/style.css` → Serves `gs://your-bucket/css/style.css`  
- `GET /` → Serves `gs://your-bucket/index.html`
- `GET /folder/` → Serves `gs://your-bucket/folder/index.html`

**Response Headers:**
- `Content-Type`: Auto-detected từ file extension
- `Cache-Control`: `public, max-age=31536000, immutable`
- `ETag`: GCS object ETag
- `Last-Modified`: Object update time
- `X-GCS-Object`: Object path trong bucket
- `X-Response-Time`: Request processing time

## Cấu hình Cloudflare

### 1. DNS Setup
Point domain hoặc subdomain tới Cloud Run service URL:

```
Type: CNAME
Name: cdn (hoặc subdomain bạn muốn)
Content: your-cloud-run-url
```

### 2. Cloudflare Cache Settings
Recommended cache settings:

```
Cache Level: Standard
Browser Cache TTL: 1 year
Edge Cache TTL: 1 month
```

### 3. Page Rules (Optional)
Tạo page rule để tối ưu caching:

```
URL: cdn.yourdomain.com/*
Settings:
  - Cache Level: Cache Everything
  - Edge Cache TTL: 1 month
```

## Monitoring và Logging

### Cloud Logging
Logs được gửi tự động tới Google Cloud Logging khi chạy trên Cloud Run.

### Metrics
Monitor các metrics sau:
- Request latency
- Error rates  
- Memory/CPU usage
- Cache hit rates (từ Cloudflare)

### Alerts
Setup alerts cho:
- High error rates (>5%)
- High latency (>2s)
- Service downtime

## Performance Tuning

### Cloud Run Configuration
```yaml
resources:
  limits:
    cpu: "1"
    memory: "512Mi"
  requests:
    cpu: "1" 
    memory: "512Mi"
containerConcurrency: 80
```

### GCS Optimization
- Enable uniform bucket-level access
- Use regional buckets gần với Cloud Run region
- Consider Coldline/Archive storage cho ít accessed files

## Troubleshooting

### Common Issues

**1. Authentication Errors**
```
Error: Could not load the default credentials
```
→ Kiểm tra service account setup trong [AUTHENTICATION.md](./AUTHENTICATION.md)

**2. File Not Found**
```
{
  "error": "File not found", 
  "path": "missing-file.jpg"
}
```
→ Verify file tồn tại trong GCS bucket với đúng path

**3. Permission Denied**
```
Error: Insufficient permissions
```
→ Ensure service account có `Storage Object Viewer` role

### Debug Logs

Enable debug logging:
```bash
export NODE_ENV=development
npm start
```

## Security Considerations

- Service chỉ có quyền read-only tới GCS
- Sử dụng IAM roles thay vì service account keys trong production  
- Enable HTTPS-only trong Cloud Run
- Configure CORS phù hợp với use case

## Contributing

1. Fork repository
2. Create feature branch
3. Make changes
4. Test thoroughly
5. Submit pull request

## License

MIT License - see LICENSE file for details.