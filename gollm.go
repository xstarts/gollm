// File: gollm.go

package gollm

import (
	"context"
	"fmt"
	"strings"
	"time"

	"gollm/internal/llm"
)

// LLM is the interface that wraps the basic LLM operations
type LLM interface {
	// Generate produces a response given a context, prompt, and optional generate options
	Generate(ctx context.Context, prompt *Prompt, opts ...GenerateOption) (string, error)

	// SetOption sets an option for the LLM
	SetOption(key string, value interface{})

	// GetPromptJSONSchema returns the JSON schema for the prompt
	GetPromptJSONSchema(opts ...SchemaOption) ([]byte, error)

	// GetProvider returns the provider of the LLM
	GetProvider() string

	// GetModel returns the model of the LLM
	GetModel() string

	// UpdateDebugLevel updates the debug level of the LLM
	UpdateDebugLevel(level LogLevel)

	// Debug logs a debug message with optional key-value pairs
	Debug(msg string, keysAndValues ...interface{})

	// GetDebugLevel returns the current debug level of the LLM
	GetDebugLevel() LogLevel

	SetOllamaEndpoint(endpoint string) error
}

// llmImpl is the concrete implementation of the LLM interface
type llmImpl struct {
	llm.LLM  // Embedded LLM interface from internal package
	logger   llm.Logger
	provider string
	model    string
	config   *Config
}

// GenerateOption is a function type for configuring generate options
type GenerateOption func(*generateConfig)

// generateConfig holds configuration options for the Generate method
type generateConfig struct {
	useJSONSchema bool
}

// WithJSONSchemaValidation returns a GenerateOption that enables JSON schema validation
func WithJSONSchemaValidation() GenerateOption {
	return func(c *generateConfig) {
		c.useJSONSchema = true
	}
}

// GetProvider returns the provider of the LLM
func (l *llmImpl) GetProvider() string {
	return l.provider
}

// GetModel returns the model of the LLM
func (l *llmImpl) GetModel() string {
	return l.model
}

// Debug logs a debug message with optional key-value pairs
func (l *llmImpl) Debug(msg string, keysAndValues ...interface{}) {
	l.logger.Debug(msg, keysAndValues...)
}

// GetDebugLevel returns the current debug level of the LLM
func (l *llmImpl) GetDebugLevel() LogLevel {
	return l.config.DebugLevel
}

// Type aliases to bridge public and internal types
type Metric = llm.Metric
type OptimizationEntry = llm.OptimizationEntry

// OptimizerOption is a function type for configuring the PromptOptimizer
type OptimizerOption func(*PromptOptimizer)

// IterationCallback is a function type for the iteration callback
type IterationCallback func(iteration int, entry OptimizationEntry)

// PromptOptimizer is the public interface for the prompt optimization system
type PromptOptimizer struct {
	internal   *llm.PromptOptimizer
	callback   IterationCallback
	memorySize int
	verbose    bool
}

func WithVerbose() OptimizerOption {
	return func(po *PromptOptimizer) {
		po.verbose = true
	}
}

// Modify the existing WithIterationCallback function
func WithIterationCallback(callback IterationCallback) OptimizerOption {
	return func(po *PromptOptimizer) {
		po.callback = callback
		po.verbose = false // Disable verbose mode when a custom callback is set
		po.internal.WithIterationCallback(func(iteration int, entry llm.OptimizationEntry) {
			if po.callback != nil {
				po.callback(iteration, OptimizationEntry(entry))
			}
		})
	}
}

// WithMaxRetries sets the maximum number of retries for the PromptOptimizer
func WithMaxRetries(maxRetries int) OptimizerOption {
	return func(po *PromptOptimizer) {
		po.internal.WithMaxRetries(maxRetries)
	}
}

// WithRetryDelay sets the delay between retries for the PromptOptimizer
func WithRetryDelay(delay time.Duration) OptimizerOption {
	return func(po *PromptOptimizer) {
		po.internal.WithRetryDelay(delay)
	}
}

// WithMemorySize sets the memory size for the PromptOptimizer
func WithMemorySize(size int) OptimizerOption {
	return func(po *PromptOptimizer) {
		po.memorySize = size
		po.internal.WithMemorySize(size)
	}
}

func WithIterations(iterations int) OptimizerOption {
	return func(po *PromptOptimizer) {
		po.internal.WithIterations(iterations)
	}
}

