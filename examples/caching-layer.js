// Ví dụ Memory Cache Layer cho GCS Proxy
const NodeCache = require('node-cache');

// Cache metadata để tránh call GCS API nhiều lần
const metadataCache = new NodeCache({ 
  stdTTL: 300, // 5 minutes
  checkperiod: 60 // cleanup every 60s
});

// Cache hot files content (cho files nhỏ)
const contentCache = new NodeCache({ 
  stdTTL: 600, // 10 minutes  
  maxKeys: 1000 // limit memory usage
});

// Usage trong proxy endpoint:
app.get('/*', async (req, res) => {
  const cacheKey = `metadata:${finalObjectName}`;
  
  // Check metadata cache first
  let metadata = metadataCache.get(cacheKey);
  
  if (!metadata) {
    // Cache miss - get from GCS
    [metadata] = await file.getMetadata();
    metadataCache.set(cacheKey, metadata);
    logger.info(`Metadata cache MISS for: ${finalObjectName}`);
  } else {
    logger.info(`Metadata cache HIT for: ${finalObjectName}`);
  }
  
  // For small files, cache content too
  if (metadata.size < 1024 * 1024) { // < 1MB
    const contentKey = `content:${finalObjectName}`;
    let content = contentCache.get(contentKey);
    
    if (!content) {
      const [buffer] = await file.download();
      contentCache.set(contentKey, buffer);
      res.send(buffer);
    } else {
      res.send(content); // Serve from memory - VERY fast!
    }
  } else {
    // Stream large files as usual
    file.createReadStream().pipe(res);
  }
});