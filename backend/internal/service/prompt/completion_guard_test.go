package prompt

import "testing"

func TestShouldContinueAfterPromise(t *testing.T) {
	if !shouldContinueAfterPromise("Implement the requested feature", "I will implement it and get back to you.", false) {
		t.Fatal("expected implementation promise to continue")
	}
	if !shouldContinueAfterPromise("أضف الميزة المطلوبة", "سأنفذها وسأعود بالنتيجة.", false) {
		t.Fatal("expected Arabic implementation promise to continue")
	}
}

func TestShouldNotContinueAfterPromise(t *testing.T) {
	cases := []struct {
		name     string
		request  string
		response string
		tools    bool
	}{
		{"answer", "What is TypeScript?", "I will explain the concept.", false},
		{"already used tools", "Implement the feature", "I will summarize the changes.", true},
		{"completed answer", "Implement the feature", "The feature is complete and verified.", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if shouldContinueAfterPromise(tc.request, tc.response, tc.tools) {
				t.Fatal("did not expect continuation")
			}
		})
	}
}
