package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCalculateCid(t *testing.T) {
	tests := []struct {
		content  []byte
		expected string
	}{
		{
			content:  []byte("{\"What I said\":\"Hellvvvv\"}"),
			expected: "QmNjNs7cVaU4mXdsrG8sdhVR8CteRLFcCzw7XJyXUPsUgi",
		},
		// Add more test cases as needed
	}

	for _, test := range tests {
		actualCid, err := CalculateCid(test.content)
		assert.NoError(t, err, "CalculateCid should not return an error")

		actual := actualCid.String()
		assert.Equal(t, test.expected, actual, "Expected CID does not match actual CID")
	}
}
