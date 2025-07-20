// Advanced Blueprint Tracking & Logic Analysis
// Deep analysis of Blueprint files for corruption, logic integrity, and dependency tracking

package integrity

import (
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/Telerallc/gamedev-vcs/internal/analyzer"
)

// BlueprintTracker provides deep Blueprint analysis and tracking
type BlueprintTracker struct {
	analyzer           *analyzer.UE5AssetAnalyzer
	logicAnalyzer      *BlueprintLogicAnalyzer
	referenceTracker   *BlueprintReferenceTracker
	corruptionDetector *BlueprintCorruptionDetector
}

// BlueprintAnalysis contains comprehensive Blueprint analysis results
type BlueprintAnalysis struct {
	BlueprintID         string                `json:"blueprint_id"`
	BlueprintClass      string                `json:"blueprint_class"`
	ParentClass         string                `json:"parent_class"`
	BlueprintType       BlueprintType         `json:"blueprint_type"`
	LogicComplexity     int                   `json:"logic_complexity"`
	NodeCount           int                   `json:"node_count"`
	FunctionCount       int                   `json:"function_count"`
	VariableCount       int                   `json:"variable_count"`
	EventCount          int                   `json:"event_count"`
	LogicNodes          []BlueprintNode       `json:"logic_nodes"`
	Functions           []BlueprintFunction   `json:"functions"`
	Variables           []BlueprintVariable   `json:"variables"`
	Events              []BlueprintEvent      `json:"events"`
	Dependencies        []BlueprintDependency `json:"dependencies"`
	References          []BlueprintReference  `json:"references"`
	PotentialIssues     []BlueprintIssue      `json:"potential_issues"`
	HasLogicCorruption  bool                  `json:"has_logic_corruption"`
	HasBrokenReferences bool                  `json:"has_broken_references"`
	HasInfiniteLoops    bool                  `json:"has_infinite_loops"`
	HasUnreachableCode  bool                  `json:"has_unreachable_code"`
	PerformanceRisk     PerformanceRisk       `json:"performance_risk"`
	SecurityRisk        SecurityRisk          `json:"security_risk"`
	MaintenanceRisk     MaintenanceRisk       `json:"maintenance_risk"`
	QualityScore        float64               `json:"quality_score"`
	RecommendedActions  []string              `json:"recommended_actions"`
}

// BlueprintNode represents a node in the Blueprint graph
type BlueprintNode struct {
	NodeID            string                 `json:"node_id"`
	NodeType          string                 `json:"node_type"`
	NodeClass         string                 `json:"node_class"`
	FunctionName      string                 `json:"function_name"`
	Position          NodePosition           `json:"position"`
	Connections       []NodeConnection       `json:"connections"`
	Properties        map[string]interface{} `json:"properties"`
	IsValid           bool                   `json:"is_valid"`
	ErrorMessage      string                 `json:"error_message"`
	PerformanceImpact string                 `json:"performance_impact"`
}

// BlueprintFunction represents a custom function in the Blueprint
type BlueprintFunction struct {
	FunctionName   string              `json:"function_name"`
	FunctionType   string              `json:"function_type"`
	Parameters     []FunctionParameter `json:"parameters"`
	ReturnType     string              `json:"return_type"`
	AccessModifier string              `json:"access_modifier"`
	IsEvent        bool                `json:"is_event"`
	IsPure         bool                `json:"is_pure"`
	NodeCount      int                 `json:"node_count"`
	Complexity     int                 `json:"complexity"`
	CallCount      int                 `json:"call_count"`
	IsRecursive    bool                `json:"is_recursive"`
	HasSideEffects bool                `json:"has_side_effects"`
}

// BlueprintVariable represents a variable in the Blueprint
type BlueprintVariable struct {
	VariableName    string      `json:"variable_name"`
	VariableType    string      `json:"variable_type"`
	DefaultValue    interface{} `json:"default_value"`
	IsPublic        bool        `json:"is_public"`
	IsEditable      bool        `json:"is_editable"`
	Category        string      `json:"category"`
	Tooltip         string      `json:"tooltip"`
	ReplicationMode string      `json:"replication_mode"`
	UsageCount      int         `json:"usage_count"`
	IsUnused        bool        `json:"is_unused"`
}

