package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/opensearch-project/opensearch-go/v2"
	"github.com/opensearch-project/opensearch-go/v2/opensearchapi"
)

func main() {
	// Define command-line flags
	indicesPattern := flag.String("pattern", "", "Indices pattern (e.g., 'uat_*')")
	opensearchURL := flag.String("url", "", "OpenSearch URL")
	numToBypass := flag.Int("bypass", 0, "Number of latest indices to bypass")
	repoName := flag.String("repo", "", "Repository name in OpenSearch")
	enableAnalyze := flag.Bool("analyze", false, "Enable min/max timestamp analysis for indices")

	flag.Parse()

	// Validate inputs
	if *indicesPattern == "" || *opensearchURL == "" || *repoName == "" {
		log.Fatalf("Missing required arguments. Use --help for usage instructions.")
	}

	// Create OpenSearch client
	client, err := opensearch.NewClient(opensearch.Config{
		Addresses: []string{*opensearchURL},
	})
	if err != nil {
		log.Fatalf("Failed to create OpenSearch client: %s", err)
	}

	ctx := context.Background()

	// Fetch indices matching the pattern
	indices, err := getIndices(ctx, client, *indicesPattern)
	if err != nil {
		log.Fatalf("Error fetching indices: %s", err)
	}

	// Filter indices to archive
	if len(indices) <= *numToBypass {
		log.Println("No indices to archive.")
		return
	}
	indicesToArchive := indices[:len(indices)-*numToBypass]

	// Process each index
	for _, index := range indicesToArchive {
		snapshotName, err := generateSnapshotName(ctx, client, index, *enableAnalyze)
		if err != nil {
			log.Printf("Error generating snapshot name for index %s: %s", index, err)
			continue
		}

		log.Printf("Creating snapshot for index %s: %s", index, snapshotName)

		if err := createSnapshot(ctx, client, *repoName, index, snapshotName); err != nil {
			log.Printf("Error creating snapshot for index %s: %s", index, err)
		} else {
			log.Printf("Snapshot created successfully: %s", snapshotName)
		}
	}
}

// Fetch indices matching the pattern
func getIndices(ctx context.Context, client *opensearch.Client, pattern string) ([]string, error) {
	res, err := client.Cat.Indices(
		client.Cat.Indices.WithFormat("json"),
		client.Cat.Indices.WithIndex(pattern),
	)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	var indices []struct {
		Index string `json:"index"`
	}
	if err := json.NewDecoder(res.Body).Decode(&indices); err != nil {
		return nil, err
	}

	indexNames := make([]string, len(indices))
	for i, index := range indices {
		indexNames[i] = index.Index
	}

	// Extract numeric suffix and sort indices
	sort.Slice(indexNames, func(i, j int) bool {
		return extractIndexNumber(indexNames[i]) < extractIndexNumber(indexNames[j])
	})

	return indexNames, nil
}

// Extract numeric suffix from index name
func extractIndexNumber(index string) int {
	re := regexp.MustCompile(`\d+$`)
	match := re.FindString(index)
	if match == "" {
		return 0
	}
	num, _ := strconv.Atoi(match)
	return num
}

// Generate snapshot name
func generateSnapshotName(ctx context.Context, client *opensearch.Client, index string, analyze bool) (string, error) {
	if analyze {
		minTS, maxTS, err := analyzeTimestamps(ctx, client, index)
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("%s.%s.%s", index, minTS, maxTS), nil
	}

	return fmt.Sprintf("%s", index), nil
}

// Analyze min/max timestamps of data in the index
func analyzeTimestamps(ctx context.Context, client *opensearch.Client, index string) (string, string, error) {
	query := `{
		"size": 0,
		"aggs": {
			"min_time": { "min": { "field": "timestamp" } },
			"max_time": { "max": { "field": "timestamp" } }
		}
	}`

	res, err := client.Search(
		client.Search.WithContext(ctx),
		client.Search.WithIndex(index),
		client.Search.WithBody(strings.NewReader(query)),
		client.Search.WithPretty(),
	)
	if err != nil {
		return "", "", err
	}
	defer res.Body.Close()

	var result struct {
		Aggregations struct {
			MinTime struct {
				Value float64 `json:"value"`
			} `json:"min_time"`
			MaxTime struct {
				Value float64 `json:"value"`
			} `json:"max_time"`
		} `json:"aggregations"`
	}

	if err := json.NewDecoder(res.Body).Decode(&result); err != nil {
		return "", "", err
	}

	minTime := time.Unix(int64(result.Aggregations.MinTime.Value/1000), 0).Format("20060102-1504")
	maxTime := time.Unix(int64(result.Aggregations.MaxTime.Value/1000), 0).Format("20060102-1504")
	return minTime, maxTime, nil
}

// Create snapshot for the index
func createSnapshot(ctx context.Context, client *opensearch.Client, repo, index, snapshot string) error {
	// Check if the snapshot already exists
	if snapshotExists(ctx, client, repo, snapshot) {
		log.Printf("Snapshot %s already exists. Skipping creation.", snapshot)
		return nil
	}

	body := fmt.Sprintf(`{
		"indices": "%s",
		"include_global_state": false
	}`, index)

	req := opensearchapi.SnapshotCreateRequest{
		Repository: repo,
		Snapshot:   snapshot,
		Body:       strings.NewReader(body),
	}
	res, err := req.Do(ctx, client)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	if res.IsError() {
		return fmt.Errorf("failed to create snapshot: %s", res.String())
	}
	return nil
}

func snapshotExists(ctx context.Context, client *opensearch.Client, repo, snapshot string) bool {
	req := opensearchapi.SnapshotGetRequest{
		Repository: repo,
		Snapshot:   []string{snapshot},
	}
	res, err := req.Do(ctx, client)
	if err != nil {
		log.Printf("Error checking for snapshot %s: %s", snapshot, err)
		return false
	}
	defer res.Body.Close()

	// If the snapshot exists, the response won't be an error
	return !res.IsError()
}
