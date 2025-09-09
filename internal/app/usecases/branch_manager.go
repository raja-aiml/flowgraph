package usecases

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/flowgraph/flowgraph/internal/app/dto"
	"github.com/google/uuid"
)

// BranchManager handles parallel execution paths and branch specifications
// PRINCIPLES:
// - Parallel execution path management
// - BranchSpec struct and join conditions
// - Dynamic branch creation and synchronization
type BranchManager struct {
	branches       map[string]*BranchSpec
	activeBranches map[string]*BranchExecution
	joinPoints     map[string]*JoinPoint
	mu             sync.RWMutex
}

// BranchSpec defines a parallel execution branch specification
type BranchSpec struct {
	ID            string                 `json:"id"`
	Name          string                 `json:"name"`
	ParentBranch  string                 `json:"parent_branch,omitempty"`
	Condition     BranchCondition        `json:"condition"`
	Nodes         []string               `json:"nodes"`         // Node IDs in this branch
	JoinStrategy  string                 `json:"join_strategy"` // "wait_all", "wait_any", "wait_first", "timeout"
	JoinTimeout   time.Duration          `json:"join_timeout,omitempty"`
	MaxConcurrent int                    `json:"max_concurrent"` // Maximum concurrent executions
	Priority      int                    `json:"priority"`       // Execution priority
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt     time.Time              `json:"created_at"`
}

// BranchCondition determines when a branch should execute
type BranchCondition struct {
	Type       string                 `json:"type"`        // "always", "conditional", "data_driven", "event_driven"
	Expression string                 `json:"expression"`  // Condition expression
	InputPaths []string               `json:"input_paths"` // Required input data paths
	EventTypes []string               `json:"event_types"` // Event types that trigger branch
	Parameters map[string]interface{} `json:"parameters,omitempty"`
}

// BranchExecution represents an active branch execution
type BranchExecution struct {
	ID            string                 `json:"id"`
	BranchSpecID  string                 `json:"branch_spec_id"`
	ExecutionID   string                 `json:"execution_id"`
	GraphID       string                 `json:"graph_id"`
	ThreadID      string                 `json:"thread_id"`
	Status        string                 `json:"status"` // "pending", "running", "completed", "failed", "cancelled"
	StartedAt     time.Time              `json:"started_at"`
	CompletedAt   *time.Time             `json:"completed_at,omitempty"`
	Results       map[string]interface{} `json:"results,omitempty"`
	Error         string                 `json:"error,omitempty"`
	ChildBranches []string               `json:"child_branches,omitempty"`
	JoinPointID   string                 `json:"join_point_id,omitempty"`
	Context       map[string]interface{} `json:"context,omitempty"`
}

// JoinPoint manages the synchronization of parallel branches
type JoinPoint struct {
	ID                string                 `json:"id"`
	Name              string                 `json:"name"`
	Strategy          string                 `json:"strategy"` // "wait_all", "wait_any", "wait_first", "timeout"
	Timeout           time.Duration          `json:"timeout,omitempty"`
	ExpectedBranches  []string               `json:"expected_branches"`
	CompletedBranches []string               `json:"completed_branches"`
	Results           map[string]interface{} `json:"results,omitempty"`
	Status            string                 `json:"status"` // "waiting", "completed", "timeout", "failed"
	CreatedAt         time.Time              `json:"created_at"`
	CompletedAt       *time.Time             `json:"completed_at,omitempty"`
	OnCompletion      JoinCompletionHandler  `json:"-"`
}

// JoinCompletionHandler is called when a join point completes
type JoinCompletionHandler func(ctx context.Context, joinPoint *JoinPoint, results map[string]interface{}) error

// BranchConfiguration defines branch execution settings
type BranchConfiguration struct {
	MaxConcurrentBranches int                    `json:"max_concurrent_branches"`
	DefaultJoinStrategy   string                 `json:"default_join_strategy"`
	DefaultJoinTimeout    time.Duration          `json:"default_join_timeout"`
	BranchTemplate        map[string]interface{} `json:"branch_template,omitempty"`
}

// NewBranchManager creates a new branch manager
func NewBranchManager() *BranchManager {
	return &BranchManager{
		branches:       make(map[string]*BranchSpec),
		activeBranches: make(map[string]*BranchExecution),
		joinPoints:     make(map[string]*JoinPoint),
	}
}

// CreateBranchSpec creates a new branch specification
func (bm *BranchManager) CreateBranchSpec(spec *BranchSpec) error {
	if spec.ID == "" {
		spec.ID = uuid.New().String()
	}

	// Validate branch specification
	if err := bm.validateBranchSpec(spec); err != nil {
		return err
	}

	spec.CreatedAt = time.Now()

	bm.mu.Lock()
	bm.branches[spec.ID] = spec
	bm.mu.Unlock()

	return nil
}

