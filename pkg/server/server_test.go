package server

import (
	"reflect"
	"testing"

	"github.com/google/go-github/github"
)

func TestExtractChangedFiles(t *testing.T) {
	event := &github.PushEvent{
		Commits: []github.PushEventCommit{
			{
				Added:    []string{"file1.txt"},
				Modified: []string{"file2.txt"},
				Removed:  []string{"file3.txt"},
			},
			{
				Added:    []string{"file4.txt"},
				Modified: []string{"file2.txt", "file5.txt"},
				Removed:  []string{"file6.txt"},
			},
		},
	}

	wantAdded := []string{"file1.txt", "file4.txt"}
	wantModified := []string{"file2.txt", "file5.txt"}
	wantDeleted := []string{"file3.txt", "file6.txt"}

	addedFiles, modifiedFiles, deletedFiles := extractChangedFiles(event)

	if !reflect.DeepEqual(addedFiles, wantAdded) {
		t.Errorf("Expected added files: %v, got: %v", wantAdded, addedFiles)
	}

	if !reflect.DeepEqual(modifiedFiles, wantModified) {
		t.Errorf("Expected modified files: %v, got: %v", wantModified, modifiedFiles)
	}

	if !reflect.DeepEqual(deletedFiles, wantDeleted) {
		t.Errorf("Expected deleted files: %v, got: %v", wantDeleted, deletedFiles)
	}
}