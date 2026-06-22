package main

import (
	"strings"
	"testing"
	"time"
)

func TestNormalizeS3Config(t *testing.T) {
	tests := []struct {
		name         string
		region       string
		endpoint     string
		wantRegion   string
		wantEndpoint string
	}{
		{
			name:         "trims quoted region and endpoint",
			region:       ` "us-east-1" `,
			endpoint:     ` 's3.example.com' `,
			wantRegion:   "us-east-1",
			wantEndpoint: "https://s3.example.com",
		},
		{
			name:         "uses signing region for custom endpoint",
			endpoint:     "https://minio.example.com",
			wantRegion:   "us-east-1",
			wantEndpoint: "https://minio.example.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotRegion, gotEndpoint := normalizeS3Config(tt.region, tt.endpoint)
			if gotRegion != tt.wantRegion || gotEndpoint != tt.wantEndpoint {
				t.Fatalf("normalizeS3Config() = (%q, %q), want (%q, %q)", gotRegion, gotEndpoint, tt.wantRegion, tt.wantEndpoint)
			}
		})
	}
}

func TestRegionValidation(t *testing.T) {
	valid := []string{"us-east-1", "nyc3", "s3.internal"}
	for _, region := range valid {
		if !isValidS3Region(region) {
			t.Fatalf("expected %q to be valid", region)
		}
	}

	invalid := []string{"", "https://s3.example.com", "bad_region"}
	for _, region := range invalid {
		if isValidS3Region(region) {
			t.Fatalf("expected %q to be invalid", region)
		}
	}
}

func TestMaskSecret(t *testing.T) {
	if got := maskSecret("abcd1234wxyz"); got != "abcd****wxyz" {
		t.Fatalf("maskSecret() = %q", got)
	}

	if got := maskSecret("short"); got != "****" {
		t.Fatalf("maskSecret() short value = %q", got)
	}
}

func TestObjectContentType(t *testing.T) {
	if got := objectContentType("image.svg", "application/octet-stream", []byte(`<svg xmlns="http://www.w3.org/2000/svg"></svg>`)); got != "image/svg+xml" {
		t.Fatalf("expected SVG content type, got %q", got)
	}

	if got := objectContentType("photo.bin", "image/jpeg", nil); got != "image/jpeg" {
		t.Fatalf("expected provided content type, got %q", got)
	}

	if got := objectContentType("data.unknown", "", nil); got != "application/octet-stream" {
		t.Fatalf("expected default content type, got %q", got)
	}
}

func TestResponseContentDisposition(t *testing.T) {
	if got := responseContentDisposition("", "image/png"); got != "inline" {
		t.Fatalf("expected inline image disposition, got %q", got)
	}

	if got := responseContentDisposition(`attachment; filename="photo.png"`, "image/png"); got != `inline; filename="photo.png"` {
		t.Fatalf("expected inline image filename disposition, got %q", got)
	}

	if got := responseContentDisposition("attachment", "application/pdf"); got != "attachment" {
		t.Fatalf("expected non-image disposition to stay unchanged, got %q", got)
	}
}

func TestPathHelpers(t *testing.T) {
	if got := canonicalURLPath("folder//file.txt"); got != "/folder/file.txt" {
		t.Fatalf("canonicalURLPath() = %q", got)
	}

	if got := objectKeyFromDestination("https://example.com/folder/file.txt"); got != "folder/file.txt" {
		t.Fatalf("objectKeyFromDestination() = %q", got)
	}

	if got := assetPathFromKey("/folder/file.txt"); got != "/folder/file.txt" {
		t.Fatalf("assetPathFromKey() = %q", got)
	}
}

func TestEscapingHelpers(t *testing.T) {
	if got := htmlAttribute(`folder/"bad"&file`); strings.ContainsAny(got, `"&`) && !strings.Contains(got, "&#34;") {
		t.Fatalf("htmlAttribute() did not escape attribute value: %q", got)
	}

	if got := xmlText(`<folder&file>`); strings.Contains(got, "<") || strings.Contains(got, ">") {
		t.Fatalf("xmlText() did not escape XML text: %q", got)
	}
}

func TestFormatHTTPTime(t *testing.T) {
	timestamp := time.Date(2026, 6, 21, 12, 30, 0, 0, time.FixedZone("EDT", -4*60*60))
	if got := formatHTTPTime(&timestamp); got != "Sun, 21 Jun 2026 16:30:00 GMT" {
		t.Fatalf("formatHTTPTime() = %q", got)
	}
}
