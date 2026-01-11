package git

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"

	"github.com/jakoblorz/go-changesets/internal/models"
)

// MockGitClient implements GitClient for testing with full commit graph simulation
type MockGitClient struct {
	mu       sync.RWMutex
	tags     map[string]*MockTag    // key: tag name
	commits  map[string]*MockCommit // key: commit hash
	branches map[string]*MockBranch // key: branch name
	head     string                 // current HEAD commit hash
	branch   string                 // current branch name
	isRepo   bool
	ctx      context.Context

	// File tracking for git history simulation
	fileCreationCommits map[string]string // filePath -> commit SHA

	// Hooks for testing error scenarios
	GetLatestTagError     error
	CreateTagError        error
	PushTagError          error
	TagExistsError        error
	GetTagAnnotationError error
}

// MockTag represents a git tag
type MockTag struct {
	Name       string
	Message    string
	IsPushed   bool
	ProjectTag string // e.g., "auth@v1.2.3"
	CommitHash string // NEW: which commit this tag points to
}

// MockCommit represents a git commit
type MockCommit struct {
	Hash    string
	Parents []string // parent commit hashes
	Message string
}

// MockBranch represents a git branch
type MockBranch struct {
	Name string
	Head string // commit hash this branch points to
}

// NewMockGitClient creates a new MockGitClient with initial commit and main branch
func NewMockGitClient() *MockGitClient {
	mock := &MockGitClient{
		tags:                make(map[string]*MockTag),
		commits:             make(map[string]*MockCommit),
		branches:            make(map[string]*MockBranch),
		isRepo:              true,
		branch:              "main",
		ctx:                 context.Background(),
		fileCreationCommits: make(map[string]string),
	}

	// Create initial commit (like real git init)
	initialHash := mock.createInitialCommit()

	// Create main branch pointing to initial commit
	mock.branches["main"] = &MockBranch{
		Name: "main",
		Head: initialHash,
	}
	mock.head = initialHash

	return mock
}

// createInitialCommit creates the initial commit (called during initialization)
func (m *MockGitClient) createInitialCommit() string {
	hash := generateCommitHash()
	m.commits[hash] = &MockCommit{
		Hash:    hash,
		Parents: []string{}, // No parents
		Message: "Initial commit",
	}
	return hash
}

// generateCommitHash generates a unique commit hash (7-char hex like git)
func generateCommitHash() string {
	hashCounterMu.Lock()
	defer hashCounterMu.Unlock()
	hashCounterValue++
	return fmt.Sprintf("%07x", hashCounterValue&0xFFFFFFF)
}

// Simple counter for unique commit hashes
var (
	hashCounterMu    sync.Mutex
	hashCounterValue uint64
)

// WithContext returns a new client with the given context
func (m *MockGitClient) WithContext(ctx context.Context) GitClient {
	m.mu.Lock()
	defer m.mu.Unlock()

	return &MockGitClient{
		tags:                m.tags,
		commits:             m.commits,
		branches:            m.branches,
		head:                m.head,
		branch:              m.branch,
		isRepo:              m.isRepo,
		ctx:                 ctx,
		fileCreationCommits: m.fileCreationCommits,

		GetLatestTagError:     m.GetLatestTagError,
		CreateTagError:        m.CreateTagError,
		PushTagError:          m.PushTagError,
		TagExistsError:        m.TagExistsError,
		GetTagAnnotationError: m.GetTagAnnotationError,
	}
}

// AddTag adds a tag to the mock (for test setup)
// For backward compatibility, creates tag at current HEAD
func (m *MockGitClient) AddTag(projectName, version, message string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	tagName := fmt.Sprintf("%s@v%s", projectName, version)
	m.tags[tagName] = &MockTag{
		Name:       tagName,
		Message:    message,
		IsPushed:   false,
		ProjectTag: tagName,
		CommitHash: m.head, // Tag points to current HEAD
	}
}