// validateBranchSpec validates a branch specification
func (bm *BranchManager) validateBranchSpec(spec *BranchSpec) error {
	if spec.Name == "" {
		return errors.New("branch name is required")
	}

	if len(spec.Nodes) == 0 {
		return errors.New("branch must have at least one node")
	}

	validStrategies := map[string]bool{
		"wait_all":   true,
		"wait_any":   true,
		"wait_first": true,
		"timeout":    true,
	}

	if !validStrategies[spec.JoinStrategy] {
		return errors.New("invalid join strategy")
	}

	if spec.JoinStrategy == "timeout" && spec.JoinTimeout == 0 {
		return errors.New("timeout strategy requires join_timeout")
	}

	return nil
}

// CreateBranch creates a new branch execution from a specification
func (bm *BranchManager) CreateBranch(
	ctx context.Context,
	branchSpecID string,
	executionID string,
	graphID string,
	threadID string,
	branchContext map[string]interface{},
) (*BranchExecution, error) {
	bm.mu.RLock()
	spec, exists := bm.branches[branchSpecID]
	bm.mu.RUnlock()

	if !exists {
		return nil, errors.New("branch specification not found")
	}

	// Evaluate branch condition
	shouldExecute, err := bm.evaluateBranchCondition(&spec.Condition, branchContext)
	if err != nil {
		return nil, err
	}

	if !shouldExecute {
		return nil, fmt.Errorf("branch condition not met for spec %s", branchSpecID)
	}

	// Create branch execution
	branchExec := &BranchExecution{
		ID:           uuid.New().String(),
		BranchSpecID: branchSpecID,
		ExecutionID:  executionID,
		GraphID:      graphID,
		ThreadID:     threadID,
		Status:       "pending",
		StartedAt:    time.Now(),
		Context:      branchContext,
	}

	bm.mu.Lock()
	bm.activeBranches[branchExec.ID] = branchExec
	bm.mu.Unlock()

	return branchExec, nil
}

// evaluateBranchCondition evaluates whether a branch should execute
func (bm *BranchManager) evaluateBranchCondition(
	condition *BranchCondition,
	context map[string]interface{},
) (bool, error) {
	switch condition.Type {
	case "always":
		return true, nil
	case "conditional":
		return bm.evaluateExpression(condition.Expression, context)
	case "data_driven":
		return bm.evaluateDataCondition(condition.InputPaths, context)
	case "event_driven":
		return bm.evaluateEventCondition(condition.EventTypes, context)
	default:
		return false, fmt.Errorf("unknown condition type: %s", condition.Type)
	}
}

// evaluateExpression evaluates a conditional expression
func (bm *BranchManager) evaluateExpression(expression string, context map[string]interface{}) (bool, error) {
	// Simplified expression evaluation - in practice, you might use a proper expression parser
	// For now, we'll do basic string matching and value checks

	if expression == "" {
		return true, nil
	}

	// Example: "input.type == 'parallel'" or "context.enable_branching == true"
	// This is a simplified implementation - use a proper expression evaluator in production
	switch expression {
	case "true":
		return true, nil
	case "false":
		return false, nil
	default:
		// For more complex expressions, integrate with an expression library
		// like github.com/Knetic/govaluate or similar
		return true, nil // Default to true for now
	}
}

// evaluateDataCondition checks if required data paths are available
func (bm *BranchManager) evaluateDataCondition(inputPaths []string, context map[string]interface{}) (bool, error) {
	for _, path := range inputPaths {
		if !bm.hasNestedValue(context, path) {
			return false, nil
		}
	}
	return true, nil
}

// evaluateEventCondition checks if required events are present
func (bm *BranchManager) evaluateEventCondition(eventTypes []string, context map[string]interface{}) (bool, error) {
	events, ok := context["events"].([]interface{})
	if !ok {
		return len(eventTypes) == 0, nil
	}

	eventSet := make(map[string]bool)
	for _, event := range events {
		if eventMap, ok := event.(map[string]interface{}); ok {
			if eventType, ok := eventMap["type"].(string); ok {
				eventSet[eventType] = true
			}
		}
	}

	for _, requiredType := range eventTypes {
		if !eventSet[requiredType] {
			return false, nil
		}
	}

	return true, nil
}

