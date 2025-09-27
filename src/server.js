const express = require('express');
const cors = require('cors');
const helmet = require('helmet');
const compression = require('compression');
const { Storage } = require('@google-cloud/storage');
const winston = require('winston');
const mime = require('mime-types');

const app = express();
const PORT = process.env.PORT || 8080;

// Configure logging
const logger = winston.createLogger({
  level: 'info',
  format: winston.format.combine(
    winston.format.timestamp(),
    winston.format.json()
  ),
  transports: [
    new winston.transports.Console()
  ]
});

// Security middleware
app.use(helmet());
app.use(compression());
app.use(cors({
  origin: '*', // Configure this based on your needs
  credentials: false
}));

// Initialize Google Cloud Storage
const storage = new Storage({
  projectId: process.env.GOOGLE_CLOUD_PROJECT_ID,
  keyFilename: process.env.GOOGLE_CLOUD_KEY_FILE
});

// Health check endpoint
app.get('/health', (req, res) => {
  res.status(200).json({ 
    status: 'healthy', 
    timestamp: new Date().toISOString(),
    version: process.env.npm_package_version || '1.0.0'
  });
});

// Main proxy endpoint to serve files from GCS
app.get('/*', async (req, res) => {
  const startTime = Date.now();
  const requestPath = req.path;
  
  // Remove leading slash
  const objectName = requestPath.startsWith('/') ? requestPath.slice(1) : requestPath;
  
  // Default to index.html if path ends with /
  const finalObjectName = objectName === '' || objectName.endsWith('/') 
    ? `${objectName}index.html`.replace('//', '/') 
    : objectName;

  const bucketName = process.env.GCS_BUCKET_NAME;
  
  if (!bucketName) {
    logger.error('GCS_BUCKET_NAME environment variable not set');
    return res.status(500).json({ error: 'Server configuration error' });
  }

  try {
    logger.info(`Proxying request for: ${finalObjectName} from bucket: ${bucketName}`);
    
    const bucket = storage.bucket(bucketName);
    const file = bucket.file(finalObjectName);
    
    // Check if file exists and get metadata
    const [exists] = await file.exists();
    if (!exists) {
      logger.warn(`File not found: ${finalObjectName}`);
      return res.status(404).json({ 
        error: 'File not found',
        path: finalObjectName 
      });
    }

    // Get file metadata
    const [metadata] = await file.getMetadata();
    
    // Set appropriate headers for CDN caching
    const contentType = mime.lookup(finalObjectName) || 'application/octet-stream';
    
    res.set({
      'Content-Type': contentType,
      'Cache-Control': 'public, max-age=31536000, immutable', // 1 year cache
      'ETag': metadata.etag,
      'Last-Modified': metadata.updated,
      'Content-Length': metadata.size,
      'X-Proxy-Cache': 'MISS', // Cloudflare will override this
      'X-GCS-Object': finalObjectName,
      'X-Response-Time': `${Date.now() - startTime}ms`
    });

    // Handle conditional requests (304 Not Modified)
    const ifNoneMatch = req.get('If-None-Match');
    const ifModifiedSince = req.get('If-Modified-Since');
    
    if (ifNoneMatch === metadata.etag) {
      logger.info(`304 Not Modified for: ${finalObjectName}`);
      return res.status(304).end();
    }
    
    if (ifModifiedSince && new Date(ifModifiedSince) >= new Date(metadata.updated)) {
      logger.info(`304 Not Modified (date) for: ${finalObjectName}`);
      return res.status(304).end();
    }

    // Stream the file content
    const stream = file.createReadStream();
    
    stream.on('error', (error) => {
      logger.error(`Stream error for ${finalObjectName}:`, error);
      if (!res.headersSent) {
        res.status(500).json({ error: 'Failed to stream file' });
      }
    });

    stream.on('end', () => {
      logger.info(`Successfully served: ${finalObjectName} (${metadata.size} bytes) in ${Date.now() - startTime}ms`);
    });

    // Pipe the file stream to the response
    stream.pipe(res);
    
  } catch (error) {
    logger.error(`Error serving ${finalObjectName}:`, {
      error: error.message,
      stack: error.stack,
      bucket: bucketName
    });
    
    if (!res.headersSent) {
      res.status(500).json({ 
        error: 'Internal server error',
        message: process.env.NODE_ENV === 'development' ? error.message : 'Server error'
      });
    }
  }
});

// Error handling middleware
app.use((error, req, res, next) => {
  logger.error('Unhandled error:', {
    error: error.message,
    stack: error.stack,
    url: req.url,
    method: req.method
  });
  
  if (!res.headersSent) {
    res.status(500).json({ error: 'Internal server error' });
  }
});

// 404 handler
app.use((req, res) => {
  logger.warn(`404 Not Found: ${req.method} ${req.url}`);
  res.status(404).json({ 
    error: 'Not found',
    path: req.path 
  });
});

// Graceful shutdown
process.on('SIGTERM', () => {
  logger.info('SIGTERM received, shutting down gracefully');
  process.exit(0);
});

process.on('SIGINT', () => {
  logger.info('SIGINT received, shutting down gracefully');
  process.exit(0);
});

// Start server
app.listen(PORT, () => {
  logger.info(`GCS Proxy server running on port ${PORT}`);
  logger.info(`Environment: ${process.env.NODE_ENV || 'development'}`);
  logger.info(`GCS Bucket: ${process.env.GCS_BUCKET_NAME || 'NOT_SET'}`);
});

module.exports = app;