// BlueprintEvent represents an event in the Blueprint
type BlueprintEvent struct {
	EventName       string   `json:"event_name"`
	EventType       string   `json:"event_type"`
	TriggerType     string   `json:"trigger_type"`
	Parameters      []string `json:"parameters"`
	IsCustomEvent   bool     `json:"is_custom_event"`
	IsNetworked     bool     `json:"is_networked"`
	CallFrequency   string   `json:"call_frequency"`
	PerformanceRisk string   `json:"performance_risk"`
}

// BlueprintDependency represents a dependency on another Blueprint or asset
type BlueprintDependency struct {
	DependencyPath   string           `json:"dependency_path"`
	DependencyType   string           `json:"dependency_type"`
	ReferenceType    string           `json:"reference_type"`
	UsageContext     string           `json:"usage_context"`
	IsRequired       bool             `json:"is_required"`
	IsSoftReference  bool             `json:"is_soft_reference"`
	LoadBehavior     string           `json:"load_behavior"`
	ValidationStatus ValidationStatus `json:"validation_status"`
	LastValidated    time.Time        `json:"last_validated"`
}

// BlueprintReference represents a reference within the Blueprint logic
type BlueprintReference struct {
	SourceNode      string    `json:"source_node"`
	TargetAsset     string    `json:"target_asset"`
	ReferenceType   string    `json:"reference_type"`
	PropertyName    string    `json:"property_name"`
	IsValid         bool      `json:"is_valid"`
	ValidationError string    `json:"validation_error"`
	LastChecked     time.Time `json:"last_checked"`
}

// BlueprintIssue represents a potential problem in the Blueprint
type BlueprintIssue struct {
	IssueID           string        `json:"issue_id"`
	IssueType         IssueType     `json:"issue_type"`
	Severity          IssueSeverity `json:"severity"`
	Description       string        `json:"description"`
	Location          string        `json:"location"`
	NodeID            string        `json:"node_id"`
	FunctionName      string        `json:"function_name"`
	RecommendedFix    string        `json:"recommended_fix"`
	AutoFixable       bool          `json:"auto_fixable"`
	PerformanceImpact string        `json:"performance_impact"`
}

// Supporting structures
type NodePosition struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

type NodeConnection struct {
	TargetNodeID   string `json:"target_node_id"`
	ConnectionType string `json:"connection_type"`
	PinName        string `json:"pin_name"`
	IsValid        bool   `json:"is_valid"`
}

type FunctionParameter struct {
	ParameterName string `json:"parameter_name"`
	ParameterType string `json:"parameter_type"`
	DefaultValue  string `json:"default_value"`
	IsOptional    bool   `json:"is_optional"`
}

// Enums
type BlueprintType string

const (
	BlueprintTypeActor     BlueprintType = "Actor"
	BlueprintTypePawn      BlueprintType = "Pawn"
	BlueprintTypeCharacter BlueprintType = "Character"
	BlueprintTypeGameMode  BlueprintType = "GameMode"
	BlueprintTypeWidget    BlueprintType = "Widget"
	BlueprintTypeComponent BlueprintType = "Component"
	BlueprintTypeInterface BlueprintType = "Interface"
	BlueprintTypeFunction  BlueprintType = "Function"
	BlueprintTypeMacro     BlueprintType = "Macro"
	BlueprintTypeUnknown   BlueprintType = "Unknown"
)

type PerformanceRisk string

const (
	PerformanceLow      PerformanceRisk = "low"
	PerformanceMedium   PerformanceRisk = "medium"
	PerformanceHigh     PerformanceRisk = "high"
	PerformanceCritical PerformanceRisk = "critical"
)

type SecurityRisk string

const (
	SecurityLow      SecurityRisk = "low"
	SecurityMedium   SecurityRisk = "medium"
	SecurityHigh     SecurityRisk = "high"
	SecurityCritical SecurityRisk = "critical"
)

type MaintenanceRisk string

const (
	MaintenanceLow      MaintenanceRisk = "low"
	MaintenanceMedium   MaintenanceRisk = "medium"
	MaintenanceHigh     MaintenanceRisk = "high"
	MaintenanceCritical MaintenanceRisk = "critical"
)

type IssueType string

