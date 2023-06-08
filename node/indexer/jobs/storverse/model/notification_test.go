package storverse
import (
	"testing"
)

func TestTruncateMessageContent(t *testing.T) {
	var tests = []struct {
		input    string
		expected string
	}{
		{"Hello%20world!", "Hello world!"},
		{"This%20is%20a%20very%20long%20sentence.", "This is a very..."},
		{"short", "short"},
		{"%3Chtml%3E", "<html>"},
	}

	for _, test := range tests {
		if output := truncateMessageContent(test.input); output != test.expected {
			t.Errorf("Test failed: input: %v, output: %v, expected: %v", test.input, output, test.expected)
		}
	}
}