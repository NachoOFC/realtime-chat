package chat

import (
	"errors"
	"testing"
	"time"
)

func TestCursorValidationAndRoundTrip(t *testing.T) {
	original := Message{ID: "3a95a461-04eb-41fb-9697-1c30573922da", CreatedAt: time.Now().UTC()}
	cursor, err := decodeCursor(encodeCursor(original))
	if err != nil || cursor.ID != original.ID || !cursor.At.Equal(original.CreatedAt) {
		t.Fatal(cursor, err)
	}
	for _, input := range []string{"garbage", "e30", "", "eyJpZCI6ImJhZCJ9"} {
		if _, err := decodeCursor(input); !errors.Is(err, ErrValidation) {
			t.Errorf("accepted %q", input)
		}
	}
}
func TestCanonicalIDs(t *testing.T) {
	for _, id := range []string{"", "invalid", "00000000-0000-0000-0000-000000000000", "3A95A461-04EB-41FB-9697-1C30573922DA"} {
		if validID(id) {
			t.Errorf("accepted %q", id)
		}
	}
	if !validID("3a95a461-04eb-41fb-9697-1c30573922da") {
		t.Fatal("rejected UUID")
	}
}