// hasNestedValue checks if a nested path exists in a map
func (bm *BranchManager) hasNestedValue(data map[string]interface{}, path string) bool {
	// Simple path traversal implementation
	// In production, use a proper path library like github.com/tidwall/gjson
	parts := []string{path} // Simplified - split by '.' in practice

	current := data
	for _, part := range parts {
		if value, exists := current[part]; exists {
			if nextMap, ok := value.(map[string]interface{}); ok {
				current = nextMap
			} else {
				return len(parts) == 1 // Last part, value exists
			}
		} else {
			return false
		}
	}
	return true
}

// CreateJoinPoint creates a new join point for branch synchronization
func (bm *BranchManager) CreateJoinPoint(
	name string,
	strategy string,
	expectedBranches []string,
	timeout time.Duration,
	handler JoinCompletionHandler,
) (*JoinPoint, error) {
	joinPoint := &JoinPoint{
		ID:                uuid.New().String(),
		Name:              name,
		Strategy:          strategy,
		Timeout:           timeout,
		ExpectedBranches:  expectedBranches,
		CompletedBranches: make([]string, 0),
		Results:           make(map[string]interface{}),
		Status:            "waiting",
		CreatedAt:         time.Now(),
		OnCompletion:      handler,
	}

	bm.mu.Lock()
	bm.joinPoints[joinPoint.ID] = joinPoint
	bm.mu.Unlock()

	// Start timeout if specified
	if timeout > 0 {
		go bm.handleJoinTimeout(joinPoint)
	}

	return joinPoint, nil
}

// CompleteBranch marks a branch as completed and checks join conditions
func (bm *BranchManager) CompleteBranch(
	ctx context.Context,
	branchID string,
	results map[string]interface{},
	err error,
) error {
	bm.mu.Lock()
	branchExec, exists := bm.activeBranches[branchID]
	if !exists {
		bm.mu.Unlock()
		return errors.New("branch execution not found")
	}

	// Update branch status
	now := time.Now()
	branchExec.CompletedAt = &now
	branchExec.Results = results

	if err != nil {
		branchExec.Status = "failed"
		branchExec.Error = err.Error()
	} else {
		branchExec.Status = "completed"
	}

	// Get join point if exists
	joinPointID := branchExec.JoinPointID
	bm.mu.Unlock()

	// Process join point if this branch is part of one
	if joinPointID != "" {
		return bm.processBranchCompletion(ctx, joinPointID, branchID, results)
	}

	return nil
}

// processBranchCompletion handles branch completion and join point evaluation
func (bm *BranchManager) processBranchCompletion(
	ctx context.Context,
	joinPointID string,
	branchID string,
	results map[string]interface{},
) error {
	bm.mu.Lock()
	joinPoint, exists := bm.joinPoints[joinPointID]
	if !exists {
		bm.mu.Unlock()
		return errors.New("join point not found")
	}

	// Add branch to completed list
	joinPoint.CompletedBranches = append(joinPoint.CompletedBranches, branchID)
	joinPoint.Results[branchID] = results

	// Check if join condition is met
	shouldComplete := bm.evaluateJoinCondition(joinPoint)
	bm.mu.Unlock()

	if shouldComplete {
		return bm.completeJoinPoint(ctx, joinPoint)
	}

	return nil
}

// evaluateJoinCondition determines if a join point should complete
func (bm *BranchManager) evaluateJoinCondition(joinPoint *JoinPoint) bool {
	completed := len(joinPoint.CompletedBranches)
	expected := len(joinPoint.ExpectedBranches)

	switch joinPoint.Strategy {
	case "wait_all":
		return completed >= expected
	case "wait_any":
		return completed > 0
	case "wait_first":
		return completed > 0
	case "timeout":
		// Timeout is handled separately
		return completed >= expected
	default:
		return completed >= expected
	}
}

// completeJoinPoint finalizes a join point and calls completion handler
func (bm *BranchManager) completeJoinPoint(ctx context.Context, joinPoint *JoinPoint) error {
	bm.mu.Lock()
	if joinPoint.Status != "waiting" {
		bm.mu.Unlock()
		return nil // Already completed
	}

	joinPoint.Status = "completed"
	now := time.Now()
	joinPoint.CompletedAt = &now
	bm.mu.Unlock()

	// Call completion handler if provided
	if joinPoint.OnCompletion != nil {
		return joinPoint.OnCompletion(ctx, joinPoint, joinPoint.Results)
	}

	return nil
}

// handleJoinTimeout handles join point timeouts
func (bm *BranchManager) handleJoinTimeout(joinPoint *JoinPoint) {
	time.Sleep(joinPoint.Timeout)

	bm.mu.Lock()
	if joinPoint.Status == "waiting" {
		joinPoint.Status = "timeout"
		now := time.Now()
		joinPoint.CompletedAt = &now
	}
	bm.mu.Unlock()
}

