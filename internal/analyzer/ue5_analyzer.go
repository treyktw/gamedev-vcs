package analyzer

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
)

// UE5AssetAnalyzer analyzes Unreal Engine 5 assets for dependencies and metadata
type UE5AssetAnalyzer struct {
	packagePrefixMap map[string]string
	classTypeMap     map[string]AssetType
}

// AssetType represents different types of UE5 assets
type AssetType string

const (
	AssetTypeBlueprint        AssetType = "Blueprint"
	AssetTypeStaticMesh       AssetType = "StaticMesh"
	AssetTypeSkeletalMesh     AssetType = "SkeletalMesh"
	AssetTypeTexture2D        AssetType = "Texture2D"
	AssetTypeMaterial         AssetType = "Material"
	AssetTypeMaterialInstance AssetType = "MaterialInstance"
	AssetTypeSound            AssetType = "SoundWave"
	AssetTypeAnimation        AssetType = "AnimSequence"
	AssetTypeLevel            AssetType = "Level"
	AssetTypeWidget           AssetType = "UserWidget"
	AssetTypeDataAsset        AssetType = "DataAsset"
	AssetTypeUnknown          AssetType = "Unknown"
)

// DependencyType represents the strength of asset dependencies
type DependencyType string

const (
	DependencyHard       DependencyType = "HardReference"
	DependencySoft       DependencyType = "SoftReference"
	DependencySearchable DependencyType = "SearchableReference"
)

// AssetInfo contains metadata about a UE5 asset
type AssetInfo struct {
	FilePath          string                 `json:"file_path"`
	AssetName         string                 `json:"asset_name"`
	AssetType         AssetType              `json:"asset_type"`
	PackageName       string                 `json:"package_name"`
	AssetClass        string                 `json:"asset_class"`
	ParentClass       string                 `json:"parent_class"`
	IsBlueprint       bool                   `json:"is_blueprint"`
	BlueprintType     string                 `json:"blueprint_type"`
	Dependencies      []AssetDependency      `json:"dependencies"`
	Properties        map[string]interface{} `json:"properties"`
	Complexity        int                    `json:"complexity"`
	EstimatedLoadTime float64                `json:"estimated_load_time_ms"`
}

// AssetDependency represents a dependency relationship
type AssetDependency struct {
	SourceAsset    string         `json:"source_asset"`
	TargetAsset    string         `json:"target_asset"`
	DependencyType DependencyType `json:"dependency_type"`
	PropertyName   string         `json:"property_name,omitempty"`
	IsCircular     bool           `json:"is_circular"`
	Weight         float64        `json:"weight"`
}

// UAssetHeader represents the header of a .uasset file
type UAssetHeader struct {
	Magic           uint32
	FileVersion     int32
	LicenseeVersion int32
	PackageFlags    uint32
	NameCount       int32
	NameOffset      int32
	ExportCount     int32
	ExportOffset    int32
	ImportCount     int32
	ImportOffset    int32
}

// NewUE5AssetAnalyzer creates a new UE5 asset analyzer
func NewUE5AssetAnalyzer() *UE5AssetAnalyzer {
	analyzer := &UE5AssetAnalyzer{
		packagePrefixMap: make(map[string]string),
		classTypeMap:     make(map[string]AssetType),
	}

	// Initialize common UE5 class mappings
	analyzer.initializeClassMappings()

	return analyzer
}

// AnalyzeAsset analyzes a UE5 asset file and extracts dependencies
func (ua *UE5AssetAnalyzer) AnalyzeAsset(filePath string, content []byte) (*AssetInfo, error) {
	assetInfo := &AssetInfo{
		FilePath:     filePath,
		AssetName:    ua.extractAssetName(filePath),
		Properties:   make(map[string]interface{}),
		Dependencies: make([]AssetDependency, 0),
	}

	// Determine asset type based on file extension and content
	assetInfo.AssetType = ua.determineAssetType(filePath, content)

	// Extract package information
	assetInfo.PackageName = ua.extractPackageName(filePath)

	// Parse based on file type
	switch {
	case strings.HasSuffix(filePath, ".uasset"):
		return ua.analyzeUAsset(assetInfo, content)
	case strings.HasSuffix(filePath, ".umap"):
		return ua.analyzeUMap(assetInfo, content)
	case strings.HasSuffix(filePath, ".uexp"):
		// .uexp files are parsed together with .uasset
		return assetInfo, nil
	default:
		// For other UE5-related files, do basic analysis
		return ua.analyzeGenericAsset(assetInfo, content)
	}
}

