package maz

import (
	"fmt"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/queone/utl"
)

/*
Retrieves Azure directory object deltas. Returns the set of new or updated items,
and a deltaLink for running the next future Azure query. Implements the code logic
pattern described at https://docs.microsoft.com/en-us/graph/delta-query-overview in four steps:

1. Initial delta query: GET /resource/delta to start tracking changes
2. Paginate via @odata.nextLink until @odata.deltaLink is received
3. Subsequent queries use deltaLink to fetch only new changes
4. Supports $select but not with $skiptoken; use $deltatoken=latest to skip initial sync

This implementation was originally attempted using parallel logic. However, delta queries
in Microsoft Graph are fundamentally designed for sequential processing, and all parallel
attempts led to inconsistent or failed results. The parallel implementation is preserved
in the codebase for posterity, while the primary function was renamed to FetchDirObjectsDeltaParallel.

----------------------------------------------------------------------------------
Delta Query Parallelization Constraints

Core Limitations:
1. Sequential Pagination – @odata.nextLink URLs contain stateful server context:
   - Each link depends on the previous call’s execution
   - Pages cannot be fetched out of order

2. Single DeltaLink Origin – Final delta token only appears in the last page:
   - Workers cannot independently determine sync completion
   - Requires centralized coordination

3. Stateful Sessions – Each delta chain maintains server-side state:
   - Concurrent sessions create divergent sync states
   - Risk of missed updates or duplicate processing

Microsoft’s Implicit Guidance:
- All official examples show single-threaded implementations
- Documentation never suggests parallel page fetching
- Rate limits (5k requests per 10 minutes) discourage brute-force concurrency
----------------------------------------------------------------------------------
*/

// Fetches Azure object changes and returns updates + deltaLink for next query
func FetchDirObjectsDelta(apiUrl string, cache *Cache, z *Config) (AzureObjectList, AzureObject) {

	Logf("Starting directory objects delta fetch\n")
	deltaSet := AzureObjectList{}
	deltaLinkMap := AzureObject{}
	currentUrl := apiUrl

	// Save interval configuration
	const saveInterval = 5000 // Save every 5000 items
	lastSave := 0             // Moved outside the loop

	// Continue fetching until we've processed all pages
	for currentUrl != "" {
		// Fetch the current page
		resp, err := apiGetWithRetry(currentUrl, z, 3)
		if err != nil {
			Logf("Error fetching %s: %v\n", currentUrl, err)
			break
		}

		// Process the response
		if value := utl.Slice(resp["value"]); value != nil {
			for _, item := range value {
				if obj := utl.Map(item); obj != nil {
					deltaSet = append(deltaSet, obj)
				}
			}
		}

		// Log progress and save partial delta set periodically
		currentCount := len(deltaSet)
		if currentCount-lastSave >= saveInterval {
			countStr := utl.Cya(utl.ToStr(currentCount))
			Logf("Processed %s items (current URL: %s)\n", countStr, currentUrl)

			if err := SaveFileBinaryList(cache.partialFilePath, deltaSet, 0600, false); err != nil {
				Logf("WARNING: Failed to save partial delta set: %v\n", err)
			}

			lastSave = currentCount // Update last save position
		}

		// Check for delta link first (higher priority than nextLink)
		if delta := utl.Str(resp["@odata.deltaLink"]); delta != "" {
			deltaLinkMap["@odata.deltaLink"] = delta
		}

		// Move to next page if available
		currentUrl = utl.Str(resp["@odata.nextLink"])
	}

	// Final save (ensure we capture everything)
	if len(deltaSet) > 0 && len(deltaSet) != lastSave {
		if err := SaveFileBinaryList(cache.partialFilePath, deltaSet, 0600, false); err != nil {
			Logf("WARNING: Final save failed: %v\n", err)
		}
	}

	countStr := utl.Cya(utl.ToStr(len(deltaSet)))
	Logf("Completed fetch. Total items: %s\n", countStr)
	return deltaSet, deltaLinkMap
}

