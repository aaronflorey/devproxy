package devproxy

import (
	"bytes"
	"testing"
)

func TestPromptCleanupScope(t *testing.T) {
	t.Parallel()

	in := bytes.NewBufferString("y\nn\ny\nn\n")
	out := &bytes.Buffer{}
	scope, err := promptCleanupScope(in, out)
	if err != nil {
		t.Fatalf("prompt failed: %v", err)
	}
	if !scope.Config || scope.State || !scope.Logs || scope.Certificates {
		t.Fatalf("unexpected scope: %+v", scope)
	}
}