// GetBranchSpec retrieves a branch specification by ID
func (bm *BranchManager) GetBranchSpec(branchSpecID string) (*BranchSpec, error) {
	bm.mu.RLock()
	defer bm.mu.RUnlock()

	spec, exists := bm.branches[branchSpecID]
	if !exists {
		return nil, errors.New("branch specification not found")
	}

	return spec, nil
}

// GetActiveBranch retrieves an active branch execution by ID
func (bm *BranchManager) GetActiveBranch(branchID string) (*BranchExecution, error) {
	bm.mu.RLock()
	defer bm.mu.RUnlock()

	branch, exists := bm.activeBranches[branchID]
	if !exists {
		return nil, errors.New("active branch not found")
	}

	return branch, nil
}

// ListActiveBranches returns all active branch executions
func (bm *BranchManager) ListActiveBranches(executionID string) []*BranchExecution {
	bm.mu.RLock()
	defer bm.mu.RUnlock()

	var branches []*BranchExecution
	for _, branch := range bm.activeBranches {
		if executionID == "" || branch.ExecutionID == executionID {
			branches = append(branches, branch)
		}
	}

	return branches
}

// GetJoinPoint retrieves a join point by ID
func (bm *BranchManager) GetJoinPoint(joinPointID string) (*JoinPoint, error) {
	bm.mu.RLock()
	defer bm.mu.RUnlock()

	joinPoint, exists := bm.joinPoints[joinPointID]
	if !exists {
		return nil, errors.New("join point not found")
	}

	return joinPoint, nil
}

// CancelBranch cancels an active branch execution
func (bm *BranchManager) CancelBranch(branchID string) error {
	bm.mu.Lock()
	defer bm.mu.Unlock()

	branchExec, exists := bm.activeBranches[branchID]
	if !exists {
		return errors.New("branch execution not found")
	}

	branchExec.Status = "cancelled"
	now := time.Now()
	branchExec.CompletedAt = &now

	return nil
}

// CleanupCompletedBranches removes completed branch executions
func (bm *BranchManager) CleanupCompletedBranches(maxAge time.Duration) {
	bm.mu.Lock()
	defer bm.mu.Unlock()

	cutoff := time.Now().Add(-maxAge)
	for id, branch := range bm.activeBranches {
		if branch.CompletedAt != nil && branch.CompletedAt.Before(cutoff) {
			delete(bm.activeBranches, id)
		}
	}
}

// BranchAwareExecutor wraps an executor with branch management capabilities
type BranchAwareExecutor struct {
	baseExecutor  GraphExecutor
	branchManager *BranchManager
}

// NewBranchAwareExecutor creates a branch-aware executor
func NewBranchAwareExecutor(
	baseExecutor GraphExecutor,
	branchManager *BranchManager,
) *BranchAwareExecutor {
	return &BranchAwareExecutor{
		baseExecutor:  baseExecutor,
		branchManager: branchManager,
	}
}

// Execute runs graph execution with branch support
func (be *BranchAwareExecutor) Execute(ctx context.Context, req *dto.ExecutionRequest) (*dto.ExecutionResponse, error) {
	// Add branch manager to context
	ctxWithBranches := context.WithValue(ctx, branchManagerKey("branch_manager"), be.branchManager)

	return be.baseExecutor.Execute(ctxWithBranches, req)
}

// Resume continues execution from a checkpoint
func (be *BranchAwareExecutor) Resume(ctx context.Context, checkpointID string, req *dto.ExecutionRequest) (*dto.ExecutionResponse, error) {
	// Add branch manager to context
	ctxWithBranches := context.WithValue(ctx, branchManagerKey("branch_manager"), be.branchManager)

	return be.baseExecutor.Resume(ctxWithBranches, checkpointID, req)
}

// Stop halts execution
func (be *BranchAwareExecutor) Stop(ctx context.Context, executionID string) error {
	return be.baseExecutor.Stop(ctx, executionID)
}

// GetStatus returns execution status
func (be *BranchAwareExecutor) GetStatus(ctx context.Context, executionID string) (*dto.ExecutionResponse, error) {
	return be.baseExecutor.GetStatus(ctx, executionID)
}

// Define context key type to avoid collisions
type branchManagerKey string

// Helper function to get branch manager from context
func GetBranchManager(ctx context.Context) (*BranchManager, bool) {
	manager, ok := ctx.Value(branchManagerKey("branch_manager")).(*BranchManager)
	return manager, ok
}