const (
	IssueLogicError      IssueType = "logic_error"
	IssueBrokenReference IssueType = "broken_reference"
	IssueInfiniteLoop    IssueType = "infinite_loop"
	IssueUnreachableCode IssueType = "unreachable_code"
	IssuePerformance     IssueType = "performance"
	IssueDeprecated      IssueType = "deprecated"
	IssueSecurity        IssueType = "security"
	IssueComplexity      IssueType = "complexity"
	IssueUnusedVariable  IssueType = "unused_variable"
	IssueDeadCode        IssueType = "dead_code"
)

type IssueSeverity string

const (
	BlueprintSeverityInfo     IssueSeverity = "info"
	BlueprintSeverityWarning  IssueSeverity = "warning"
	BlueprintSeverityError    IssueSeverity = "error"
	BlueprintSeverityCritical IssueSeverity = "critical"
)

// NewBlueprintTracker creates a new Blueprint tracking system
func NewBlueprintTracker() *BlueprintTracker {
	return &BlueprintTracker{
		analyzer:           analyzer.NewUE5AssetAnalyzer(),
		logicAnalyzer:      NewBlueprintLogicAnalyzer(),
		referenceTracker:   NewBlueprintReferenceTracker(),
		corruptionDetector: NewBlueprintCorruptionDetector(),
	}
}

// AnalyzeBlueprint performs comprehensive Blueprint analysis
func (bt *BlueprintTracker) AnalyzeBlueprint(content []byte) (*BlueprintAnalysis, error) {
	analysis := &BlueprintAnalysis{
		LogicNodes:         []BlueprintNode{},
		Functions:          []BlueprintFunction{},
		Variables:          []BlueprintVariable{},
		Events:             []BlueprintEvent{},
		Dependencies:       []BlueprintDependency{},
		References:         []BlueprintReference{},
		PotentialIssues:    []BlueprintIssue{},
		RecommendedActions: []string{},
	}

	// Parse Blueprint header and metadata
	if err := bt.parseBlueprintHeader(content, analysis); err != nil {
		return nil, fmt.Errorf("failed to parse Blueprint header: %w", err)
	}

	// Extract and analyze Blueprint logic graph
	if err := bt.analyzeLogicGraph(content, analysis); err != nil {
		return nil, fmt.Errorf("failed to analyze logic graph: %w", err)
	}

	// Analyze functions and events
	if err := bt.analyzeFunctions(content, analysis); err != nil {
		return nil, fmt.Errorf("failed to analyze functions: %w", err)
	}

	// Analyze variables
	if err := bt.analyzeVariables(content, analysis); err != nil {
		return nil, fmt.Errorf("failed to analyze variables: %w", err)
	}

	// Track dependencies and references
	if err := bt.analyzeDependencies(content, analysis); err != nil {
		return nil, fmt.Errorf("failed to analyze dependencies: %w", err)
	}

	// Detect corruption and integrity issues
	bt.detectCorruption(analysis)

	// Perform quality analysis
	bt.analyzeQuality(analysis)

	// Calculate risks and scores
	bt.calculateRisks(analysis)

	// Generate recommendations
	bt.generateRecommendations(analysis)

	return analysis, nil
}

// parseBlueprintHeader extracts basic Blueprint information
func (bt *BlueprintTracker) parseBlueprintHeader(content []byte, analysis *BlueprintAnalysis) error {
	contentStr := string(content)

	// Extract Blueprint class information
	if match := regexp.MustCompile(`BlueprintGeneratedClass\s*['"](.*?)['"]`).FindStringSubmatch(contentStr); len(match) > 1 {
		analysis.BlueprintClass = match[1]
	}

	// Extract parent class
	if match := regexp.MustCompile(`ParentClass\s*['"](.*?)['"]`).FindStringSubmatch(contentStr); len(match) > 1 {
		analysis.ParentClass = match[1]
	}

	// Determine Blueprint type
	analysis.BlueprintType = bt.determineBlueprintType(analysis.BlueprintClass, analysis.ParentClass)

	// Generate unique Blueprint ID
	analysis.BlueprintID = bt.generateBlueprintID(content)

	return nil
}

