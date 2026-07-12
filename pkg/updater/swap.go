// Copyright (c) 2026 Xerolux. All rights reserved.
// ModBridge — Modbus TCP Proxy Manager
// Created by Xerolux
// https://github.com/Xerolux/modbridge

package updater

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// SwapBinary atomically replaces the binary at targetPath with the file at
// newBinaryPath. The previous binary is preserved at
// "<targetPath>.bak.<timestamp>". Returns the backup path.
//
// newBinaryPath MUST reside in the same directory as targetPath so the
// underlying rename is atomic (rename is only atomic within a single
// filesystem). If staging fails, the original binary is restored.
func SwapBinary(newBinaryPath, targetPath string) (string, error) {
	backupPath := targetPath + ".bak." + time.Now().Format("20060102_150405")

	// Step 1: move current binary aside (backup).
	if err := os.Rename(targetPath, backupPath); err != nil {
		return "", fmt.Errorf("creating backup %s: %w", backupPath, err)
	}

	// Step 2: move new binary into place.
	if err := os.Rename(newBinaryPath, targetPath); err != nil {
		// Rollback: restore the old binary.
		if rbErr := os.Rename(backupPath, targetPath); rbErr != nil {
			return "", fmt.Errorf("swap failed (%v) and rollback also failed (%v) — binary at %s, manual recovery needed", err, rbErr, backupPath)
		}
		return "", fmt.Errorf("installing new binary: %w (rolled back)", err)
	}

	// Step 3: ensure executable bit.
	if err := os.Chmod(targetPath, 0755); err != nil {
		return backupPath, fmt.Errorf("setting executable permissions: %w", err)
	}

	return backupPath, nil
}

// RollbackBinary copies the backup file back over the current binary path
// (derived from backupPath by stripping the ".bak.<timestamp>" suffix, or —
// failing that — its ".bak" segment). The backup file itself is preserved for
// forensics. Used when the new binary is detected as broken after a restart
// (manual or future auto-detect).
func RollbackBinary(backupPath string) error {
	execPath, err := rollbackTargetPath(backupPath)
	if err != nil {
		return err
	}

	// Copy backup -> exec (don't rename; keep backup for forensics).
	src, err := os.Open(backupPath)
	if err != nil {
		return fmt.Errorf("opening backup: %w", err)
	}
	defer src.Close()

	dst, err := os.OpenFile(execPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0755)
	if err != nil {
		return fmt.Errorf("opening target for rollback: %w", err)
	}
	defer dst.Close()

	if _, err := io.Copy(dst, src); err != nil {
		return fmt.Errorf("copying backup to target: %w", err)
	}
	return nil
}

// rollbackTargetPath turns a backup path produced by SwapBinary back into the
// live binary path by stripping the ".bak.<timestamp>" segment.
func rollbackTargetPath(backupPath string) (string, error) {
	dir := filepath.Dir(backupPath)
	base := filepath.Base(backupPath)
	// Strip at the LAST ".bak" segment so binary names that legitimately
	// contain ".bak" earlier (e.g. "modbridge.bak-tool.bak.<ts>") are handled.
	// LastIndex matches the suffix appended by SwapBinary; Index would wrongly
	// cut at the first occurrence.
	if i := strings.LastIndex(base, ".bak"); i > 0 {
		return filepath.Join(dir, base[:i]), nil
	}
	return "", fmt.Errorf("backup path %q does not contain a .bak segment", backupPath)
}
