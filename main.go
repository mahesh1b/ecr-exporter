package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ecr"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sirupsen/logrus"
)

const (
	namespace = "ecr"
	port      = ":8080"
)

var (
	log       = logrus.New()
	startTime = time.Now()
)

type HealthStatus struct {
	Status    string            `json:"status"`
	Uptime    string            `json:"uptime"`
	Memory    MemoryStats       `json:"memory"`
	CPU       CPUStats          `json:"cpu"`
	Goroutines int              `json:"goroutines"`
	Timestamp string            `json:"timestamp"`
}

type MemoryStats struct {
	AllocMB      float64 `json:"alloc_mb"`
	TotalAllocMB float64 `json:"total_alloc_mb"`
	SysMB        float64 `json:"sys_mb"`
	NumGC        uint32  `json:"num_gc"`
}

type CPUStats struct {
	NumCPU       int `json:"num_cpu"`
	NumGoroutine int `json:"num_goroutine"`
}

func getHealthStatus() HealthStatus {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	uptime := time.Since(startTime)

	return HealthStatus{
		Status:     "OK",
		Uptime:     uptime.String(),
		Goroutines: runtime.NumGoroutine(),
		Timestamp:  time.Now().Format(time.RFC3339),
		Memory: MemoryStats{
			AllocMB:      float64(m.Alloc) / 1024 / 1024,
			TotalAllocMB: float64(m.TotalAlloc) / 1024 / 1024,
			SysMB:        float64(m.Sys) / 1024 / 1024,
			NumGC:        m.NumGC,
		},
		CPU: CPUStats{
			NumCPU:       runtime.NumCPU(),
			NumGoroutine: runtime.NumGoroutine(),
		},
	}
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	health := getHealthStatus()

	// Check if client wants JSON or HTML
	acceptHeader := r.Header.Get("Accept")
	if acceptHeader == "application/json" || r.URL.Query().Get("format") == "json" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(health)
	} else {
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)
		
		html := fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head>
    <title>ECR Exporter Health</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 40px; }
        .status { color: green; font-weight: bold; }
        .metric { margin: 10px 0; }
        .value { font-weight: bold; color: #333; }
        table { border-collapse: collapse; width: 100%%; max-width: 600px; }
        th, td { border: 1px solid #ddd; padding: 12px; text-align: left; }
        th { background-color: #f2f2f2; }
        .refresh { margin-top: 20px; }
    </style>
    <script>
        function refreshPage() { location.reload(); }
        setInterval(refreshPage, 30000); // Auto-refresh every 30 seconds
    </script>
</head>
<body>
    <h1>ECR Prometheus Exporter Health Status</h1>
    <p class="status">Status: %s</p>
    <p>Last Updated: %s</p>
    
    <h2>System Metrics</h2>
    <table>
        <tr><th>Metric</th><th>Value</th></tr>
        <tr><td>Uptime</td><td class="value">%s</td></tr>
        <tr><td>Goroutines</td><td class="value">%d</td></tr>
        <tr><td>CPU Cores</td><td class="value">%d</td></tr>
        <tr><td>Memory Allocated</td><td class="value">%.2f MB</td></tr>
        <tr><td>Total Memory Allocated</td><td class="value">%.2f MB</td></tr>
        <tr><td>System Memory</td><td class="value">%.2f MB</td></tr>
        <tr><td>GC Runs</td><td class="value">%d</td></tr>
    </table>
    
    <div class="refresh">
        <button onclick="refreshPage()">Refresh Now</button>
        <span style="margin-left: 20px;">Auto-refresh: 30s</span>
    </div>
    
    <p style="margin-top: 30px;">
        <a href="/">← Back to Home</a> | 
        <a href="/metrics">View Metrics</a> | 
        <a href="/health?format=json">JSON Format</a>
    </p>
</body>
</html>`,
			health.Status,
			health.Timestamp,
			health.Uptime,
			health.Goroutines,
			health.CPU.NumCPU,
			health.Memory.AllocMB,
			health.Memory.TotalAllocMB,
			health.Memory.SysMB,
			health.Memory.NumGC,
		)
		
		w.Write([]byte(html))
	}
}

func configureLogging() {
	// Set log format to logfmt (key=value pairs)
	log.SetFormatter(&logrus.TextFormatter{
		DisableColors:   true,
		FullTimestamp:   true,
		TimestampFormat: time.RFC3339,
	})

	// Get log level from environment variable
	logLevelStr := strings.ToLower(os.Getenv("LOG_LEVEL"))
	if logLevelStr == "" {
		logLevelStr = "info" // Default to info level
	}

	var logLevel logrus.Level
	switch logLevelStr {
	case "debug":
		logLevel = logrus.DebugLevel
	case "info":
		logLevel = logrus.InfoLevel
	case "warn", "warning":
		logLevel = logrus.WarnLevel
	case "error":
		logLevel = logrus.ErrorLevel
	case "fatal":
		logLevel = logrus.FatalLevel
	case "panic":
		logLevel = logrus.PanicLevel
	default:
		log.Warnf("Invalid LOG_LEVEL '%s', defaulting to 'info'. Valid levels: debug, info, warn, error, fatal, panic", logLevelStr)
		logLevel = logrus.InfoLevel
	}

	log.SetLevel(logLevel)
	log.WithFields(logrus.Fields{
		"log_level": logLevel.String(),
		"format":    "logfmt",
	}).Info("Logging configured")
}

func main() {
	configureLogging()
	log.Info("Starting ECR Prometheus Exporter")

	// Load AWS configuration
	log.Info("Loading AWS configuration...")
	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		log.Fatalf("Failed to load AWS config: %v", err)
	}
	log.Info("AWS configuration loaded successfully")

	// Create ECR client
	log.Info("Creating ECR client...")
	ecrClient := ecr.NewFromConfig(cfg)

	// Test AWS connectivity
	log.Info("Testing AWS connectivity...")
	testCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	
	_, err = ecrClient.DescribeRepositories(testCtx, &ecr.DescribeRepositoriesInput{
		MaxResults: aws.Int32(1),
	})
	if err != nil {
		log.Errorf("AWS connectivity test failed: %v", err)
		log.Info("Continuing anyway, metrics collection will show errors...")
	} else {
		log.Info("AWS connectivity test successful")
	}

	// Create and register the collector
	log.Info("Registering Prometheus collector...")
	collector := NewECRCollector(ecrClient)
	prometheus.MustRegister(collector)

	// Setup HTTP server
	http.Handle("/metrics", promhttp.Handler())
	http.HandleFunc("/health", healthHandler)
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`<html>
			<head>
				<title>ECR Exporter</title>
				<style>
					body { font-family: Arial, sans-serif; margin: 40px; }
					.link { display: block; margin: 10px 0; padding: 10px; background: #f5f5f5; text-decoration: none; border-radius: 5px; }
					.link:hover { background: #e5e5e5; }
				</style>
			</head>
			<body>
				<h1>ECR Prometheus Exporter</h1>
				<p>Monitor your AWS ECR repositories with Prometheus metrics</p>
				
				<h2>Available Endpoints:</h2>
				<a href="/metrics" class="link">📊 Prometheus Metrics</a>
				<a href="/health" class="link">💚 Health Status (with system metrics)</a>
				<a href="/health?format=json" class="link">📋 Health Status (JSON)</a>
				
				<h2>Metrics Exported:</h2>
				<ul>
					<li>Total ECR repositories</li>
					<li>Image count per repository</li>
					<li>Image size statistics (min, max, avg)</li>
					<li>Latest push/pull timestamps</li>
					<li>Scrape performance metrics</li>
				</ul>
			</body>
			</html>`))
	})

	log.Infof("Server starting on port %s", port)
	if err := http.ListenAndServe(port, nil); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}