package main

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"
)

const (
	DATAFILE = "data/azure/latency_matrix.csv"
	PORT     = "8080"
)

type RegionLatency struct {
	OriginRegion string  `json:"origin_region"`
	MaxLatency   float64 `json:"max_latency"`
}

type RegionResponse struct {
	EligibleRegions []string `json:"eligible_regions"`
}

type ErrorResponse struct {
	Error string `json:"error"`
}

type LatencyService struct {
	latencyMatrix map[string]map[string]float64
	regions       []string
}

type Server struct {
	service *LatencyService
}

func NewLatencyService(filename string) (*LatencyService, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("error opening file: %v", err)
	}
	defer file.Close()

	reader := csv.NewReader(file)

	// Read header to get regions
	headers, err := reader.Read()
	if err != nil {
		return nil, fmt.Errorf("error reading header: %v", err)
	}

	// Initialize service
	service := &LatencyService{
		latencyMatrix: make(map[string]map[string]float64),
		regions:       headers[1:], // Skip the "Source" column
	}

	// Read data rows
	for {
		row, err := reader.Read()
		if err != nil {
			break // End of file
		}

		sourceRegion := row[0]
		service.latencyMatrix[sourceRegion] = make(map[string]float64)

		for i, latencyStr := range row[1:] {
			if latencyStr == "N/A" {
				continue
			}

			latency, err := strconv.ParseFloat(latencyStr, 64)
			if err != nil {
				log.Printf("Warning: could not parse latency value %s for region %s: %v",
					latencyStr, headers[i+1], err)
				continue
			}

			service.latencyMatrix[sourceRegion][headers[i+1]] = latency
		}
	}

	return service, nil
}

func (s *LatencyService) FindEligibleRegions(originRegion string, maxLatency float64) ([]string, error) {
	latencies, exists := s.latencyMatrix[originRegion]
	if !exists {
		return nil, fmt.Errorf("origin region %s not found", originRegion)
	}

	var eligibleRegions []string
	for region, latency := range latencies {
		if latency <= maxLatency {
			eligibleRegions = append(eligibleRegions, region)
		}
	}

	// adding the origin region to the list of eligible regions if it is not already there
	// this is to ensure that the origin region is always included in the response
	// as it could happen that in the latency matrix it has a latency of N/A
	if _, exists := latencies[originRegion]; !exists {
		eligibleRegions = append(eligibleRegions, originRegion)
	}

	return eligibleRegions, nil
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "healthy"})
}

func (s *Server) handleEligibleRegions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	var request RegionLatency
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeJSONError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate input
	if request.OriginRegion == "" {
		writeJSONError(w, "origin_region is required", http.StatusBadRequest)
		return
	}
	if request.MaxLatency <= 0 {
		writeJSONError(w, "max_latency must be greater than 0", http.StatusBadRequest)
		return
	}

	// Find eligible regions
	eligibleRegions, err := s.service.FindEligibleRegions(request.OriginRegion, request.MaxLatency)
	if err != nil {
		writeJSONError(w, err.Error(), http.StatusBadRequest)
		return
	}

	json.NewEncoder(w).Encode(RegionResponse{EligibleRegions: eligibleRegions})
}

func writeJSONError(w http.ResponseWriter, message string, statusCode int) {
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(ErrorResponse{Error: message})
}

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		log.Printf("Started %s %s", r.Method, r.URL.Path)
		next.ServeHTTP(w, r)
		log.Printf("Completed %s %s in %v", r.Method, r.URL.Path, time.Since(start))
	})
}

func main() {
	service, err := NewLatencyService(DATAFILE)
	if err != nil {
		log.Fatalf("Failed to initialize latency service: %v", err)
	}

	server := &Server{service: service}

	// Define routes
	mux := http.NewServeMux()
	mux.HandleFunc("/health", server.handleHealth)
	mux.HandleFunc("/regions/eligible", server.handleEligibleRegions)

	// Add middleware
	handler := loggingMiddleware(mux)

	// Configure server
	port := os.Getenv("PORT")
	if port == "" {
		port = PORT
	}

	srv := &http.Server{
		Addr:         ":" + port,
		Handler:      handler,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	log.Printf("Server starting on port %s", port)
	if err := srv.ListenAndServe(); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