// Performs an HTTP GET with retry and exponential backoff, up to a maximum number of attempts.
func apiGetWithRetry(url string, z *Config, maxRetries int) (resp map[string]interface{}, err error) {
	var statusCode int
	for attempt := 0; attempt < maxRetries; attempt++ {
		resp, statusCode, err = ApiGet(url, z, nil)

		if statusCode >= 200 && statusCode < 300 && err == nil {
			Logf("HTTP %s - Success (Attempt %d/%d)\n", colorStatus(statusCode), attempt+1, maxRetries)
			return resp, nil
		}

		// Log failure with colored status
		statusStr := colorStatus(statusCode)
		if err == nil {
			err = fmt.Errorf("unexpected status code: %d", statusCode)
		}
		Logf("HTTP %s - Failed (Attempt %d/%d): %v\n", statusStr, attempt+1, maxRetries, err)

		// Exponential backoff
		if attempt < maxRetries-1 {
			backoff := time.Second * time.Duration(1<<uint(attempt))
			time.Sleep(backoff)
		}
	}
	return nil, fmt.Errorf("after %d attempts: %w", maxRetries, err)
}

// Helper function for colored status output
func colorStatus(code int) string {
	str := strconv.Itoa(code)
	if code >= 200 && code < 300 {
		return utl.Gre(str) // Green for success
	}
	return utl.Red(str) // Red for errors
}

// ----------------------------------------------------------------------------------
// ----------------------------------------------------------------------------------
// PARALLEL attempt. Does not work. Keeping for posterity. See notes above.

type deltaSyncState struct {
	pendingMu   sync.Mutex
	pendingUrls int
	visited     *sync.Map
	workQueue   chan string

	callCountMu sync.Mutex
	callCount   int
}

func (s *deltaSyncState) incrementPending() {
	s.pendingMu.Lock()
	s.pendingUrls++
	Logf("incrementPending: %d\n", s.pendingUrls)
	s.pendingMu.Unlock()
}

func (s *deltaSyncState) decrementPending() {
	s.pendingMu.Lock()
	s.pendingUrls--
	Logf("decrementPending: %d\n", s.pendingUrls)
	s.pendingMu.Unlock()
}

// Atomically enqueue a URL only if it hasn't been seen before
func (s *deltaSyncState) enqueueIfUnseen(url string) {
	_, loaded := s.visited.LoadOrStore(url, true)
	if !loaded {
		s.incrementPending()
		s.workQueue <- url
	} else {
		Logf("SKIPPED duplicate URL: %s\n", url)
	}
}

func (s *deltaSyncState) incrementCallCount() int {
	s.callCountMu.Lock()
	defer s.callCountMu.Unlock()
	s.callCount++
	return s.callCount
}

func (s *deltaSyncState) getCallCount() int {
	s.callCountMu.Lock()
	defer s.callCountMu.Unlock()
	return s.callCount
}

func getWorkerConfig() (workers, bufSize int) {
	// Number of concurrent workers processing URLs
	workers = runtime.NumCPU() * 2
	if workers < 5 {
		workers = 5 // Safety cap
	} else if workers > 20 {
		workers = 20 // Cap for API safety
	}

	// Buffer size for results channel
	bufSize = workers * 1000
	if bufSize > 20000 {
		bufSize = 20000 // Safety cap
	}

	return workers, bufSize
}

