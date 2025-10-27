package llm

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// Harness orchestrates LLM validation tests.
// It runs test cases against multiple LLM providers and generates reports.
type Harness struct {
	config          *Config
	clients         map[string]Client
	validator       *Validator
	reporter        *Reporter
	verbose         bool
	lastRequestTime map[string]time.Time // Per-provider rate limiting
	rateLimitMux    sync.Mutex           // Protects lastRequestTime
}

// NewHarness creates a new test harness with the given configuration.
func NewHarness(config *Config) (*Harness, error) {
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	// Create clients for each enabled provider
	clients := make(map[string]Client)
	for _, provider := range config.EnabledProviders() {
		client, err := NewClient(provider)
		if err != nil {
			return nil, fmt.Errorf("failed to create client for %s: %w", provider.String(), err)
		}
		clients[provider.String()] = client
	}

	return &Harness{
		config:          config,
		clients:         clients,
		validator:       NewValidator(),
		reporter:        NewReporter(),
		verbose:         false,
		lastRequestTime: make(map[string]time.Time),
	}, nil
}

// SetVerbose enables or disables verbose output.
func (h *Harness) SetVerbose(verbose bool) {
	h.verbose = verbose
}

// waitForRateLimit implements per-provider rate limiting.
// Only delays if necessary based on time since last request to this provider.
func (h *Harness) waitForRateLimit(provider string) {
	if h.config.RateLimitDelay == 0 {
		return
	}

	h.rateLimitMux.Lock()
	defer h.rateLimitMux.Unlock()

	if lastTime, ok := h.lastRequestTime[provider]; ok {
		elapsed := time.Since(lastTime)
		if elapsed < h.config.RateLimitDelay {
			time.Sleep(h.config.RateLimitDelay - elapsed)
		}
	}
	h.lastRequestTime[provider] = time.Now()
}

// Run executes all test cases against all configured LLM providers.
// Returns a report with results.
func (h *Harness) Run(ctx context.Context, testCases []TestCase) (Report, error) {
	startTime := time.Now()

	if h.verbose {
		fmt.Printf("Starting test harness with %d test cases and %d providers\n",
			len(testCases), len(h.clients))
	}

	// Create a list of all test executions (test case × provider)
	type testExecution struct {
		testCase TestCase
		provider string
		client   Client
	}

	var executions []testExecution
	for _, tc := range testCases {
		for provider, client := range h.clients {
			executions = append(executions, testExecution{
				testCase: tc,
				provider: provider,
				client:   client,
			})
		}
	}

	// Execute tests with controlled concurrency
	results := make([]TestResult, len(executions))
	resultIndex := 0
	resultsChan := make(chan TestResult, len(executions))
	semaphore := make(chan struct{}, h.config.MaxConcurrentRequests)

	var wg sync.WaitGroup

	for _, exec := range executions {
		wg.Add(1)
		go func(exec testExecution) {
			defer wg.Done()

			// Per-provider rate limiting
			h.waitForRateLimit(exec.provider)

			// Acquire semaphore
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			result := h.executeTest(ctx, exec.testCase, exec.provider, exec.client)
			resultsChan <- result

			if h.verbose {
				status := "PASS"
				if result.Error != nil {
					status = "ERROR"
				} else if !result.Validation.Passed {
					status = "FAIL"
				}
				fmt.Printf("[%s] %s - %s\n", status, exec.provider, exec.testCase.Name)
			}

		}(exec)
	}

	// Wait for all tests to complete
	go func() {
		wg.Wait()
		close(resultsChan)
	}()

	// Collect results
	for result := range resultsChan {
		results[resultIndex] = result
		resultIndex++
	}

	executionTime := time.Since(startTime)

	// Generate report
	report := h.reporter.GenerateReport(results, executionTime)

	return report, nil
}

// executeTest executes a single test case with a specific provider.
func (h *Harness) executeTest(ctx context.Context, tc TestCase, provider string, client Client) TestResult {
	startTime := time.Now()

	result := TestResult{
		TestCase:  tc,
		Provider:  provider,
		Timestamp: startTime,
	}

	// Call LLM
	response, err := client.Generate(ctx, tc.Prompt)
	if err != nil {
		result.Error = fmt.Errorf("LLM call failed: %w", err)
		result.Duration = time.Since(startTime)
		return result
	}

	result.Response = response

	// Validate response
	validation := h.validator.Validate(response, tc.ExpectedPattern, tc.ValidationMode)
	result.Validation = validation
	result.Duration = time.Since(startTime)

	return result
}

// RunWithMockLLM runs tests with a mock LLM for testing purposes.
// The mock function receives the prompt and returns a response.
func (h *Harness) RunWithMockLLM(ctx context.Context, testCases []TestCase, mockFn func(string) string) (Report, error) {
	startTime := time.Now()

	var results []TestResult

	for _, tc := range testCases {
		for provider := range h.clients {
			result := TestResult{
				TestCase:  tc,
				Provider:  provider,
				Timestamp: time.Now(),
			}

			// Call mock
			response := mockFn(tc.Prompt)
			result.Response = response

			// Validate
			validation := h.validator.Validate(response, tc.ExpectedPattern, tc.ValidationMode)
			result.Validation = validation
			result.Duration = 10 * time.Millisecond // Simulated duration

			results = append(results, result)
		}
	}

	executionTime := time.Since(startTime)
	report := h.reporter.GenerateReport(results, executionTime)

	return report, nil
}

// RunSingle executes a single test case with all providers.
// Useful for debugging specific test cases.
func (h *Harness) RunSingle(ctx context.Context, tc TestCase) ([]TestResult, error) {
	var results []TestResult

	for provider, client := range h.clients {
		result := h.executeTest(ctx, tc, provider, client)
		results = append(results, result)

		// Rate limiting
		if h.config.RateLimitDelay > 0 {
			time.Sleep(h.config.RateLimitDelay)
		}
	}

	return results, nil
}

// ValidateProvider checks if a specific provider is configured and available.
func (h *Harness) ValidateProvider(providerType ProviderType) error {
	for provider := range h.clients {
		if provider == string(providerType) {
			return nil
		}
	}
	return fmt.Errorf("provider %s not configured or unavailable", providerType)
}

// GetProviders returns the list of configured provider names.
func (h *Harness) GetProviders() []string {
	providers := make([]string, 0, len(h.clients))
	for provider := range h.clients {
		providers = append(providers, provider)
	}
	return providers
}

// QuickTest runs a quick validation test with a simple test case.
// Useful for smoke testing that the harness is working.
func (h *Harness) QuickTest(ctx context.Context) error {
	testCase := TestCase{
		Name:     "Quick test",
		Category: "authentication",
		Prompt: `Generate a middleware declaration for authentication on the create operation.
Return only: @on create: [auth]`,
		ExpectedPattern: "@on create: [auth]",
		ValidationMode:  "semantic",
	}

	if h.verbose {
		fmt.Println("Running quick test...")
	}

	results, err := h.RunSingle(ctx, testCase)
	if err != nil {
		return fmt.Errorf("quick test failed: %w", err)
	}

	passed := 0
	for _, result := range results {
		if result.Error == nil && result.Validation.Passed {
			passed++
		}
	}

	if h.verbose {
		fmt.Printf("Quick test: %d/%d providers passed\n", passed, len(results))
	}

	if passed == 0 {
		return fmt.Errorf("quick test failed: no providers passed")
	}

	return nil
}