// analyzeLogicGraph extracts and analyzes the Blueprint node graph
func (bt *BlueprintTracker) analyzeLogicGraph(content []byte, analysis *BlueprintAnalysis) error {
	// Extract node information from Blueprint data
	nodes, err := bt.extractNodes(content)
	if err != nil {
		return err
	}

	analysis.LogicNodes = nodes
	analysis.NodeCount = len(nodes)

	// Analyze node connections and flow
	bt.analyzeNodeConnections(nodes, analysis)

	// Detect logic issues
	bt.detectLogicIssues(nodes, analysis)

	return nil
}

// extractNodes parses Blueprint content to extract node information
func (bt *BlueprintTracker) extractNodes(content []byte) ([]BlueprintNode, error) {
	var nodes []BlueprintNode
	contentStr := string(content)

	// Look for node patterns in the Blueprint data
	nodePattern := regexp.MustCompile(`K2Node_(\w+)`)
	matches := nodePattern.FindAllStringSubmatch(contentStr, -1)

	for i, match := range matches {
		if len(match) > 1 {
			node := BlueprintNode{
				NodeID:      fmt.Sprintf("node_%d", i),
				NodeType:    match[1],
				NodeClass:   fmt.Sprintf("K2Node_%s", match[1]),
				IsValid:     true,
				Connections: []NodeConnection{},
				Properties:  make(map[string]interface{}),
			}

			// Analyze node type for performance impact
			node.PerformanceImpact = bt.analyzeNodePerformance(node.NodeType)

			nodes = append(nodes, node)
		}
	}

	return nodes, nil
}

// analyzeNodeConnections traces connections between nodes
func (bt *BlueprintTracker) analyzeNodeConnections(nodes []BlueprintNode, analysis *BlueprintAnalysis) {
	// Build connection graph
	connectionMap := make(map[string][]string)

	for _, node := range nodes {
		for _, conn := range node.Connections {
			connectionMap[node.NodeID] = append(connectionMap[node.NodeID], conn.TargetNodeID)
		}
	}

	// Detect infinite loops
	analysis.HasInfiniteLoops = bt.detectInfiniteLoops(connectionMap)

	// Detect unreachable code
	analysis.HasUnreachableCode = bt.detectUnreachableCode(connectionMap)

	// Calculate logic complexity
	analysis.LogicComplexity = bt.calculateLogicComplexity(connectionMap)
}

// detectLogicIssues identifies various logic problems
func (bt *BlueprintTracker) detectLogicIssues(nodes []BlueprintNode, analysis *BlueprintAnalysis) {
	for _, node := range nodes {
		// Check for deprecated nodes
		if bt.isDeprecatedNode(node.NodeType) {
			issue := BlueprintIssue{
				IssueID:        fmt.Sprintf("deprecated_%s", node.NodeID),
				IssueType:      IssueDeprecated,
				Severity:       BlueprintSeverityWarning,
				Description:    fmt.Sprintf("Node %s is deprecated", node.NodeType),
				Location:       node.NodeID,
				NodeID:         node.NodeID,
				RecommendedFix: "Replace with modern equivalent",
				AutoFixable:    false,
			}
			analysis.PotentialIssues = append(analysis.PotentialIssues, issue)
		}

		// Check for performance issues
		if node.PerformanceImpact == "high" || node.PerformanceImpact == "critical" {
			issue := BlueprintIssue{
				IssueID:           fmt.Sprintf("performance_%s", node.NodeID),
				IssueType:         IssuePerformance,
				Severity:          BlueprintSeverityWarning,
				Description:       fmt.Sprintf("Node %s has high performance impact", node.NodeType),
				Location:          node.NodeID,
				NodeID:            node.NodeID,
				RecommendedFix:    "Consider optimizing or moving to C++",
				AutoFixable:       false,
				PerformanceImpact: node.PerformanceImpact,
			}
			analysis.PotentialIssues = append(analysis.PotentialIssues, issue)
		}
	}
}