// AddPushedTag adds a tag that's already pushed (for test setup)
// For backward compatibility, creates tag at current HEAD
func (m *MockGitClient) AddPushedTag(projectName, version, message string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	tagName := fmt.Sprintf("%s@v%s", projectName, version)
	m.tags[tagName] = &MockTag{
		Name:       tagName,
		Message:    message,
		IsPushed:   true,
		ProjectTag: tagName,
		CommitHash: m.head, // Tag points to current HEAD
	}
}

// SetIsRepo sets whether this is a git repository
func (m *MockGitClient) SetIsRepo(isRepo bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.isRepo = isRepo
}

// SetBranch sets the current branch name
func (m *MockGitClient) SetBranch(branch string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.branch = branch
}

// Graph operations for simulating git commit history

// CreateCommit creates a new commit with the given parent(s)
// If no parents specified, uses current HEAD as parent
// Returns the commit hash
func (m *MockGitClient) CreateCommit(message string, parents ...string) string {
	m.mu.Lock()
	defer m.mu.Unlock()

	// If no parents specified, use current HEAD as parent (if it exists)
	if len(parents) == 0 && m.head != "" {
		parents = []string{m.head}
	}

	hash := generateCommitHash()
	m.commits[hash] = &MockCommit{
		Hash:    hash,
		Parents: parents,
		Message: message,
	}

	// Move HEAD to this commit
	m.head = hash

	// Update current branch to point to this commit
	if branch, exists := m.branches[m.branch]; exists {
		branch.Head = hash
	}

	return hash
}

// CreateBranch creates a new branch pointing to current HEAD
func (m *MockGitClient) CreateBranch(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.branches[name]; exists {
		return fmt.Errorf("branch %s already exists", name)
	}

	m.branches[name] = &MockBranch{
		Name: name,
		Head: m.head,
	}

	return nil
}

// CheckoutBranch switches to an existing branch
func (m *MockGitClient) CheckoutBranch(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	branch, exists := m.branches[name]
	if !exists {
		return fmt.Errorf("branch %s not found", name)
	}

	m.branch = name
	m.head = branch.Head

	return nil
}

// MergeBranch merges a branch into the current branch (creates merge commit)
func (m *MockGitClient) MergeBranch(branchName string) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	sourceBranch, exists := m.branches[branchName]
	if !exists {
		return "", fmt.Errorf("branch %s not found", branchName)
	}

	// Create merge commit with two parents: current HEAD and source branch HEAD
	currentHead := m.head
	sourceHead := sourceBranch.Head

	hash := generateCommitHash()
	m.commits[hash] = &MockCommit{
		Hash:    hash,
		Parents: []string{currentHead, sourceHead},
		Message: fmt.Sprintf("Merge branch '%s'", branchName),
	}

	// Move HEAD to merge commit
	m.head = hash

	// Update current branch
	if branch, exists := m.branches[m.branch]; exists {
		branch.Head = hash
	}

	return hash, nil
}

// isAncestor checks if ancestorHash is an ancestor of descendantHash using BFS
func (m *MockGitClient) isAncestor(ancestorHash, descendantHash string) bool {
	if ancestorHash == descendantHash {
		return true
	}

	// BFS through parent chain
	visited := make(map[string]bool)
	queue := []string{descendantHash}

	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]

		if visited[current] {
			continue
		}
		visited[current] = true

		if current == ancestorHash {
			return true
		}

		commit, exists := m.commits[current]
		if exists {
			queue = append(queue, commit.Parents...)
		}
	}

	return false
}

// isReachableFromHEAD checks if a commit is reachable from current HEAD
func (m *MockGitClient) isReachableFromHEAD(commitHash string) bool {
	if commitHash == "" {
		return false
	}
	return m.isAncestor(commitHash, m.head)
}

