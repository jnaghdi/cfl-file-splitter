// SPDX-License-Identifier: MIT
// Copyright (c) 2026 Computer Forensics Lab Ltd
package core

import (
	"strings"
	"testing"
)

func TestBrandingAndLicense(t *testing.T) {
	if AppName != "CFL FileSplitter For Uploading Large Files To Claude" {
		t.Fatalf("unexpected application name: %q", AppName)
	}
	if LicenseID != "MIT" || !strings.HasPrefix(LicenseText, "MIT License\n") {
		t.Fatal("MIT licence must be included")
	}
	if !strings.Contains(LicenseText, Copyright) || !strings.Contains(LicenseText, "Permission is hereby granted") {
		t.Fatal("licence notice or permission missing")
	}
	if PrimaryWebsite != "https://cflab.uk" || DiscoveryWebsite != "https://e-discovery.uk" {
		t.Fatal("requested project websites missing")
	}
}

func TestBrandedInstructionsKeepFormatAndWarnings(t *testing.T) {
	instructions := Instructions("fictional.bin", strings.Repeat("a", 32), strings.Repeat("b", 64), 100, 19, "binary")
	for _, expected := range []string{AppName, PrimaryWebsite, DiscoveryWebsite, "Software licence: MIT", "CFLSPLIT/1", "WARNING", "BINARY MODE", "not digital signatures"} {
		if !strings.Contains(instructions, expected) {
			t.Errorf("instructions missing %q", expected)
		}
	}
}