// analyzeFunctions extracts and analyzes Blueprint functions
func (bt *BlueprintTracker) analyzeFunctions(content []byte, analysis *BlueprintAnalysis) error {
	contentStr := string(content)

	// Extract function definitions
	functionPattern := regexp.MustCompile(`UserDefinedFunction\s*['"](.*?)['"]`)
	matches := functionPattern.FindAllStringSubmatch(contentStr, -1)

	for _, match := range matches {
		if len(match) > 1 {
			function := BlueprintFunction{
				FunctionName:   match[1],
				FunctionType:   "UserDefined",
				Parameters:     []FunctionParameter{},
				AccessModifier: "Public",
				NodeCount:      bt.countFunctionNodes(contentStr, match[1]),
			}

			// Calculate function complexity
			function.Complexity = bt.calculateFunctionComplexity(function)

			// Check if function is recursive
			function.IsRecursive = bt.detectRecursion(contentStr, match[1])

			analysis.Functions = append(analysis.Functions, function)
		}
	}

	analysis.FunctionCount = len(analysis.Functions)
	return nil
}

// analyzeVariables extracts and analyzes Blueprint variables
func (bt *BlueprintTracker) analyzeVariables(content []byte, analysis *BlueprintAnalysis) error {
	contentStr := string(content)

	// Extract variable definitions
	variablePattern := regexp.MustCompile(`VariableName\s*['"](.*?)['"]`)
	matches := variablePattern.FindAllStringSubmatch(contentStr, -1)

	for _, match := range matches {
		if len(match) > 1 {
			variable := BlueprintVariable{
				VariableName: match[1],
				VariableType: bt.extractVariableType(contentStr, match[1]),
				IsPublic:     bt.isPublicVariable(contentStr, match[1]),
				UsageCount:   bt.countVariableUsage(contentStr, match[1]),
			}

			// Mark unused variables
			variable.IsUnused = variable.UsageCount == 0

			if variable.IsUnused {
				issue := BlueprintIssue{
					IssueID:        fmt.Sprintf("unused_var_%s", variable.VariableName),
					IssueType:      IssueUnusedVariable,
					Severity:       BlueprintSeverityInfo,
					Description:    fmt.Sprintf("Variable %s is unused", variable.VariableName),
					RecommendedFix: "Remove unused variable",
					AutoFixable:    true,
				}
				analysis.PotentialIssues = append(analysis.PotentialIssues, issue)
			}

			analysis.Variables = append(analysis.Variables, variable)
		}
	}

	analysis.VariableCount = len(analysis.Variables)
	return nil
}

// analyzeDependencies extracts and validates dependencies
func (bt *BlueprintTracker) analyzeDependencies(content []byte, analysis *BlueprintAnalysis) error {
	// Use existing asset analyzer for basic dependencies
	assetInfo, err := bt.analyzer.AnalyzeAsset("", content)
	if err != nil {
		return err
	}

	// Convert to Blueprint-specific dependencies
	for _, dep := range assetInfo.Dependencies {
		bpDep := BlueprintDependency{
			DependencyPath:   dep.TargetAsset,
			DependencyType:   string(dep.DependencyType),
			ReferenceType:    "AssetReference",
			IsRequired:       dep.DependencyType == "HardReference",
			IsSoftReference:  dep.DependencyType == "SoftReference",
			ValidationStatus: ValidationPending,
			LastValidated:    time.Now(),
		}

		analysis.Dependencies = append(analysis.Dependencies, bpDep)
	}

	// Validate each dependency
	for i := range analysis.Dependencies {
		bt.validateDependency(&analysis.Dependencies[i])
		if analysis.Dependencies[i].ValidationStatus == ValidationInvalid {
			analysis.HasBrokenReferences = true
		}
	}

	return nil
}

// detectCorruption identifies various types of corruption
func (bt *BlueprintTracker) detectCorruption(analysis *BlueprintAnalysis) {
	// Check for structural corruption
	if analysis.NodeCount == 0 && analysis.FunctionCount > 0 {
		analysis.HasLogicCorruption = true
		issue := BlueprintIssue{
			IssueID:        "structural_corruption",
			IssueType:      IssueLogicError,
			Severity:       BlueprintSeverityCritical,
			Description:    "Blueprint structure appears corrupted - functions exist but no nodes found",
			RecommendedFix: "Restore from backup or rebuild Blueprint",
			AutoFixable:    false,
		}
		analysis.PotentialIssues = append(analysis.PotentialIssues, issue)
	}

	// Check for missing parent class
	if analysis.ParentClass == "" && analysis.BlueprintType != BlueprintTypeFunction {
		issue := BlueprintIssue{
			IssueID:        "missing_parent",
			IssueType:      IssueLogicError,
			Severity:       BlueprintSeverityError,
			Description:    "Blueprint missing parent class reference",
			RecommendedFix: "Restore parent class reference",
			AutoFixable:    false,
		}
		analysis.PotentialIssues = append(analysis.PotentialIssues, issue)
	}
}

