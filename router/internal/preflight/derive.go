package preflight

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"

	"loom/router/internal/provider"
)

// DeriveInput is everything the payload is assembled from. Nothing here is a
// diff: the declaration is what a person declared, the facts are what already
// exists, and the criteria are the registry's material.
type DeriveInput struct {
	// Raw is the exact declaration bytes received, hashed for payload_hash.
	Raw json.RawMessage
	// Declaration is the parsed and validated form of Raw.
	Declaration *Declaration
	// Facts are the observable session facts (branch, modified-file count).
	Facts provider.SessionFacts
	// Criteria are the dispatchable methods' criteria, in registry order.
	Criteria []provider.Criterion
}

// Payload is the assembled request material.
type Payload struct {
	// State is the state document sent to the endpoint.
	State json.RawMessage
	// Questions are the typed questions sent alongside it.
	Questions []provider.Question
	// PayloadHash is the SHA-256 of the exact pre-flight JSON bytes. The prose
	// itself is never persisted in tracked files; the hash is what the record
	// keeps, so a later audit can prove which declaration was sent.
	PayloadHash string
}

// Derive assembles the payload from the validated declaration plus the
// observable session facts. The empty-diff question does not arise here: no
// measurement of the change is attempted, because at decision time the change
// usually does not exist yet.
func Derive(in DeriveInput) (*Payload, error) {
	if in.Declaration == nil {
		return nil, fmt.Errorf("derive: missing declaration")
	}
	if len(in.Raw) == 0 {
		return nil, fmt.Errorf("derive: missing pre-flight bytes for the payload hash")
	}
	if err := in.Declaration.ValidateFields(); err != nil {
		return nil, fmt.Errorf("derive: %w", err)
	}
	state, err := provider.BuildState(in.Raw, in.Facts, in.Criteria)
	if err != nil {
		return nil, fmt.Errorf("derive: %w", err)
	}
	sum := sha256.Sum256(in.Raw)
	return &Payload{
		State:       state,
		Questions:   provider.BuildQuestions(in.Criteria),
		PayloadHash: hex.EncodeToString(sum[:]),
	}, nil
}

// ModifiedFileCount reports how many tracked files are currently modified. It is
// a cleanliness summary only: paths are never sent to the endpoint.
func ModifiedFileCount(root string) (int, error) {
	cmd := exec.Command("git", "status", "--porcelain", "--untracked-files=no")
	cmd.Dir = root
	out, err := cmd.Output()
	if err != nil {
		return 0, fmt.Errorf("count modified files: %w", err)
	}
	count := 0
	for _, line := range strings.Split(string(out), "\n") {
		if strings.TrimSpace(line) != "" {
			count++
		}
	}
	return count, nil
}
