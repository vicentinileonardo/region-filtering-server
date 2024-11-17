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
	AZURE_LATENCY_MATRIX_FILE = "data/azure/azure_regions_latency_matrix.csv"
	AZURE_REGION_MAP_FILE     = "data/azure/azure_region_city_mapping.csv"
	PORT                      = "8080"
	CLOUD_AZURE               = "azure"
)

type RegionRequest struct {
	OriginRegion  string  `json:"origin_region"`
	MaxLatency    float64 `json:"max_latency"`
	CloudProvider string  `json:"cloud_provider"`
}

type RegionInfo struct {
	Name             string `json:"name"`
	ISOCountryCodeA2 string `json:"iso_country_code_a2"`
	PhysicalLocation string `json:"physical_location"`
}

type RegionMapping struct {
	isoCode  string
	location string
}

type RegionResponse struct {
	EligibleRegions []RegionInfo `json:"eligible_regions"`
}

type ErrorResponse struct {
	Error string `json:"error"`
}

type LatencyService struct {
	latencyMatrix  map[string]map[string]float64
	regions        []string
	regionMappings map[string]RegionMapping
}

type Server struct {
	service *LatencyService
}

func loadRegionMappings(filename string) (map[string]RegionMapping, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("error opening region mapping file: %v", err)
	}
	defer file.Close()

	reader := csv.NewReader(file)

	// Skip header
	_, err = reader.Read()
	if err != nil {
		return nil, fmt.Errorf("error reading header: %v", err)
	}

	mappings := make(map[string]RegionMapping)

	for {
		row, err := reader.Read()
		if err != nil {
			break // End of file
		}

		region := row[0]   // Region
		isoCode := row[1]  // ISO alpha-2
		location := row[3] // Physical Location

		// Some locations might be empty in the CSV, store what we have
		mappings[region] = RegionMapping{
			isoCode:  isoCode,
			location: location,
		}
	}

	return mappings, nil
}

func NewLatencyService(latencyFile string, mappingFile string) (*LatencyService, error) {
	// Load region mappings first
	regionMappings, err := loadRegionMappings(mappingFile)
	if err != nil {
		return nil, fmt.Errorf("error loading region mappings: %v", err)
	}

	// Load latency matrix
	file, err := os.Open(latencyFile)
	if err != nil {
		return nil, fmt.Errorf("error opening latency file: %v", err)
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
		latencyMatrix:  make(map[string]map[string]float64),
		regions:        headers[1:], // Skip the "Source" column
		regionMappings: regionMappings,
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

func (s *LatencyService) FindEligibleRegions(originRegion string, maxLatency float64) ([]RegionInfo, error) {
	latencies, exists := s.latencyMatrix[originRegion]
	if !exists {
		return nil, fmt.Errorf("origin region %s not found", originRegion)
	}

	var eligibleRegions []RegionInfo
	for region, latency := range latencies {
		if latency <= maxLatency {
			mapping, exists := s.regionMappings[region]
			if exists {
				eligibleRegions = append(eligibleRegions, RegionInfo{
					Name:             region,
					ISOCountryCodeA2: mapping.isoCode,
					PhysicalLocation: mapping.location,
				})
			} else {
				// If mapping doesn't exist, include the region with empty location info
				eligibleRegions = append(eligibleRegions, RegionInfo{
					Name:             region,
					ISOCountryCodeA2: "",
					PhysicalLocation: "",
				})
			}
		}
	}

	// adding the origin region to the list of eligible regions if it is not already there
	// this is to ensure that the origin region is always included in the response
	// as it could happen that in the latency matrix it has a latency of N/A
	if _, exists := latencies[originRegion]; !exists {
		mapping, exists := s.regionMappings[originRegion]
		if exists {
			eligibleRegions = append(eligibleRegions, RegionInfo{
				Name:             originRegion,
				ISOCountryCodeA2: mapping.isoCode,
				PhysicalLocation: mapping.location,
			})
		} else {
			eligibleRegions = append(eligibleRegions, RegionInfo{
				Name:             originRegion,
				ISOCountryCodeA2: "",
				PhysicalLocation: "",
			})
		}
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

	var request RegionRequest
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
	if request.CloudProvider == "" {
		writeJSONError(w, "cloud_provider is required", http.StatusBadRequest)
		return
	}
	if request.CloudProvider != CLOUD_AZURE {
		writeJSONError(w, "unsupported cloud provider", http.StatusBadRequest)
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
	service, err := NewLatencyService(AZURE_LATENCY_MATRIX_FILE, AZURE_REGION_MAP_FILE)
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