// ExtractDependencies extracts dependencies from asset content
func (ua *UE5AssetAnalyzer) ExtractDependencies(assetPath string, content []byte) ([]AssetDependency, error) {
	assetInfo, err := ua.AnalyzeAsset(assetPath, content)
	if err != nil {
		return nil, err
	}

	return assetInfo.Dependencies, nil
}

// ValidateAssetIntegrity checks if an asset's dependencies are intact
func (ua *UE5AssetAnalyzer) ValidateAssetIntegrity(assetInfo *AssetInfo, availableAssets map[string]bool) []string {
	var missingDependencies []string

	for _, dep := range assetInfo.Dependencies {
		if !availableAssets[dep.TargetAsset] {
			missingDependencies = append(missingDependencies, dep.TargetAsset)
		}
	}

	return missingDependencies
}

// CalculateComplexity estimates the complexity of an asset
func (ua *UE5AssetAnalyzer) CalculateComplexity(assetInfo *AssetInfo) int {
	complexity := 0

	// Base complexity from dependencies
	complexity += len(assetInfo.Dependencies) * 2

	// Asset type specific complexity
	switch assetInfo.AssetType {
	case AssetTypeBlueprint:
		complexity += 10 // Blueprints are inherently complex
	case AssetTypeLevel:
		complexity += 20 // Levels are very complex
	case AssetTypeMaterial:
		complexity += 5
	case AssetTypeStaticMesh, AssetTypeSkeletalMesh:
		complexity += 3
	}

	// Hard dependencies increase complexity more
	for _, dep := range assetInfo.Dependencies {
		if dep.DependencyType == DependencyHard {
			complexity += 2
		}
	}

	return complexity
}

// Private methods for asset analysis

func (ua *UE5AssetAnalyzer) analyzeUAsset(assetInfo *AssetInfo, content []byte) (*AssetInfo, error) {
	if len(content) < 100 {
		return assetInfo, fmt.Errorf("content too small to be a valid .uasset file")
	}

	// Parse UAsset header
	header, err := ua.parseUAssetHeader(content)
	if err != nil {
		return assetInfo, fmt.Errorf("failed to parse UAsset header: %w", err)
	}

	// Extract names table for string references
	names, err := ua.extractNamesTable(content, header)
	if err != nil {
		return assetInfo, fmt.Errorf("failed to extract names table: %w", err)
	}

	// Extract imports (external dependencies)
	imports, err := ua.extractImports(content, header, names)
	if err != nil {
		return assetInfo, fmt.Errorf("failed to extract imports: %w", err)
	}

	// Convert imports to dependencies
	for _, imp := range imports {
		if ua.isAssetReference(imp) {
			dependency := AssetDependency{
				SourceAsset:    assetInfo.FilePath,
				TargetAsset:    ua.normalizeAssetPath(imp),
				DependencyType: DependencyHard,
				Weight:         1.0,
			}
			assetInfo.Dependencies = append(assetInfo.Dependencies, dependency)
		}
	}

	// Detect if this is a Blueprint
	assetInfo.IsBlueprint = ua.isBlueprintAsset(names)
	if assetInfo.IsBlueprint {
		assetInfo.AssetType = AssetTypeBlueprint
		assetInfo.BlueprintType = ua.detectBlueprintType(names)
	}

	// Extract soft references from the content
	softRefs := ua.extractSoftReferences(content)
	for _, ref := range softRefs {
		dependency := AssetDependency{
			SourceAsset:    assetInfo.FilePath,
			TargetAsset:    ua.normalizeAssetPath(ref),
			DependencyType: DependencySoft,
			Weight:         0.5,
		}
		assetInfo.Dependencies = append(assetInfo.Dependencies, dependency)
	}

	// Calculate complexity
	assetInfo.Complexity = ua.CalculateComplexity(assetInfo)

	return assetInfo, nil
}