func defaultVerboseCallback(iteration int, entry OptimizationEntry) {
	fmt.Printf("\nIteration %d complete:\n", iteration)
	fmt.Printf("  Prompt: %s\n", entry.Prompt.Input)
	fmt.Printf("  Overall Score: %.2f\n", entry.Assessment.OverallScore)
	fmt.Printf("  Overall Grade: %s\n", entry.Assessment.OverallGrade)
	fmt.Println("  Metrics:")
	for _, metric := range entry.Assessment.Metrics {
		fmt.Printf("    - %s: %.2f\n", metric.Name, metric.Value)
	}
	fmt.Println("  Strengths:")
	for _, strength := range entry.Assessment.Strengths {
		fmt.Printf("    - %s (Example: %s)\n", strength.Point, strength.Example)
	}
	fmt.Println("  Weaknesses:")
	for _, weakness := range entry.Assessment.Weaknesses {
		fmt.Printf("    - %s (Example: %s)\n", weakness.Point, weakness.Example)
	}
	fmt.Println("  Suggestions:")
	for _, suggestion := range entry.Assessment.Suggestions {
		fmt.Printf("    - %s (Expected Impact: %.2f, Reasoning: %s)\n", suggestion.Description, suggestion.ExpectedImpact, suggestion.Reasoning)
	}
	fmt.Printf("%s\n", strings.Repeat("-", 50))
}

// OptimizePrompt runs the optimization process
func (po *PromptOptimizer) OptimizePrompt(ctx context.Context) (string, error) {
	optimizedPrompt, err := po.internal.OptimizePrompt(ctx)
	if err != nil {
		return "", fmt.Errorf("optimization failed: %w", err)
	}
	return optimizedPrompt.Input, nil
}

// NewPromptOptimizer creates a new PromptOptimizer
func NewPromptOptimizer(l LLM, initialPrompt string, taskDesc string, opts ...OptimizerOption) *PromptOptimizer {
	internalLLM, ok := l.(*llmImpl)
	if !ok {
		return nil
	}

	debugOptions := llm.DebugOptions{
		LogPrompts:   true,
		LogResponses: true,
	}
	debugManager := llm.NewDebugManager(internalLLM.logger, debugOptions)

	internalPrompt := &llm.Prompt{
		Input: initialPrompt,
	}

	po := &PromptOptimizer{
		internal:   llm.NewPromptOptimizer(internalLLM.LLM, debugManager, internalPrompt, taskDesc),
		memorySize: 2,
		verbose:    false, // Default to false
	}

	for _, opt := range opts {
		opt(po)
	}

	// Set the internal callback if verbose is true or a custom callback is set
	if po.verbose || po.callback != nil {
		po.internal.WithIterationCallback(func(iteration int, entry llm.OptimizationEntry) {
			if po.callback != nil {
				po.callback(iteration, OptimizationEntry(entry))
			} else if po.verbose {
				defaultVerboseCallback(iteration, OptimizationEntry(entry))
			}
		})
	}

	return po
}

// GetOptimizationHistory returns the history of optimization attempts
func (po *PromptOptimizer) GetOptimizationHistory() []OptimizationEntry {
	return po.internal.GetOptimizationHistory()
}

// WithCustomMetrics adds custom metrics to the PromptOptimizer
func WithCustomMetrics(metrics ...Metric) OptimizerOption {
	return func(po *PromptOptimizer) {
		po.internal.WithCustomMetrics(metrics...)
	}
}

// WithOptimizationGoal sets the optimization goal for the PromptOptimizer
func WithOptimizationGoal(goal string) OptimizerOption {
	return func(po *PromptOptimizer) {
		po.internal.WithOptimizationGoal(goal)
	}
}

// WithRatingSystem sets the rating system for the PromptOptimizer
func WithRatingSystem(system string) OptimizerOption {
	return func(po *PromptOptimizer) {
		po.internal.WithRatingSystem(system)
	}
}

// WithThreshold sets the threshold for the PromptOptimizer
func WithThreshold(threshold float64) OptimizerOption {
	return func(po *PromptOptimizer) {
		po.internal.WithThreshold(threshold)
	}
}

// SetOption sets an option for the LLM with the given key and value
func (l *llmImpl) SetOption(key string, value interface{}) {
	// Log the attempt to set an option
	l.logger.Debug("Setting option", "key", key, "value", value)

	// Call the SetOption method of the embedded LLM
	l.LLM.SetOption(key, value)

	// Log the successful setting of the option
	l.logger.Debug("Option set successfully")
}

func (l *llmImpl) SetOllamaEndpoint(endpoint string) error {
	if p, ok := l.LLM.(interface{ SetEndpoint(string) }); ok {
		p.SetEndpoint(endpoint)
		return nil
	}
	return fmt.Errorf("current provider does not support setting custom endpoint")
}

func (l *llmImpl) ClearMemory() {
	if llmWithMemory, ok := l.LLM.(*llm.LLMWithMemory); ok {
		llmWithMemory.ClearMemory()
	}
}

func (l *llmImpl) GetMemory() []llm.Message {
	if llmWithMemory, ok := l.LLM.(*llm.LLMWithMemory); ok {
		return llmWithMemory.GetMemory()
	}
	return nil
}

// GetPromptJSONSchema generates and returns the JSON schema for the Prompt
// It accepts optional SchemaOptions to customize the schema generation
func (l *llmImpl) GetPromptJSONSchema(opts ...SchemaOption) ([]byte, error) {
	// Create a new Prompt instance
	p := &Prompt{}

	// Generate and return the JSON schema for the Prompt
	// Pass along any provided SchemaOptions
	return p.GenerateJSONSchema(opts...)
}

