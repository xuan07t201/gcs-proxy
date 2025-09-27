# GCS Proxy Authentication Setup Guide

## Phương thức xác thực với Google Cloud Storage

### 1. Sử dụng Service Account (Development)

1. Tạo Service Account trong GCP Console:
   - Vào IAM & Admin > Service Accounts
   - Tạo service account mới
   - Cấp quyền: Storage Object Viewer (để đọc files)

2. Tạo và download key file:
   - Tạo key cho service account (JSON format)
   - Download về và đặt tên `service-account-key.json`
   - Đặt file trong thư mục gốc dự án

3. Cấu hình biến môi trường:
   ```bash
   export GOOGLE_CLOUD_PROJECT_ID="your-project-id"
   export GCS_BUCKET_NAME="your-bucket-name"
   export GOOGLE_CLOUD_KEY_FILE="./service-account-key.json"
   ```

### 2. Sử dụng Application Default Credentials (Production/Cloud Run)

Khi deploy lên Cloud Run, bạn có thể:

1. Sử dụng default service account của Cloud Run
2. Hoặc gán một service account cụ thể cho Cloud Run service

Trong trường hợp này, không cần `GOOGLE_CLOUD_KEY_FILE`, ADC sẽ tự động được sử dụng.

## Required GCS Permissions

Service account cần có ít nhất các quyền sau:
- `storage.objects.get` - Đọc object
- `storage.objects.list` - List objects (nếu cần)
- `storage.buckets.get` - Thông tin bucket

Hoặc có thể sử dụng role có sẵn:
- `Storage Object Viewer` (recommended cho read-only)
- `Storage Legacy Bucket Reader`

## Bucket Configuration

Đảm bảo GCS bucket được cấu hình đúng:
- Enable uniform bucket-level access (recommended)
- Configure CORS nếu cần thiết
- Set appropriate lifecycle policies cho caching