// AttemptRepair tries to automatically fix Blueprint issues
func (bt *BlueprintTracker) AttemptRepair(assetPath string) error {
	// Read Blueprint file
	content, err := os.ReadFile(assetPath)
	if err != nil {
		return fmt.Errorf("failed to read Blueprint: %w", err)
	}

	// Analyze to find fixable issues
	analysis, err := bt.AnalyzeBlueprint(content)
	if err != nil {
		return fmt.Errorf("failed to analyze Blueprint: %w", err)
	}

	repairMade := false
	newContent := content

	// Attempt to fix auto-fixable issues
	for _, issue := range analysis.PotentialIssues {
		if issue.AutoFixable {
			switch issue.IssueType {
			case IssueUnusedVariable:
				newContent = bt.removeUnusedVariable(newContent, issue.Location)
				repairMade = true
			case IssueDeadCode:
				newContent = bt.removeDeadCode(newContent, issue.NodeID)
				repairMade = true
			}
		}
	}

	// Write repaired content back if repairs were made
	if repairMade {
		return os.WriteFile(assetPath, newContent, 0644)
	}

	return fmt.Errorf("no auto-fixable issues found")
}

// Helper methods for analysis

func (bt *BlueprintTracker) determineBlueprintType(blueprintClass, parentClass string) BlueprintType {
	if strings.Contains(strings.ToLower(parentClass), "character") {
		return BlueprintTypeCharacter
	}
	if strings.Contains(strings.ToLower(parentClass), "pawn") {
		return BlueprintTypePawn
	}
	if strings.Contains(strings.ToLower(parentClass), "actor") {
		return BlueprintTypeActor
	}
	if strings.Contains(strings.ToLower(parentClass), "gamemode") {
		return BlueprintTypeGameMode
	}
	if strings.Contains(strings.ToLower(parentClass), "widget") {
		return BlueprintTypeWidget
	}
	return BlueprintTypeUnknown
}

func (bt *BlueprintTracker) generateBlueprintID(content []byte) string {
	// Generate a unique ID based on content hash
	hash := fmt.Sprintf("%x", content[:min(len(content), 32)])
	return fmt.Sprintf("bp_%s", hash)
}

func (bt *BlueprintTracker) analyzeNodePerformance(nodeType string) string {
	heavyNodes := []string{"CallFunction", "ForEach", "WhileLoop", "Delay"}
	mediumNodes := []string{"Branch", "Sequence", "Gate"}

	for _, heavy := range heavyNodes {
		if strings.Contains(nodeType, heavy) {
			return "high"
		}
	}

	for _, medium := range mediumNodes {
		if strings.Contains(nodeType, medium) {
			return "medium"
		}
	}

	return "low"
}

func (bt *BlueprintTracker) detectInfiniteLoops(connectionMap map[string][]string) bool {
	visited := make(map[string]bool)
	recStack := make(map[string]bool)

	for node := range connectionMap {
		if bt.hasCycleDFS(node, connectionMap, visited, recStack) {
			return true
		}
	}

	return false
}

func (bt *BlueprintTracker) hasCycleDFS(node string, graph map[string][]string, visited, recStack map[string]bool) bool {
	visited[node] = true
	recStack[node] = true

	for _, neighbor := range graph[node] {
		if !visited[neighbor] {
			if bt.hasCycleDFS(neighbor, graph, visited, recStack) {
				return true
			}
		} else if recStack[neighbor] {
			return true
		}
	}

	recStack[node] = false
	return false
}

func (bt *BlueprintTracker) detectUnreachableCode(connectionMap map[string][]string) bool {
	// Find entry points (nodes with no incoming connections)
	hasIncoming := make(map[string]bool)
	for _, targets := range connectionMap {
		for _, target := range targets {
			hasIncoming[target] = true
		}
	}

	// Start DFS from entry points
	reachable := make(map[string]bool)
	for node := range connectionMap {
		if !hasIncoming[node] {
			bt.markReachableDFS(node, connectionMap, reachable)
		}
	}

	// Check if any nodes are unreachable
	for node := range connectionMap {
		if !reachable[node] {
			return true
		}
	}

	return false
}

