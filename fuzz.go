package fuzz

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	gofuzz "github.com/google/gofuzz"
	structuralschema "k8s.io/apiextensions-apiserver/pkg/apiserver/schema"
	structuralpruning "k8s.io/apiextensions-apiserver/pkg/apiserver/schema/pruning"
	"k8s.io/apimachinery/pkg/runtime"
)

func ObjectNTimes(t *testing.T, fuzzer *gofuzz.Fuzzer, obj runtime.Object, schema *structuralschema.Structural, iterations int) {
	t.Logf("Running CRD schema pruning fuzz test for object %v", obj.GetObjectKind())
	for i := 0; i < iterations; i++ {
		fuzzed := obj.DeepCopyObject()
		fuzzer.Fuzz(fuzzed)
		pruned := fuzzed.DeepCopyObject()
		structuralpruning.Prune(pruned, schema, true)
		if !cmp.Equal(fuzzed, pruned) {
			t.Errorf("Failed fuzz test, difference: %v", cmp.Diff(fuzzed, pruned))
		}
		t.Logf("Passed fuzz test iteration %d", i)
	}
}
