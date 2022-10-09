package json_patch

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestJsonpatch(t *testing.T) {
	svc := NewJsonpatchSvc()
	require.NotNil(t, svc)

	originalModel := []byte(`{"name": "John", "age": 24, "height": 3.21}`)
	newModel := []byte(`{"name": "John", "age": 25, "height": 3.86}`)

	patch, err1 := svc.CreatePatch(originalModel, newModel)
	require.NoError(t, err1)

	t.Logf("Patch %s", patch)

	targetModel, err2 := svc.ApplyPatch(originalModel, patch)
	require.NoError(t, err2)

	t.Logf("Target %s", targetModel)
	var newDoc interface{}
	json.Unmarshal(newModel, &newDoc)
	var targetDoc interface{}
	json.Unmarshal(targetModel, &targetDoc)

	require.Equal(t, newDoc, targetDoc)
}
