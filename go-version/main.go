package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"cloud.google.com/go/storage"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"google.golang.org/api/option"
)

var (
	storageClient *storage.Client
	bucketName    string
	logger        *logrus.Logger
)

// HealthResponse represents the health check response
type HealthResponse struct {
	Status    string `json:"status"`
	Timestamp string `json:"timestamp"`
	Version   string `json:"version"`
}

// ErrorResponse represents an error response
type ErrorResponse struct {
	Error   string `json:"error"`
	Path    string `json:"path,omitempty"`
	Message string `json:"message,omitempty"`
}

func init() {
	// Initialize logger
	logger = logrus.New()
	logger.SetFormatter(&logrus.JSONFormatter{})
	logger.SetLevel(logrus.InfoLevel)

	// Set Gin mode from environment
	ginMode := os.Getenv("GIN_MODE")
	if ginMode == "" {
		ginMode = "release"
	}
	gin.SetMode(ginMode)
}

func initializeGCS() error {
	ctx := context.Background()
	
	var err error
	
	// Get project ID and bucket name from environment
	projectID := os.Getenv("GOOGLE_CLOUD_PROJECT_ID")
	bucketName = os.Getenv("GCS_BUCKET_NAME")
	keyFile := os.Getenv("GOOGLE_CLOUD_KEY_FILE")

	if bucketName == "" {
		bucketName = "test-bucket" // Default for testing
		logger.Warn("GCS_BUCKET_NAME not set, using test-bucket")
	}

	// Skip GCS initialization if no project ID (test mode)
	if projectID == "" {
		logger.Info("No GOOGLE_CLOUD_PROJECT_ID set - running in test mode")
		return nil
	}

	// Initialize GCS client
	if keyFile != "" {
		// Use service account key file
		storageClient, err = storage.NewClient(ctx, option.WithCredentialsFile(keyFile))
	} else {
		// Use Application Default Credentials
		storageClient, err = storage.NewClient(ctx)
	}

	if err != nil {
		return fmt.Errorf("failed to create storage client: %v", err)
	}

	logger.WithFields(logrus.Fields{
		"project_id":  projectID,
		"bucket_name": bucketName,
		"auth_method": func() string {
			if keyFile != "" {
				return "service_account_key"
			}
			return "application_default_credentials"
		}(),
	}).Info("GCS client initialized successfully")

	return nil
}

func healthHandler(c *gin.Context) {
	response := HealthResponse{
		Status:    "healthy",
		Timestamp: time.Now().Format(time.RFC3339),
		Version:   "1.0.0",
	}
	c.JSON(http.StatusOK, response)
}

func corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Accept, Authorization")
		
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusOK)
			return
		}
		
		c.Next()
	}
}

func securityMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Security headers (similar to Helmet.js)
		c.Header("Content-Security-Policy", "default-src 'self';base-uri 'self';font-src 'self' https: data:;form-action 'self';frame-ancestors 'self';img-src 'self' data:;object-src 'none';script-src 'self';script-src-attr 'none';style-src 'self' https: 'unsafe-inline';upgrade-insecure-requests")
		c.Header("Cross-Origin-Opener-Policy", "same-origin")
		c.Header("Cross-Origin-Resource-Policy", "same-origin")
		c.Header("Origin-Agent-Cluster", "?1")
		c.Header("Referrer-Policy", "no-referrer")
		c.Header("Strict-Transport-Security", "max-age=15552000; includeSubDomains")
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("X-DNS-Prefetch-Control", "off")
		c.Header("X-Download-Options", "noopen")
		c.Header("X-Frame-Options", "SAMEORIGIN")
		c.Header("X-Permitted-Cross-Domain-Policies", "none")
		c.Header("X-XSS-Protection", "0")
		
		c.Next()
	}
}

