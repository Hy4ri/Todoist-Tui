package api

import "testing"

func TestAPIErrorErrorDoesNotExposeBody(t *testing.T) {
	err := (&APIError{StatusCode: 404}).Error()
	if err != "API error (status 404)" {
		t.Fatalf("unexpected error string: %q", err)
	}
}
