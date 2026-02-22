package models

import "testing"

func TestSendMessageRequestIsStopRequest(t *testing.T) {
	tests := []struct {
		name string
		typ  string
		want bool
	}{
		{name: "stop", typ: "stop", want: true},
		{name: "message", typ: "message", want: false},
		{name: "empty defaults to false", typ: "", want: false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := &SendMessageRequest{Type: tc.typ}
			if got := req.IsStopRequest(); got != tc.want {
				t.Fatalf("IsStopRequest() = %v, want %v", got, tc.want)
			}
		})
	}
}
