package models

import "testing"

func TestJobStatus(t *testing.T) {
	if JobStatusPending == "" {
		t.Error("JobStatusPending should not be empty")
	}
	if JobStatusProcessing == "" {
		t.Error("JobStatusProcessing should not be empty")
	}
	if JobStatusCompleted == "" {
		t.Error("JobStatusCompleted should not be empty")
	}
}
