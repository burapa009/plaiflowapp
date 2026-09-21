package job

import (
	"testing"
	"time"
)

func TestArtifactTokenExpiresAndCannotBeAltered(t *testing.T) {
	now := time.Date(2026, 9, 21, 0, 0, 0, 0, time.UTC)
	signer, _ := NewArtifactToken([]byte("01234567890123456789012345678901"))
	token, err := signer.Sign("job-1", "org-1", "user-1", now)
	if err != nil {
		t.Fatal(err)
	}
	jobID, orgID, userID, err := signer.Verify(token, now.Add(14*time.Minute))
	if err != nil || jobID != "job-1" || orgID != "org-1" || userID != "user-1" {
		t.Fatalf("job=%s org=%s user=%s err=%v", jobID, orgID, userID, err)
	}
	if _, _, _, err = signer.Verify(token+"x", now); err == nil {
		t.Fatal("altered token accepted")
	}
	if _, _, _, err = signer.Verify(token, now.Add(16*time.Minute)); err == nil {
		t.Fatal("expired token accepted")
	}
}
