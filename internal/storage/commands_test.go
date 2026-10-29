package storage

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseS3URI(t *testing.T) {
	tests := []struct {
		input       string
		wantBucket  string
		wantKey     string
		expectError bool
	}{
		{"s3://my-bucket", "my-bucket", "", false},
		{"s3://my-bucket/path/to/object.txt", "my-bucket", "path/to/object.txt", false},
		{"s3://sovereign-bd-fintech/audit.parquet", "sovereign-bd-fintech", "audit.parquet", false},
		{"http://my-bucket", "", "", true},
		{"s3://", "", "", true},
		{"invalid", "", "", true},
	}

	for _, tc := range tests {
		b, k, err := ParseS3URI(tc.input)
		if tc.expectError {
			if err == nil {
				t.Errorf("ParseS3URI(%q): expected error, got nil", tc.input)
			}
		} else {
			if err != nil {
				t.Errorf("ParseS3URI(%q): unexpected error: %v", tc.input, err)
			}
			if b != tc.wantBucket || k != tc.wantKey {
				t.Errorf("ParseS3URI(%q): got bucket=%q, key=%q; want bucket=%q, key=%q",
					tc.input, b, k, tc.wantBucket, tc.wantKey)
			}
		}
	}
}

func TestHandleStorageCommand_Help(t *testing.T) {
	client := NewClient()

	if err := HandleStorageCommand(client, []string{}); err != nil {
		t.Fatalf("unexpected error on empty args: %v", err)
	}
	if err := HandleStorageCommand(client, []string{"help"}); err != nil {
		t.Fatalf("unexpected error on help: %v", err)
	}
	if err := HandleStorageCommand(client, []string{"--help"}); err != nil {
		t.Fatalf("unexpected error on --help: %v", err)
	}
}

func TestHandleStorageCommand_MakeBucketAndList(t *testing.T) {
	client := NewClient()

	// 1. List initial buckets
	if err := HandleStorageCommand(client, []string{"ls"}); err != nil {
		t.Fatalf("ls failed: %v", err)
	}

	// 2. Create new Bangladesh sovereign bucket
	err := HandleStorageCommand(client, []string{"mb", "s3://gov-records-bd", "--sovereignty", "Bangladesh"})
	if err != nil {
		t.Fatalf("mb failed: %v", err)
	}

	// 3. Attempt duplicate create -> should error
	err = HandleStorageCommand(client, []string{"mb", "s3://gov-records-bd"})
	if err == nil {
		t.Fatalf("expected error creating duplicate bucket, got nil")
	}

	// 4. Invalid sovereignty policy
	err = HandleStorageCommand(client, []string{"mb", "s3://invalid-sovereignty", "--sovereignty", "Mars"})
	if err == nil {
		t.Fatalf("expected error for invalid sovereignty, got nil")
	}

	// 5. List objects in sovereign-bd-fintech (has 1 seeded object)
	if err := HandleStorageCommand(client, []string{"ls", "s3://sovereign-bd-fintech"}); err != nil {
		t.Fatalf("ls s3://sovereign-bd-fintech failed: %v", err)
	}

	// 6. List empty bucket
	if err := HandleStorageCommand(client, []string{"ls", "s3://gov-records-bd"}); err != nil {
		t.Fatalf("ls s3://gov-records-bd failed: %v", err)
	}
}

func TestHandleStorageCommand_CopyUploadDownload(t *testing.T) {
	client := NewClient()
	tmpDir := t.TempDir()

	// Create test file
	localSrc := filepath.Join(tmpDir, "test-data.txt")
	testContent := "Aeitron Sovereign Data: NDMA 2026 Invariant Protection."
	if err := os.WriteFile(localSrc, []byte(testContent), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	// 1. Upload to S3
	err := HandleStorageCommand(client, []string{"cp", localSrc, "s3://sovereign-bd-fintech/test-data.txt"})
	if err != nil {
		t.Fatalf("cp upload failed: %v", err)
	}

	// 2. Download from S3 to new local destination
	localDst := filepath.Join(tmpDir, "downloaded.txt")
	err = HandleStorageCommand(client, []string{"cp", "s3://sovereign-bd-fintech/test-data.txt", localDst})
	if err != nil {
		t.Fatalf("cp download failed: %v", err)
	}

	downloaded, err := os.ReadFile(localDst)
	if err != nil {
		t.Fatalf("failed to read downloaded file: %v", err)
	}
	if string(downloaded) != testContent {
		t.Fatalf("download content mismatch: got %q, want %q", string(downloaded), testContent)
	}

	// 3. Delete uploaded object
	err = HandleStorageCommand(client, []string{"rm", "s3://sovereign-bd-fintech/test-data.txt"})
	if err != nil {
		t.Fatalf("rm failed: %v", err)
	}

	// 4. Verify object is deleted
	err = HandleStorageCommand(client, []string{"cp", "s3://sovereign-bd-fintech/test-data.txt", localDst})
	if err == nil {
		t.Fatalf("expected error downloading deleted object, got nil")
	}
}

func TestHandleStorageCommand_RemoveBucket(t *testing.T) {
	client := NewClient()

	// 1. Create a temporary bucket
	bucketName := "temp-empty-bucket"
	err := HandleStorageCommand(client, []string{"mb", "s3://" + bucketName, "--sovereignty", "Global"})
	if err != nil {
		t.Fatalf("mb failed: %v", err)
	}

	// 2. Remove empty bucket -> should succeed
	err = HandleStorageCommand(client, []string{"rb", "s3://" + bucketName})
	if err != nil {
		t.Fatalf("rb failed: %v", err)
	}

	// 3. Attempting to remove non-empty bucket -> should error
	err = HandleStorageCommand(client, []string{"rb", "s3://sovereign-bd-fintech"})
	if err == nil {
		t.Fatalf("expected error removing non-empty bucket, got nil")
	}
}

func TestHandleStorageCommand_ErrorHandling(t *testing.T) {
	client := NewClient()

	// Unknown subcommand
	err := HandleStorageCommand(client, []string{"invalid-subcommand"})
	if err == nil || !strings.Contains(err.Error(), "unknown storage subcommand") {
		t.Fatalf("expected unknown subcommand error, got: %v", err)
	}

	// Missing arguments
	if err := HandleStorageCommand(client, []string{"mb"}); err == nil {
		t.Fatalf("expected error on mb with no args, got nil")
	}
	if err := HandleStorageCommand(client, []string{"cp"}); err == nil {
		t.Fatalf("expected error on cp with no args, got nil")
	}
	if err := HandleStorageCommand(client, []string{"rm"}); err == nil {
		t.Fatalf("expected error on rm with no args, got nil")
	}
	if err := HandleStorageCommand(client, []string{"rb"}); err == nil {
		t.Fatalf("expected error on rb with no args, got nil")
	}
}
