package hibp

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// Checker handles the password checking logic
type Checker struct {
	client       *Client
	exposedUsers map[string]bool // only stores accounts that are exposed
	userHashes   map[string][]string // hash -> list of accounts with that hash
	cacheMu      sync.RWMutex
}

// NewChecker creates a new Checker instance
func NewChecker() *Checker {
	return &Checker{
		client:       NewClient(),
		exposedUsers: make(map[string]bool),
		userHashes:   make(map[string][]string),
	}
}

// User represents an account with its hash
type User struct {
	Account string
	Hash    string
}

// prefixJob represents a hash prefix to query
type prefixJob struct {
	prefix string
}

// ResultWriter handles streaming results to output
type ResultWriter struct {
	writer  io.Writer
	mu      sync.Mutex
	count   int
	enabled bool
}

// NewResultWriter creates a new result writer
func NewResultWriter(w io.Writer) *ResultWriter {
	return &ResultWriter{
		writer:  w,
		enabled: w != nil,
	}
}

// Write writes an exposed account to the output
func (rw *ResultWriter) Write(account string) error {
	if !rw.enabled {
		return nil
	}
	rw.mu.Lock()
	defer rw.mu.Unlock()
	_, err := fmt.Fprintln(rw.writer, account)
	if err == nil {
		rw.count++
	}
	return err
}

// Count returns the number of accounts written
func (rw *ResultWriter) Count() int {
	rw.mu.Lock()
	defer rw.mu.Unlock()
	return rw.count
}

// CheckFile reads a file and checks all hashes against HIBP
func (c *Checker) CheckFile(filename, delimiter string, skipHeader bool, workers int, limit int, resultWriter *ResultWriter) (int, error) {
	users, err := c.loadUsers(filename, delimiter, skipHeader, limit)
	if err != nil {
		return 0, err
	}

	// Build a map of hash -> accounts and collect unique prefixes
	prefixesToQuery := c.buildHashIndex(users)

	fmt.Printf("Found %d users, %d unique hash prefixes to query\n", len(users), len(prefixesToQuery))

	// Query all prefixes concurrently - this now checks against userHashes directly
	if len(prefixesToQuery) > 0 {
		c.queryPrefixesConcurrently(prefixesToQuery, workers, resultWriter)
	}

	// Count exposed users
	c.cacheMu.RLock()
	exposedCount := len(c.exposedUsers)
	c.cacheMu.RUnlock()

	return exposedCount, nil
}

// buildHashIndex builds a map of hash -> accounts and returns unique prefixes
func (c *Checker) buildHashIndex(users []User) []string {
	seen := make(map[string]bool)
	var prefixes []string

	for _, user := range users {
		// Skip computer accounts and empty hashes
		if strings.HasSuffix(user.Account, "$") || strings.TrimSpace(user.Hash) == "" {
			continue
		}

		hash := strings.ToUpper(user.Hash)
		if len(hash) < 5 {
			continue
		}

		// Store hash -> account mapping
		c.userHashes[hash] = append(c.userHashes[hash], user.Account)

		prefix := hash[:5]
		if !seen[prefix] {
			seen[prefix] = true
			prefixes = append(prefixes, prefix)
		}
	}

	return prefixes
}

// queryPrefixesConcurrently queries HIBP API for all prefixes using worker pool
// It checks matches against userHashes inline to avoid storing all HIBP results
func (c *Checker) queryPrefixesConcurrently(prefixes []string, workers int, resultWriter *ResultWriter) {
	if workers < 1 {
		workers = 1
	}

	jobs := make(chan prefixJob, len(prefixes))
	var wg sync.WaitGroup

	var completed atomic.Int64
	total := len(prefixes)
	startTime := time.Now()

	// Start workers
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for job := range jobs {
				c.checkPrefixMatches(job.prefix, resultWriter)

				current := completed.Add(1)
				elapsed := time.Since(startTime)
				c.printQueryProgress(int(current), total, elapsed)
			}
		}()
	}

	// Send all jobs
	for _, prefix := range prefixes {
		jobs <- prefixJob{prefix: prefix}
	}
	close(jobs)

	// Wait for all workers to complete
	wg.Wait()

	// Clear progress line
	fmt.Print("\r\033[K")
	fmt.Printf("Queried %d prefixes in %s using %d workers\n",
		total, time.Since(startTime).Round(time.Millisecond), workers)
}

// checkPrefixMatches queries a prefix and checks for matches against user hashes
func (c *Checker) checkPrefixMatches(prefix string, resultWriter *ResultWriter) {
	resp, err := c.client.QueryRangeRaw(prefix)
	if err != nil {
		fmt.Printf("\n[ERROR] Failed to query prefix %s: %v\n", prefix, err)
		return
	}

	// Check each returned hash suffix against our user hashes
	for _, line := range strings.Split(resp, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}

		// Reconstruct full hash and check if any user has it
		fullHash := prefix + strings.ToUpper(parts[0])

		c.cacheMu.RLock()
		accounts, exists := c.userHashes[fullHash]
		c.cacheMu.RUnlock()

		if exists {
			c.cacheMu.Lock()
			for _, account := range accounts {
				if !c.exposedUsers[account] {
					c.exposedUsers[account] = true
					fmt.Printf("[EXPOSED] %s\n", account)
					if err := resultWriter.Write(account); err != nil {
						fmt.Printf("[ERROR] Failed to write result: %v\n", err)
					}
				}
			}
			c.cacheMu.Unlock()
		}
	}
}

// loadUsers reads the input file and returns a slice of Users
func (c *Checker) loadUsers(filename, delimiter string, skipHeader bool, limit int) ([]User, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	var users []User
	scanner := bufio.NewScanner(file)
	firstLine := true

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		// Skip header row if requested
		if firstLine && skipHeader {
			firstLine = false
			continue
		}
		firstLine = false

		parts := strings.SplitN(line, delimiter, 2)
		if len(parts) != 2 {
			continue
		}

		users = append(users, User{
			Account: parts[0],
			Hash:    parts[1],
		})

		// Stop if we've reached the limit
		if limit > 0 && len(users) >= limit {
			break
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	return users, nil
}

// printQueryProgress displays the query progress
func (c *Checker) printQueryProgress(current, total int, elapsed time.Duration) {
	percent := float64(current) / float64(total) * 100
	fmt.Printf("\r\033[KQuerying HIBP [%s] %d/%d (%.1f%%)",
		elapsed.Round(time.Millisecond),
		current,
		total,
		percent,
	)
}