func (ua *UE5AssetAnalyzer) analyzeUMap(assetInfo *AssetInfo, content []byte) (*AssetInfo, error) {
	assetInfo.AssetType = AssetTypeLevel

	// Extract level-specific information
	// UMaps contain references to all actors and their assets

	// Find asset references in the level data
	assetRefs := ua.extractAssetReferencesFromLevel(content)

	for _, ref := range assetRefs {
		dependency := AssetDependency{
			SourceAsset:    assetInfo.FilePath,
			TargetAsset:    ua.normalizeAssetPath(ref),
			DependencyType: DependencyHard,
			Weight:         1.0,
		}
		assetInfo.Dependencies = append(assetInfo.Dependencies, dependency)
	}

	// Levels are complex by nature
	assetInfo.Complexity = ua.CalculateComplexity(assetInfo) + 20

	return assetInfo, nil
}

func (ua *UE5AssetAnalyzer) analyzeGenericAsset(assetInfo *AssetInfo, content []byte) (*AssetInfo, error) {
	// For non-UE5 specific files, look for asset path references in the content
	assetRefs := ua.findAssetPathReferences(content)

	for _, ref := range assetRefs {
		dependency := AssetDependency{
			SourceAsset:    assetInfo.FilePath,
			TargetAsset:    ua.normalizeAssetPath(ref),
			DependencyType: DependencySoft,
			Weight:         0.3,
		}
		assetInfo.Dependencies = append(assetInfo.Dependencies, dependency)
	}

	assetInfo.Complexity = ua.CalculateComplexity(assetInfo)
	return assetInfo, nil
}

func (ua *UE5AssetAnalyzer) parseUAssetHeader(content []byte) (*UAssetHeader, error) {
	if len(content) < 32 {
		return nil, fmt.Errorf("content too small for UAsset header")
	}

	header := &UAssetHeader{}
	reader := bytes.NewReader(content)

	if err := binary.Read(reader, binary.LittleEndian, &header.Magic); err != nil {
		return nil, err
	}

	// Verify UE5 magic number
	if header.Magic != 0x9E2A83C1 {
		return nil, fmt.Errorf("invalid UAsset magic number: 0x%x", header.Magic)
	}

	// Read remaining header fields
	if err := binary.Read(reader, binary.LittleEndian, &header.FileVersion); err != nil {
		return nil, err
	}
	if err := binary.Read(reader, binary.LittleEndian, &header.LicenseeVersion); err != nil {
		return nil, err
	}
	if err := binary.Read(reader, binary.LittleEndian, &header.PackageFlags); err != nil {
		return nil, err
	}
	if err := binary.Read(reader, binary.LittleEndian, &header.NameCount); err != nil {
		return nil, err
	}
	if err := binary.Read(reader, binary.LittleEndian, &header.NameOffset); err != nil {
		return nil, err
	}
	if err := binary.Read(reader, binary.LittleEndian, &header.ExportCount); err != nil {
		return nil, err
	}
	if err := binary.Read(reader, binary.LittleEndian, &header.ExportOffset); err != nil {
		return nil, err
	}
	if err := binary.Read(reader, binary.LittleEndian, &header.ImportCount); err != nil {
		return nil, err
	}
	if err := binary.Read(reader, binary.LittleEndian, &header.ImportOffset); err != nil {
		return nil, err
	}

	return header, nil
}

func (ua *UE5AssetAnalyzer) extractNamesTable(content []byte, header *UAssetHeader) ([]string, error) {
	if int(header.NameOffset) >= len(content) {
		return nil, fmt.Errorf("invalid name offset")
	}

	names := make([]string, header.NameCount)
	offset := int(header.NameOffset)

	for i := 0; i < int(header.NameCount); i++ {
		if offset+8 >= len(content) {
			break
		}

		// Read string length
		reader := bytes.NewReader(content[offset:])
		var nameLen int32
		if err := binary.Read(reader, binary.LittleEndian, &nameLen); err != nil {
			break
		}
		offset += 4

		if nameLen <= 0 || int(nameLen) > 1000 { // Reasonable limit
			offset += 4 // Skip hash
			continue
		}

		// Read string data
		if offset+int(nameLen) > len(content) {
			break
		}

		nameBytes := content[offset : offset+int(nameLen)-1] // -1 to exclude null terminator
		names[i] = string(nameBytes)
		offset += int(nameLen)

		// Skip hash (4 bytes)
		offset += 4
	}

	return names, nil
}

