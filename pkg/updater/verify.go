// Copyright (c) 2026 Xerolux. All rights reserved.
// ModBridge — Modbus TCP Proxy Manager
// Created by Xerolux
// https://github.com/Xerolux/modbridge

package updater

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"
)

// ErrChecksumMissing is returned when the asset name does not appear in
// the downloaded checksums.txt file.
var ErrChecksumMissing = errors.New("asset not found in checksums file")

// ErrChecksumMismatch is returned when the computed SHA256 of the
// downloaded asset does not match the value recorded in checksums.txt.
var ErrChecksumMismatch = errors.New("checksum mismatch — download may be corrupted or tampered")

// VerifyChecksum parses checksumsTxt (standard sha256sum output format:
// "<hex>  <filename>" per line, two spaces) and confirms that the SHA256
// of assetBytes matches the entry for assetName.
func VerifyChecksum(assetBytes []byte, assetName string, checksumsTxt []byte) error {
	expected, err := findChecksum(checksumsTxt, assetName)
	if err != nil {
		return err
	}

	computed := sha256.Sum256(assetBytes)
	computedHex := hex.EncodeToString(computed[:])

	if !strings.EqualFold(computedHex, expected) {
		return ErrChecksumMismatch
	}
	return nil
}

// findChecksum scans the sha256sum-formatted text for the given filename
// and returns the hex digest. Lines that don't match the expected format
// are silently skipped (defensive parsing).
func findChecksum(checksumsTxt []byte, assetName string) (string, error) {
	lines := strings.Split(string(checksumsTxt), "\n")
	for _, line := range lines {
		// sha256sum format: "<64-hex>  <filename>" (two spaces)
		parts := strings.SplitN(strings.TrimSpace(line), "  ", 2)
		if len(parts) != 2 {
			continue
		}
		digest := strings.TrimSpace(parts[0])
		name := strings.TrimSpace(parts[1])
		if name == assetName && len(digest) == 64 {
			return digest, nil
		}
	}
	return "", ErrChecksumMissing
}