// Fetches Azure object changes and returns updates + deltaLink for next query
func FetchDirObjectsDeltaParallel(apiUrl string, z *Config) (AzureObjectList, AzureObject) {
	Logf("Starting directory objects delta fetch\n")
	deltaSet := AzureObjectList{}
	deltaLinkMap := AzureObject{}

	// WHAT EXACTLY ARE WE PARALLELIZING?: Multiple API URLs (e.g. paged results) are fetched
	// in parallel using a pool of deltaWorker goroutines, speeding up large directory syncs.

	workerCount, resultBufSize := getWorkerConfig()
	results := make(chan AzureObject, resultBufSize)
	workQueue := make(chan string, 100) // Buffered channel for URLs to process
	state := &deltaSyncState{
		visited:   &sync.Map{}, // Thread-safe map for tracking seen URLs
		workQueue: workQueue,   // Shared work queue
	}

	// Use a waitgroup to track both workers and the results processor
	var wg sync.WaitGroup

	// Start worker goroutines
	for i := 0; i < workerCount; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			deltaWorker(id, results, z, state)
		}(i)
	}

	// Start a separate goroutine to close channels when all workers are done
	go func() {
		wg.Wait()
		close(results)
	}()

	// Enqueue the initial URL after workers are started to prevent races
	state.enqueueIfUnseen(apiUrl)

	// Process results in the main goroutine
	dedupe := map[string]struct{}{} // Local deduplication map
	for obj := range results {
		id := utl.Str(obj["id"])
		if id != "" {
			if _, exists := dedupe[id]; exists {
				continue // Skip duplicates
			}
			dedupe[id] = struct{}{}
		}
		deltaSet = append(deltaSet, obj)

		// Periodic progress logging
		if len(deltaSet)%100 == 0 {
			Logf("Call %05d : count %07d\n", state.getCallCount(), len(deltaSet))
		}
	}

	// Find the final deltaLink (last one seen by any thread)
	state.visited.Range(func(key, val any) bool {
		if url, ok := key.(string); ok && strings.Contains(url, "deltaLink") {
			deltaLinkMap["@odata.deltaLink"] = url
		}
		return true
	})

	return deltaSet, deltaLinkMap
}

// Processes a stream of API URLs from the work queue and sends parsed objects to the results
// channel. Workers follow pagination by enqueueing @odata.nextLink if unseen.
func deltaWorker(workerID int, results chan<- AzureObject, z *Config, state *deltaSyncState) {
	defer Logf("Worker %02d exiting\n", workerID)

	for url := range state.workQueue {
		resp, err := apiGetWithRetry(url, z, 3)
		if err != nil {
			Logf("Worker %02d error: %v\n", workerID, err)
			state.decrementPending()
			continue
		}

		count := state.incrementCallCount()
		Logf("Call %05d : Worker %02d finished: %s\n", count, workerID, url)
		processApiResponse(resp, results, workerID)

		// Handle delta link (for tracking the final sync state)
		if delta := utl.Str(resp["@odata.deltaLink"]); delta != "" {
			state.visited.Store(delta, true)
		}

		// Handle next link (for pagination)
		if next := utl.Str(resp["@odata.nextLink"]); next != "" {
			state.enqueueIfUnseen(next)
		}

		state.decrementPending()
	}
}

// Extracts directory objects from the raw API response and pushes them to the results channel.
func processApiResponse(resp map[string]interface{}, results chan<- AzureObject, workerID int) {
	// Optionally tag each object with metadata (e.g. fetch source or worker ID) to help with
	// future debugging or analysis. These fields are omitted for now but can be re-enabled easily.

	// source := utl.Str(resp["@odata.context"]) // Could also use "@odata.deltaLink" or similar

	if value := utl.Slice(resp["value"]); value != nil {
		for _, item := range value {
			if obj := utl.Map(item); obj != nil {

				// obj["fetched_from"] = source // Enable to include source URL/context

				// // Track which goroutine fetched it
				// fetchedBy := "main"
				// if workerID >= 0 {
				// 	fetchedBy = fmt.Sprintf("worker-%02d", workerID)
				// }
				// obj["fetched_by"] = fetchedBy
				_ = workerID // keep compiler happy for now

				// Logf("Fetched by worker %02\n", workerID) // Or maybe just Log it?

				results <- AzureObject(obj)
			}
		}
	}
}