func loggingMiddleware() gin.HandlerFunc {
	return gin.LoggerWithFormatter(func(param gin.LogFormatterParams) string {
		logger.WithFields(logrus.Fields{
			"method":     param.Method,
			"path":       param.Path,
			"status":     param.StatusCode,
			"latency":    param.Latency.String(),
			"client_ip":  param.ClientIP,
			"user_agent": param.Request.UserAgent(),
			"error":      param.ErrorMessage,
		}).Info("Request processed")
		
		return ""
	})
}

func getContentType(filename string) string {
	ext := strings.ToLower(filepath.Ext(filename))
	
	contentTypes := map[string]string{
		".html": "text/html; charset=utf-8",
		".css":  "text/css; charset=utf-8",
		".js":   "application/javascript; charset=utf-8",
		".json": "application/json; charset=utf-8",
		".xml":  "application/xml; charset=utf-8",
		".jpg":  "image/jpeg",
		".jpeg": "image/jpeg",
		".png":  "image/png",
		".gif":  "image/gif",
		".svg":  "image/svg+xml",
		".webp": "image/webp",
		".pdf":  "application/pdf",
		".txt":  "text/plain; charset=utf-8",
		".ico":  "image/x-icon",
	}
	
	if contentType, exists := contentTypes[ext]; exists {
		return contentType
	}
	
	return "application/octet-stream"
}

func getCacheControl(filename string) string {
	ext := strings.ToLower(filepath.Ext(filename))
	
	switch ext {
	case ".html":
		return "public, max-age=300" // 5 minutes for HTML
	case ".js", ".css":
		return "public, max-age=31536000, immutable" // 1 year for static assets
	case ".jpg", ".jpeg", ".png", ".gif", ".webp", ".svg", ".ico":
		return "public, max-age=2592000" // 30 days for images
	case ".pdf", ".txt":
		return "public, max-age=86400" // 1 day for documents
	default:
		return "public, max-age=31536000, immutable" // 1 year default
	}
}

