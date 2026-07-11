// Copyright (c) 2026 Xerolux. All rights reserved.
// ModBridge — Modbus TCP Proxy Manager
// Created by Xerolux
// https://github.com/Xerolux/modbridge

package updater

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// targetBinaryPath derives the install path of the binary being replaced from
// the path of the staged new binary. The updater stages the freshly downloaded
// build next to the live binary with a ".new" suffix (e.g. "/opt/modbridge/modbridge.new"),
// so the live binary is the same path with that suffix removed. Deriving the
// target from newBinaryPath (rather than os.Executable) keeps SwapBinary
// independent of whichever process invoked it and makes it testable against an
// isolated temp directory.
func targetBinaryPath(newBinaryPath string) (string, error) {
	dir := filepath.Dir(newBinaryPath)
	base := filepath.Base(newBinaryPath)
	if trimmed := strings.TrimSuffix(base, ".new"); trimmed != base && trimmed != "" {
		return filepath.Join(dir, trimmed), nil
	}
	return "", fmt.Errorf("new binary path %q must end with .new", newBinaryPath)
}

// SwapBinary atomically replaces the live binary (located by stripping the
// trailing ".new" from newBinaryPath) with the file at newBinaryPath. The
// previous binary is preserved at "<binaryPath>.bak.<timestamp>". Returns the
// backup path.
//
// The new binary MUST reside in the same directory as the target so that the
// underlying rename is atomic (a rename is only atomic within a single
// filesystem / mount point). If staging the new binary into place fails, the
// original binary is restored from the backup.
func SwapBinary(newBinaryPath string) (string, error) {
	execPath, err := targetBinaryPath(newBinaryPath)
	if err != nil {
		return "", err
	}

	backupPath := execPath + ".bak." + time.Now().Format("20060102_150405")

	// Step 1: move current binary aside (backup).
	if err := os.Rename(execPath, backupPath); err != nil {
		return "", fmt.Errorf("creating backup %s: %w", backupPath, err)
	}

	// Step 2: move new binary into place.
	if err := os.Rename(newBinaryPath, execPath); err != nil {
		// Rollback: restore the old binary.
		if rbErr := os.Rename(backupPath, execPath); rbErr != nil {
			return "", fmt.Errorf("swap failed (%v) and rollback also failed (%v) — binary at %s, manual recovery needed", err, rbErr, backupPath)
		}
		return "", fmt.Errorf("installing new binary: %w (rolled back)", err)
	}

	// Step 3: ensure executable bit.
	if err := os.Chmod(execPath, 0755); err != nil {
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

	if _, err := copyAll(dst, src); err != nil {
		return fmt.Errorf("copying backup to target: %w", err)
	}
	return nil
}

// rollbackTargetPath turns a backup path produced by SwapBinary back into the
// live binary path by stripping the ".bak.<timestamp>" segment.
func rollbackTargetPath(backupPath string) (string, error) {
	dir := filepath.Dir(backupPath)
	base := filepath.Base(backupPath)
	// Strip a single ".bak" segment plus any trailing ".<timestamp>".
	if i := strings.Index(base, ".bak"); i > 0 {
		return filepath.Join(dir, base[:i]), nil
	}
	return "", fmt.Errorf("backup path %q does not contain a .bak segment", backupPath)
}

// copyAll is a small io.Copy-style helper used by the rollback path.
func copyAll(dst *os.File, src *os.File) (int64, error) {
	buf := make([]byte, 32*1024)
	var total int64
	for {
		n, err := src.Read(buf)
		if n > 0 {
			written, werr := dst.Write(buf[:n])
			total += int64(written)
			if werr != nil {
				return total, werr
			}
		}
		if err != nil {
			if err.Error() == "EOF" {
				return total, nil
			}
			return total, err
		}
	}
}