func (bt *BlueprintTracker) markReachableDFS(node string, graph map[string][]string, reachable map[string]bool) {
	reachable[node] = true
	for _, neighbor := range graph[node] {
		if !reachable[neighbor] {
			bt.markReachableDFS(neighbor, graph, reachable)
		}
	}
}

func (bt *BlueprintTracker) calculateLogicComplexity(connectionMap map[string][]string) int {
	// Simplified cyclomatic complexity calculation
	edges := 0
	for _, targets := range connectionMap {
		edges += len(targets)
	}
	nodes := len(connectionMap)

	// Cyclomatic complexity = E - N + 2P (P = connected components, simplified to 1)
	complexity := edges - nodes + 2
	if complexity < 1 {
		complexity = 1
	}

	return complexity
}

func (bt *BlueprintTracker) isDeprecatedNode(nodeType string) bool {
	deprecatedNodes := []string{"OldFunction", "DeprecatedNode", "LegacyCall"}
	for _, deprecated := range deprecatedNodes {
		if strings.Contains(nodeType, deprecated) {
			return true
		}
	}
	return false
}

func (bt *BlueprintTracker) countFunctionNodes(content, functionName string) int {
	// Count nodes in a specific function (simplified)
	pattern := fmt.Sprintf(`%s.*?K2Node`, regexp.QuoteMeta(functionName))
	matches := regexp.MustCompile(pattern).FindAllString(content, -1)
	return len(matches)
}

func (bt *BlueprintTracker) calculateFunctionComplexity(function BlueprintFunction) int {
	// Base complexity
	complexity := 1

	// Add complexity for parameters
	complexity += len(function.Parameters)

	// Add complexity for node count
	complexity += function.NodeCount / 10

	// Recursive functions are more complex
	if function.IsRecursive {
		complexity += 5
	}

	return complexity
}

func (bt *BlueprintTracker) detectRecursion(content, functionName string) bool {
	// Look for function calling itself
	pattern := fmt.Sprintf(`%s.*?CallFunction.*?%s`, regexp.QuoteMeta(functionName), regexp.QuoteMeta(functionName))
	matched, _ := regexp.MatchString(pattern, content)
	return matched
}

func (bt *BlueprintTracker) extractVariableType(content, varName string) string {
	pattern := fmt.Sprintf(`%s.*?VariableType.*?["'](.*?)["']`, regexp.QuoteMeta(varName))
	matches := regexp.MustCompile(pattern).FindStringSubmatch(content)
	if len(matches) > 1 {
		return matches[1]
	}
	return "Unknown"
}

func (bt *BlueprintTracker) isPublicVariable(content, varName string) bool {
	pattern := fmt.Sprintf(`%s.*?bExposeOnSpawn.*?true`, regexp.QuoteMeta(varName))
	matched, _ := regexp.MatchString(pattern, content)
	return matched
}

func (bt *BlueprintTracker) countVariableUsage(content, varName string) int {
	matches := regexp.MustCompile(regexp.QuoteMeta(varName)).FindAllString(content, -1)
	return len(matches) - 1 // Subtract 1 for the declaration
}

func (bt *BlueprintTracker) validateDependency(dep *BlueprintDependency) {
	// Simplified validation - check if file exists
	if _, err := os.Stat(dep.DependencyPath); os.IsNotExist(err) {
		dep.ValidationStatus = ValidationMissing
	} else {
		dep.ValidationStatus = ValidationValid
	}
	dep.LastValidated = time.Now()
}

func (bt *BlueprintTracker) analyzeQuality(analysis *BlueprintAnalysis) {
	score := 100.0

	// Reduce score for issues
	for _, issue := range analysis.PotentialIssues {
		switch issue.Severity {
		case BlueprintSeverityCritical:
			score -= 25
		case BlueprintSeverityError:
			score -= 15
		case BlueprintSeverityWarning:
			score -= 5
		case BlueprintSeverityInfo:
			score -= 1
		}
	}

	// Reduce score for complexity
	if analysis.LogicComplexity > 20 {
		score -= float64(analysis.LogicComplexity-20) * 2
	}

	// Reduce score for too many nodes
	if analysis.NodeCount > 100 {
		score -= float64(analysis.NodeCount-100) * 0.1
	}

	if score < 0 {
		score = 0
	}

	analysis.QualityScore = score
}

