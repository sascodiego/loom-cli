package preflight

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

// Validate checks the declaration fields and then every declared subsystem
// against the repository's tracked top-level directories. A mismatch is a
// declaration error (exit 2): the agent declared work in a place that does not
// exist, so the classification would be built on a false premise.
func (d *Declaration) Validate(root string) error {
	if err := d.ValidateFields(); err != nil {
		return err
	}
	if len(d.Subsystems) == 0 {
		// Legitimate for a root-file-only change; Clef simply receives less
		// breadth signal. Nothing to validate.
		return nil
	}
	dirs, err := TopLevelDirs(root)
	if err != nil {
		return err
	}
	known := make(map[string]bool, len(dirs))
	for _, dir := range dirs {
		known[dir] = true
	}
	var unknown []string
	for _, declared := range d.Subsystems {
		if !known[declared] {
			unknown = append(unknown, declared)
		}
	}
	if len(unknown) > 0 {
		return fmt.Errorf("unknown subsystem(s) %s: declared subsystems must be tracked top-level directories of this repository (%s)",
			strings.Join(unknown, ", "), strings.Join(dirs, ", "))
	}
	return nil
}

// TopLevelDirs lists the tracked top-level directories of the repository at
// root, derived from `git ls-files`. Root-level files (which have no directory
// component) are not subsystems and are excluded.
func TopLevelDirs(root string) ([]string, error) {
	cmd := exec.Command("git", "ls-files")
	cmd.Dir = root
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("list tracked files: %w", err)
	}
	seen := map[string]bool{}
	dirs := make([]string, 0, 8)
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		slash := strings.Index(line, "/")
		if slash <= 0 {
			continue
		}
		dir := line[:slash]
		if !seen[dir] {
			seen[dir] = true
			dirs = append(dirs, dir)
		}
	}
	sort.Strings(dirs)
	if len(dirs) == 0 {
		return nil, fmt.Errorf("repository has no tracked top-level directories to validate against")
	}
	return dirs, nil
}

// normalisePath keeps repository-relative comparisons stable across platforms.
func normalisePath(p string) string { return filepath.ToSlash(p) }
