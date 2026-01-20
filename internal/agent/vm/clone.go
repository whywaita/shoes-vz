package vm

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// cloneFile performs APFS clone of a file
func cloneFile(src, dst string) error {
	// Ensure source exists
	if _, err := os.Stat(src); err != nil {
		return fmt.Errorf("source file does not exist: %w", err)
	}

	// Ensure destination directory exists
	dstDir := filepath.Dir(dst)
	if err := os.MkdirAll(dstDir, 0755); err != nil {
		return fmt.Errorf("failed to create destination directory: %w", err)
	}

	// Use 'cp -c' for APFS clone
	// -c flag uses clonefile(2) system call for Copy-on-Write
	cmd := exec.Command("cp", "-c", src, dst)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to clone file: %w, output: %s", err, string(output))
	}

	return nil
}

// copyFile performs regular file copy (fallback)
func copyFile(src, dst string) error {
	input, err := os.ReadFile(src)
	if err != nil {
		return fmt.Errorf("failed to read source file: %w", err)
	}

	if err := os.WriteFile(dst, input, 0644); err != nil {
		return fmt.Errorf("failed to write destination file: %w", err)
	}

	return nil
}