func (ua *UE5AssetAnalyzer) extractImports(content []byte, header *UAssetHeader, names []string) ([]string, error) {
	if int(header.ImportOffset) >= len(content) {
		return nil, fmt.Errorf("invalid import offset")
	}

	var imports []string
	offset := int(header.ImportOffset)

	for i := 0; i < int(header.ImportCount); i++ {
		if offset+20 >= len(content) { // Minimum import entry size
			break
		}

		reader := bytes.NewReader(content[offset:])

		// Read import entry structure (simplified)
		var classPackage, className, packageName, objectName int32

		if err := binary.Read(reader, binary.LittleEndian, &classPackage); err != nil {
			break
		}
		if err := binary.Read(reader, binary.LittleEndian, &className); err != nil {
			break
		}
		if err := binary.Read(reader, binary.LittleEndian, &packageName); err != nil {
			break
		}
		if err := binary.Read(reader, binary.LittleEndian, &objectName); err != nil {
			break
		}

		// Build import path from name indices
		if packageName >= 0 && int(packageName) < len(names) && objectName >= 0 && int(objectName) < len(names) {
			packageStr := names[packageName]
			objectStr := names[objectName]

			if packageStr != "" && objectStr != "" {
				importPath := fmt.Sprintf("%s.%s", packageStr, objectStr)
				imports = append(imports, importPath)
			}
		}

		offset += 20 // Move to next import entry
	}

	return imports, nil
}

func (ua *UE5AssetAnalyzer) extractSoftReferences(content []byte) []string {
	// Look for soft object path patterns in the content
	// Soft references often appear as "/Game/Path/To/Asset.Asset"

	contentStr := string(content)

	// Regex pattern for UE5 asset paths
	assetPathPattern := regexp.MustCompile(`/Game/[A-Za-z0-9/_-]+\.[A-Za-z0-9_-]+`)
	matches := assetPathPattern.FindAllString(contentStr, -1)

	// Deduplicate
	seen := make(map[string]bool)
	var uniqueRefs []string
	for _, match := range matches {
		if !seen[match] {
			seen[match] = true
			uniqueRefs = append(uniqueRefs, match)
		}
	}

	return uniqueRefs
}

func (ua *UE5AssetAnalyzer) extractAssetReferencesFromLevel(content []byte) []string {
	// Level files contain references to all placed actors and their assets
	// This is a simplified implementation - full parsing would be much more complex

	return ua.findAssetPathReferences(content)
}

func (ua *UE5AssetAnalyzer) findAssetPathReferences(content []byte) []string {
	contentStr := string(content)

	// Common UE5 asset path patterns
	patterns := []string{
		`/Game/[A-Za-z0-9/_-]+\.uasset`,
		`/Game/[A-Za-z0-9/_-]+\.[A-Za-z0-9_-]+`,
		`Blueprint'[^']*'`,
		`StaticMesh'[^']*'`,
		`Texture2D'[^']*'`,
		`Material'[^']*'`,
	}

	var refs []string
	seen := make(map[string]bool)

	for _, pattern := range patterns {
		regex := regexp.MustCompile(pattern)
		matches := regex.FindAllString(contentStr, -1)

		for _, match := range matches {
			cleaned := strings.Trim(match, "'\"")
			if !seen[cleaned] && ua.isValidAssetPath(cleaned) {
				seen[cleaned] = true
				refs = append(refs, cleaned)
			}
		}
	}

	return refs
}

// Helper methods

func (ua *UE5AssetAnalyzer) determineAssetType(filePath string, content []byte) AssetType {
	ext := strings.ToLower(filepath.Ext(filePath))

	switch ext {
	case ".umap":
		return AssetTypeLevel
	case ".uasset":
		// Need to analyze content to determine specific type
		if ua.isBlueprintAssetFromContent(content) {
			return AssetTypeBlueprint
		}
		// Default to unknown, will be refined during analysis
		return AssetTypeUnknown
	default:
		return AssetTypeUnknown
	}
}

func (ua *UE5AssetAnalyzer) extractAssetName(filePath string) string {
	base := filepath.Base(filePath)
	return strings.TrimSuffix(base, filepath.Ext(base))
}

func (ua *UE5AssetAnalyzer) extractPackageName(filePath string) string {
	// Convert file path to UE5 package name
	// /Game/Characters/Hero.uasset -> /Game/Characters/Hero

	// Remove file extension
	packagePath := strings.TrimSuffix(filePath, filepath.Ext(filePath))

	// Normalize path separators
	packagePath = strings.ReplaceAll(packagePath, "\\", "/")

	// Ensure it starts with /Game if it's a content asset
	if !strings.HasPrefix(packagePath, "/") {
		packagePath = "/Game/" + packagePath
	}

	return packagePath
}