func (bt *BlueprintTracker) calculateRisks(analysis *BlueprintAnalysis) {
	// Performance risk
	highPerfNodes := 0
	for _, node := range analysis.LogicNodes {
		if node.PerformanceImpact == "high" || node.PerformanceImpact == "critical" {
			highPerfNodes++
		}
	}

	if highPerfNodes > 10 || analysis.LogicComplexity > 30 {
		analysis.PerformanceRisk = PerformanceCritical
	} else if highPerfNodes > 5 || analysis.LogicComplexity > 20 {
		analysis.PerformanceRisk = PerformanceHigh
	} else if highPerfNodes > 2 || analysis.LogicComplexity > 10 {
		analysis.PerformanceRisk = PerformanceMedium
	} else {
		analysis.PerformanceRisk = PerformanceLow
	}

	// Security risk (simplified)
	if analysis.HasBrokenReferences {
		analysis.SecurityRisk = SecurityHigh
	} else {
		analysis.SecurityRisk = SecurityLow
	}

	// Maintenance risk
	if analysis.QualityScore < 50 || len(analysis.PotentialIssues) > 10 {
		analysis.MaintenanceRisk = MaintenanceHigh
	} else if analysis.QualityScore < 75 || len(analysis.PotentialIssues) > 5 {
		analysis.MaintenanceRisk = MaintenanceMedium
	} else {
		analysis.MaintenanceRisk = MaintenanceLow
	}
}

func (bt *BlueprintTracker) generateRecommendations(analysis *BlueprintAnalysis) {
	if analysis.PerformanceRisk == PerformanceHigh || analysis.PerformanceRisk == PerformanceCritical {
		analysis.RecommendedActions = append(analysis.RecommendedActions, "Consider moving performance-critical logic to C++")
	}

	if analysis.LogicComplexity > 20 {
		analysis.RecommendedActions = append(analysis.RecommendedActions, "Break down complex functions into smaller, more manageable pieces")
	}

	if analysis.HasBrokenReferences {
		analysis.RecommendedActions = append(analysis.RecommendedActions, "Fix broken asset references to prevent runtime errors")
	}

	if analysis.HasInfiniteLoops {
		analysis.RecommendedActions = append(analysis.RecommendedActions, "Review and fix infinite loop conditions")
	}

	if analysis.HasUnreachableCode {
		analysis.RecommendedActions = append(analysis.RecommendedActions, "Remove unreachable code to improve maintainability")
	}

	unusedVarCount := 0
	for _, issue := range analysis.PotentialIssues {
		if issue.IssueType == IssueUnusedVariable {
			unusedVarCount++
		}
	}

	if unusedVarCount > 3 {
		analysis.RecommendedActions = append(analysis.RecommendedActions, "Clean up unused variables to reduce clutter")
	}
}

func (bt *BlueprintTracker) removeUnusedVariable(content []byte, varName string) []byte {
	// Simplified variable removal (in practice, would need proper Blueprint parsing)
	pattern := fmt.Sprintf(`VariableName\s*["']%s["'].*?\n`, regexp.QuoteMeta(varName))
	re := regexp.MustCompile(pattern)
	return re.ReplaceAll(content, []byte(""))
}

func (bt *BlueprintTracker) removeDeadCode(content []byte, nodeID string) []byte {
	// Simplified dead code removal
	pattern := fmt.Sprintf(`%s.*?\n`, regexp.QuoteMeta(nodeID))
	re := regexp.MustCompile(pattern)
	return re.ReplaceAll(content, []byte(""))
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// Supporting analyzer classes would be implemented here...
type BlueprintLogicAnalyzer struct{}

func NewBlueprintLogicAnalyzer() *BlueprintLogicAnalyzer { return &BlueprintLogicAnalyzer{} }

type BlueprintReferenceTracker struct{}

func NewBlueprintReferenceTracker() *BlueprintReferenceTracker { return &BlueprintReferenceTracker{} }

type BlueprintCorruptionDetector struct{}

func NewBlueprintCorruptionDetector() *BlueprintCorruptionDetector {
	return &BlueprintCorruptionDetector{}
}