// UpdateDebugLevel updates the debug level for both the gollm package and the internal llm package
func (l *llmImpl) UpdateDebugLevel(level LogLevel) {
	l.logger.Debug("Updating debug level",
		"current_level", l.config.DebugLevel,
		"new_level", level)

	l.config.DebugLevel = level
	l.logger.SetLevel(llm.LogLevel(level))

	if internalLLM, ok := l.LLM.(interface{ SetDebugLevel(llm.LogLevel) }); ok {
		internalLLM.SetDebugLevel(llm.LogLevel(level))
		l.logger.Debug("Updated internal LLM debug level")
	} else {
		l.logger.Warn("Internal LLM does not support SetDebugLevel")
	}

	l.logger.Debug("Debug level updated successfully")
}

// CleanResponse removes markdown code block syntax and trims the JSON response
func CleanResponse(response string) string {
	// Remove markdown code block syntax if present
	response = strings.TrimPrefix(response, "```json")
	response = strings.TrimSuffix(response, "```")

	// Remove any text before the first '{' and after the last '}'
	start := strings.Index(response, "{")
	end := strings.LastIndex(response, "}")
	if start != -1 && end != -1 && end > start {
		response = response[start : end+1]
	}

	return strings.TrimSpace(response)
}

func (l *llmImpl) Generate(ctx context.Context, prompt *Prompt, opts ...GenerateOption) (string, error) {
	// Log the start of the Generate method
	l.logger.Debug("Starting Generate method",
		"prompt_length", len(prompt.String()),
		"context", ctx)

	// Check if llmImpl is nil
	if l == nil {
		l.logger.Error("llmImpl is nil")
		return "", fmt.Errorf("llmImpl is nil")
	}

	// Check if internal LLM is nil
	if l.LLM == nil {
		l.logger.Error("internal LLM is nil")
		return "", fmt.Errorf("internal LLM is nil")
	}

	// Ensure logger is initialized
	if l.logger == nil {
		l.logger = llm.NewLogger(llm.LogLevel(LogLevelWarn))
		l.logger.Warn("Logger was nil, created new logger with WARN level")
	}

	// Apply generate options
	config := &generateConfig{}
	for _, opt := range opts {
		opt(config)
	}
	l.logger.Debug("Generate options applied",
		"useJSONSchema", config.useJSONSchema)

	// Validate prompt with JSON schema if enabled
	if config.useJSONSchema {
		l.logger.Debug("Validating prompt with JSON schema")
		if err := llm.Validate(prompt); err != nil {
			l.logger.Error("Prompt validation failed", "error", err)
			return "", fmt.Errorf("invalid prompt: %w", err)
		}
		l.logger.Debug("Prompt validation successful")
	}

	// Send prompt to LLM
	l.logger.Debug("Sending prompt to LLM",
		"prompt", prompt.String(),
		"provider", l.GetProvider(),
		"model", l.GetModel())
	response, fullPrompt, err := l.LLM.Generate(ctx, prompt.String())
	if err != nil {
		l.logger.Error("Error from LLM.Generate",
			"error", err,
			"fullPrompt", fullPrompt)
		return "", err
	}
	l.logger.Debug("Received response from LLM",
		"response_length", len(response))

	// Clean the response
	cleanedResponse := CleanResponse(response)
	l.logger.Debug("Response cleaned",
		"original_length", len(response),
		"cleaned_length", len(cleanedResponse))

	return cleanedResponse, nil
}

// NewLLM creates a new LLM instance, potentially with memory if the option is set
func NewLLM(opts ...ConfigOption) (LLM, error) {
	config, err := LoadConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	for _, opt := range opts {
		opt(config)
	}

	logger := llm.NewLogger(llm.LogLevel(config.DebugLevel))

	internalConfig := config.toInternalConfig()

	// Pass the OllamaEndpoint to the internal config
	if config.OllamaEndpoint != "" {
		internalConfig.OllamaEndpoint = config.OllamaEndpoint
	}

	baseLLM, err := llm.NewLLM(internalConfig, logger, llm.NewProviderRegistry())
	if err != nil {
		logger.Error("Failed to create internal LLM", "error", err)
		return nil, fmt.Errorf("failed to create internal LLM: %w", err)
	}

	if config.MemoryOption != nil {
		llmWithMemory, err := llm.NewLLMWithMemory(baseLLM, config.MemoryOption.MaxTokens, config.Model, logger)
		if err != nil {
			logger.Error("Failed to create LLM with memory", "error", err)
			return nil, fmt.Errorf("failed to create LLM with memory: %w", err)
		}
		return &llmImpl{
			LLM:      llmWithMemory,
			logger:   logger,
			provider: config.Provider,
			model:    config.Model,
			config:   config,
		}, nil
	}

	return &llmImpl{
		LLM:      baseLLM,
		logger:   logger,
		provider: config.Provider,
		model:    config.Model,
		config:   config,
	}, nil
}