func (m *MockGitClient) GetLatestTag(projectName string) (string, error) {
	if m.GetLatestTagError != nil {
		return "", m.GetLatestTagError
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	// Find all tags for this project
	pattern := fmt.Sprintf("%s@v", projectName)
	var matchingTags []string

	for tagName, tag := range m.tags {
		if !strings.HasPrefix(tagName, pattern) {
			continue
		}
		if !m.isReachableFromHEAD(tag.CommitHash) {
			continue
		}
		matchingTags = append(matchingTags, tagName)
	}

	if len(matchingTags) == 0 {
		return "", fmt.Errorf("no tags found for project %s", projectName)
	}

	// Sort by version (reverse order for latest first) using semver comparison
	sort.Slice(matchingTags, func(i, j int) bool {
		return compareTagVersions(matchingTags[i], matchingTags[j])
	})

	return matchingTags[0], nil
}

func (m *MockGitClient) CreateTag(tagName, message string) error {
	if m.CreateTagError != nil {
		return m.CreateTagError
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	// Check if tag already exists
	if _, exists := m.tags[tagName]; exists {
		return fmt.Errorf("tag %s already exists", tagName)
	}

	m.tags[tagName] = &MockTag{
		Name:       tagName,
		Message:    message,
		IsPushed:   false,
		ProjectTag: tagName,
		CommitHash: m.head, // Tag points to current HEAD
	}

	return nil
}

func (m *MockGitClient) PushTag(tagName string) error {
	if m.PushTagError != nil {
		return m.PushTagError
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	tag, exists := m.tags[tagName]
	if !exists {
		return fmt.Errorf("tag %s does not exist", tagName)
	}

	tag.IsPushed = true
	return nil
}

func (m *MockGitClient) TagExists(tagName string) (bool, error) {
	if m.TagExistsError != nil {
		return false, m.TagExistsError
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	_, exists := m.tags[tagName]
	return exists, nil
}

func (m *MockGitClient) GetTagAnnotation(tagName string) (string, error) {
	if m.GetTagAnnotationError != nil {
		return "", m.GetTagAnnotationError
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	tag, exists := m.tags[tagName]
	if !exists {
		return "", fmt.Errorf("tag %s not found", tagName)
	}

	return tag.Message, nil
}

func (m *MockGitClient) IsGitRepo() (bool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.isRepo, nil
}

func (m *MockGitClient) GetCurrentBranch() (string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.branch, nil
}

// GetTagsWithPrefix returns all tags matching the prefix pattern
// that are reachable from current HEAD (uses git ancestry simulation)
// Supports wildcard patterns like "backend@v*"
func (m *MockGitClient) GetTagsWithPrefix(prefix string) ([]string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var matchingTags []string
	for tagName, tag := range m.tags {
		// Check pattern match
		if !matchesPattern(tagName, prefix) {
			continue
		}

		// Check if tag's commit is reachable from HEAD
		if !m.isReachableFromHEAD(tag.CommitHash) {
			continue
		}

		matchingTags = append(matchingTags, tagName)
	}

	// Sort by version (reverse order for latest first) using semver comparison
	sort.Slice(matchingTags, func(i, j int) bool {
		return compareTagVersions(matchingTags[i], matchingTags[j])
	})

	return matchingTags, nil
}

// matchesPattern checks if a tag name matches a pattern (supports * wildcard)
func matchesPattern(name, pattern string) bool {
	// Simple wildcard matching: convert pattern to prefix if it ends with *
	if strings.HasSuffix(pattern, "*") {
		prefix := strings.TrimSuffix(pattern, "*")
		return strings.HasPrefix(name, prefix)
	}
	// Exact match if no wildcard
	return name == pattern
}

// compareTagVersions compares two tag names by their semantic version
// Returns true if tag i should come before tag j (descending order)
// Mimics git's --sort=-version:refname behavior
func compareTagVersions(tagI, tagJ string) bool {
	// Extract version from tags: "backend@v1.2.3" -> "1.2.3"
	versionI := extractVersionFromTag(tagI)
	versionJ := extractVersionFromTag(tagJ)

	// If we can't parse, fall back to string comparison
	if versionI == "" || versionJ == "" {
		return tagI > tagJ
	}

	// Parse as semantic versions
	verI, errI := models.ParseVersion(versionI)
	verJ, errJ := models.ParseVersion(versionJ)

	if errI != nil || errJ != nil {
		// Fall back to string comparison
		return tagI > tagJ
	}

	// Use Version.Compare: returns 1 if verI > verJ
	return verI.Compare(verJ) > 0
}

// extractVersionFromTag extracts version string from tag name
// "backend@v1.2.3-rc0" -> "1.2.3-rc0"
func extractVersionFromTag(tag string) string {
	parts := strings.Split(tag, "@")
	if len(parts) != 2 {
		return ""
	}
	return parts[1]
}

// ExtractRCNumber extracts the RC number from a tag
// Delegates to the real implementation in os.go
func (m *MockGitClient) ExtractRCNumber(tag string) (int, error) {
	// Use the same logic as OSGitClient
	rcIdx := strings.Index(tag, "-rc")
	if rcIdx == -1 {
		return -1, nil // Not an RC tag
	}

	rcSuffix := tag[rcIdx+3:]
	if rcSuffix == "" {
		return -1, fmt.Errorf("invalid RC tag format: %s (expected -rc{number})", tag)
	}

	// Need to import strconv
	var num int
	_, err := fmt.Sscanf(rcSuffix, "%d", &num)
	if err != nil {
		return -1, fmt.Errorf("invalid RC number in tag %s: %w", tag, err)
	}

	return num, nil
}

// GetAllTags returns all tags (helper for testing)
func (m *MockGitClient) GetAllTags() map[string]*MockTag {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make(map[string]*MockTag)
	for k, v := range m.tags {
		result[k] = v
	}
	return result
}

// Reset clears all data from the mock and reinitializes with fresh commit graph (helper for testing)
func (m *MockGitClient) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Clear all data
	m.tags = make(map[string]*MockTag)
	m.commits = make(map[string]*MockCommit)
	m.branches = make(map[string]*MockBranch)
	m.fileCreationCommits = make(map[string]string)
	m.isRepo = true
	m.branch = "main"
	m.ctx = context.Background()

	// Reinitialize with initial commit and main branch
	initialHash := m.createInitialCommitUnsafe()
	m.branches["main"] = &MockBranch{
		Name: "main",
		Head: initialHash,
	}
	m.head = initialHash

	// Clear error hooks
	m.GetLatestTagError = nil
	m.CreateTagError = nil
	m.PushTagError = nil
	m.TagExistsError = nil
	m.GetTagAnnotationError = nil
}

// createInitialCommitUnsafe creates initial commit without locking (used by Reset)
func (m *MockGitClient) createInitialCommitUnsafe() string {
	hash := generateCommitHash()
	m.commits[hash] = &MockCommit{
		Hash:    hash,
		Parents: []string{},
		Message: "Initial commit",
	}
	return hash
}

// GetFileCreationCommit returns the commit SHA that added a file
// Returns empty string if file not tracked
func (m *MockGitClient) GetFileCreationCommit(filePath string) (string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if commit, ok := m.fileCreationCommits[filePath]; ok {
		return commit, nil
	}
	return "", nil
}

// GetCommitMessage returns the commit message for a given SHA
func (m *MockGitClient) GetCommitMessage(commitSHA string) (string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if commitSHA == "" {
		return "", fmt.Errorf("commit SHA cannot be empty")
	}

	if commit, ok := m.commits[commitSHA]; ok {
		return commit.Message, nil
	}

	return "", fmt.Errorf("commit not found: %s", commitSHA)
}

// SetFileCreationCommit tracks when a file was created (for testing)
func (m *MockGitClient) SetFileCreationCommit(filePath, commitSHA string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.fileCreationCommits[filePath] = commitSHA
}
