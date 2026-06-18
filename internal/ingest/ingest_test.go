package ingest

import (
	"testing"
)

func TestRunFileUnsupportedExtension(t *testing.T) {
	err := RunFile(nil, "collection.xml", nil)
	if err == nil {
		t.Fatal("expected error for unsupported extension, got nil")
	}
}

func TestRunFileExtensionDispatch(t *testing.T) {
	tests := []struct {
		file    string
		wantErr string
	}{
		{"collection.json", "open"},
		{"collection.txt", "open"},
		{"collection.csv", "open"},
		{"collection.xml", "unsupported"},
	}

	for _, tt := range tests {
		t.Run(tt.file, func(t *testing.T) {
			err := RunFile(nil, tt.file, nil)
			if err == nil {
				t.Fatalf("expected error, got nil")
			}
		})
	}
}