func proxyHandler(c *gin.Context) {
	// Check if GCS client is available
	if storageClient == nil {
		logger.Warn("GCS client not available - running in test mode")
		c.JSON(http.StatusServiceUnavailable, ErrorResponse{
			Error:   "Service unavailable",
			Message: "GCS client not configured",
		})
		return
	}

	startTime := time.Now()
	requestPath := c.Request.URL.Path
	
	// Remove leading slash and handle root path
	if requestPath == "/" || requestPath == "" {
		requestPath = "index.html"
	} else if strings.HasPrefix(requestPath, "/") {
		requestPath = requestPath[1:] // Remove leading slash
	}
	if strings.HasSuffix(requestPath, "/") {
		requestPath = requestPath + "index.html"
	}
	
	// Remove leading slash if present
	if strings.HasPrefix(requestPath, "/") {
		requestPath = requestPath[1:]
	}
	
	logger.WithFields(logrus.Fields{
		"object_name": requestPath,
		"bucket":      bucketName,
	}).Info("Proxying request")
	
	ctx := context.Background()
	
	// Get object handle
	bucket := storageClient.Bucket(bucketName)
	obj := bucket.Object(requestPath)
	
	// Check if object exists and get attributes
	attrs, err := obj.Attrs(ctx)
	if err != nil {
		if err == storage.ErrObjectNotExist {
			logger.WithFields(logrus.Fields{
				"object_name": requestPath,
				"error":       "file not found",
			}).Warn("File not found")
			
			c.JSON(http.StatusNotFound, ErrorResponse{
				Error: "File not found",
				Path:  requestPath,
			})
			return
		}
		
		logger.WithFields(logrus.Fields{
			"object_name": requestPath,
			"error":       err.Error(),
		}).Error("Failed to get object attributes")
		
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Internal server error",
			Message: "Failed to access file",
		})
		return
	}
	
	// Handle conditional requests
	ifNoneMatch := c.GetHeader("If-None-Match")
	ifModifiedSince := c.GetHeader("If-Modified-Since")
	
	if ifNoneMatch != "" && ifNoneMatch == attrs.Etag {
		logger.WithFields(logrus.Fields{
			"object_name": requestPath,
			"etag":        attrs.Etag,
		}).Info("304 Not Modified (ETag match)")
		
		c.Status(http.StatusNotModified)
		return
	}
	
	if ifModifiedSince != "" {
		if modTime, err := time.Parse(time.RFC1123, ifModifiedSince); err == nil {
			if !attrs.Updated.After(modTime) {
				logger.WithFields(logrus.Fields{
					"object_name":       requestPath,
					"if_modified_since": ifModifiedSince,
					"last_modified":     attrs.Updated.Format(time.RFC1123),
				}).Info("304 Not Modified (date)")
				
				c.Status(http.StatusNotModified)
				return
			}
		}
	}
	
	// Set response headers
	contentType := getContentType(requestPath)
	cacheControl := getCacheControl(requestPath)
	
	c.Header("Content-Type", contentType)
	c.Header("Cache-Control", cacheControl)
	c.Header("ETag", attrs.Etag)
	c.Header("Last-Modified", attrs.Updated.Format(time.RFC1123))
	c.Header("Content-Length", strconv.FormatInt(attrs.Size, 10))
	c.Header("X-Proxy-Cache", "MISS")
	c.Header("X-GCS-Object", requestPath)
	c.Header("X-Response-Time", time.Since(startTime).String())
	
	// Create reader and stream content
	reader, err := obj.NewReader(ctx)
	if err != nil {
		logger.WithFields(logrus.Fields{
			"object_name": requestPath,
			"error":       err.Error(),
		}).Error("Failed to create object reader")
		
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Internal server error",
			Message: "Failed to read file",
		})
		return
	}
	defer reader.Close()
	
	// Stream the content directly to the response
	c.Status(http.StatusOK)
	
	bytesWritten, err := io.Copy(c.Writer, reader)
	if err != nil {
		logger.WithFields(logrus.Fields{
			"object_name":    requestPath,
			"bytes_written":  bytesWritten,
			"error":          err.Error(),
		}).Error("Failed to stream object content")
		return
	}
	
	logger.WithFields(logrus.Fields{
		"object_name":   requestPath,
		"bytes_served":  bytesWritten,
		"response_time": time.Since(startTime).String(),
		"content_type":  contentType,
	}).Info("Successfully served object")
}

func setupRouter() *gin.Engine {
	r := gin.New()
	
	// Add middleware
	r.Use(gin.Recovery())
	r.Use(loggingMiddleware())
	r.Use(securityMiddleware())
	r.Use(corsMiddleware())
	
	// Routes
	r.GET("/health", healthHandler)
	// Catch-all route for file requests
	r.NoRoute(func(c *gin.Context) {
		proxyHandler(c)
	})
	
	return r
}

func main() {
	// Initialize GCS client
	if err := initializeGCS(); err != nil {
		logger.WithError(err).Warn("Failed to initialize GCS client - continuing in test mode")
	}
	
	// Only close client if it was successfully created
	if storageClient != nil {
		defer storageClient.Close()
	}
	
	// Get port from environment
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	
	// Setup router
	router := setupRouter()
	
	// Start server
	server := &http.Server{
		Addr:    ":" + port,
		Handler: router,
	}
	
	// Graceful shutdown
	go func() {
		sigterm := make(chan os.Signal, 1)
		signal.Notify(sigterm, syscall.SIGINT, syscall.SIGTERM)
		<-sigterm
		
		logger.Info("Shutting down server...")
		
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		
		if err := server.Shutdown(ctx); err != nil {
			logger.WithError(err).Error("Server shutdown error")
		} else {
			logger.Info("Server shutdown complete")
		}
	}()
	
	logger.WithFields(logrus.Fields{
		"port":        port,
		"environment": gin.Mode(),
		"bucket":      bucketName,
	}).Info("GCS Proxy server starting")
	
	// Start server
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		logger.WithError(err).Fatal("Server failed to start")
	}
}