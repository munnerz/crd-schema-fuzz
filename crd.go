package fuzz

import (
	"testing"

	gofuzz "github.com/google/gofuzz"
	"k8s.io/apiextensions-apiserver/pkg/apis/apiextensions"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	structuralschema "k8s.io/apiextensions-apiserver/pkg/apiserver/schema"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

const defaultIterations = 1000

func Fuzz(t *testing.T, scheme *runtime.Scheme, fuzzer *gofuzz.Fuzzer, crd *apiextensionsv1.CustomResourceDefinition) {
	gk := schema.GroupKind{
		Group: crd.Spec.Group,
		Kind:  crd.Spec.Names.Kind,
	}

	internalcrd := &apiextensions.CustomResourceDefinition{}
	if err := scheme.Convert(crd, internalcrd, runtime.InternalGroupVersioner); err != nil {
		t.Fatalf("Failed to convert v1.CustomResourceDefinition to internal type: %v", err)
	}

	for _, vers := range internalcrd.Spec.Versions {
		gvk := gk.WithVersion(vers.Name)
		t.Run(gvk.String(), func(t *testing.T) {
			obj, err := scheme.New(gvk)
			if err != nil {
				t.Errorf("Could not create Object with GroupVersionKind %v: %v", gvk, err)
				return
			}

			structural, err := structuralschema.NewStructural(vers.Schema.OpenAPIV3Schema)
			if err != nil {
				t.Errorf("Failed to construct structural schema: %v", err)
				return
			}

			ObjectNTimes(t, fuzzer, obj, structural, defaultIterations)
		})
	}
}