func (ua *UE5AssetAnalyzer) normalizeAssetPath(assetPath string) string {
	// Normalize various asset path formats to a consistent format
	assetPath = strings.TrimSpace(assetPath)
	assetPath = strings.ReplaceAll(assetPath, "\\", "/")

	// Remove quotes and prefixes
	assetPath = strings.Trim(assetPath, "'\"")

	// Handle Blueprint'...' format
	if strings.HasPrefix(assetPath, "Blueprint'") && strings.HasSuffix(assetPath, "'") {
		assetPath = assetPath[10 : len(assetPath)-1] // Remove Blueprint' and '
	}

	// Handle other class prefixes
	for _, prefix := range []string{"StaticMesh'", "Texture2D'", "Material'"} {
		if strings.HasPrefix(assetPath, prefix) && strings.HasSuffix(assetPath, "'") {
			assetPath = assetPath[len(prefix) : len(assetPath)-1]
		}
	}

	return assetPath
}

func (ua *UE5AssetAnalyzer) isAssetReference(reference string) bool {
	// Check if the reference looks like an asset path
	if strings.Contains(reference, "/Game/") {
		return true
	}

	// Check for common asset extensions in the reference
	assetExtensions := []string{".uasset", ".umap", "_C"}
	for _, ext := range assetExtensions {
		if strings.Contains(reference, ext) {
			return true
		}
	}

	return false
}

func (ua *UE5AssetAnalyzer) isValidAssetPath(path string) bool {
	// Basic validation for asset paths
	if len(path) < 5 {
		return false
	}

	// Must contain /Game/ or be a relative path
	if !strings.Contains(path, "/Game/") && !strings.Contains(path, "/") {
		return false
	}

	// Should not contain invalid characters
	invalidChars := []string{"<", ">", "|", "*", "?"}
	for _, char := range invalidChars {
		if strings.Contains(path, char) {
			return false
		}
	}

	return true
}

func (ua *UE5AssetAnalyzer) isBlueprintAsset(names []string) bool {
	// Check if the names table contains Blueprint-related entries
	blueprintKeywords := []string{"Blueprint", "_C", "GeneratedClass", "UbergraphPages"}

	for _, name := range names {
		for _, keyword := range blueprintKeywords {
			if strings.Contains(name, keyword) {
				return true
			}
		}
	}

	return false
}

func (ua *UE5AssetAnalyzer) isBlueprintAssetFromContent(content []byte) bool {
	contentStr := string(content)
	blueprintMarkers := []string{"Blueprint", "UbergraphPages", "GeneratedClass"}

	for _, marker := range blueprintMarkers {
		if strings.Contains(contentStr, marker) {
			return true
		}
	}

	return false
}

func (ua *UE5AssetAnalyzer) detectBlueprintType(names []string) string {
	// Analyze names to determine Blueprint type
	for _, name := range names {
		if strings.Contains(name, "Actor") {
			return "Actor"
		}
		if strings.Contains(name, "Pawn") {
			return "Pawn"
		}
		if strings.Contains(name, "Character") {
			return "Character"
		}
		if strings.Contains(name, "Widget") {
			return "Widget"
		}
		if strings.Contains(name, "GameMode") {
			return "GameMode"
		}
		if strings.Contains(name, "Component") {
			return "Component"
		}
	}

	return "Unknown"
}

func (ua *UE5AssetAnalyzer) initializeClassMappings() {
	ua.classTypeMap = map[string]AssetType{
		"StaticMesh":               AssetTypeStaticMesh,
		"SkeletalMesh":             AssetTypeSkeletalMesh,
		"Texture2D":                AssetTypeTexture2D,
		"Material":                 AssetTypeMaterial,
		"MaterialInstanceConstant": AssetTypeMaterialInstance,
		"SoundWave":                AssetTypeSound,
		"AnimSequence":             AssetTypeAnimation,
		"Blueprint":                AssetTypeBlueprint,
		"UserWidget":               AssetTypeWidget,
		"DataAsset":                AssetTypeDataAsset,
	}
}
