package fuzz

import (
	"io/ioutil"
	"testing"

	gofuzz "github.com/google/gofuzz"
	"k8s.io/apiextensions-apiserver/pkg/apis/apiextensions"
	apiextensionsinstall "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/install"
	structuralschema "k8s.io/apiextensions-apiserver/pkg/apiserver/schema"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	apijson "k8s.io/apimachinery/pkg/runtime/serializer/json"
	"k8s.io/apimachinery/pkg/runtime/serializer/versioning"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
)

var (
	internalScheme = runtime.NewScheme()
)

func init() {
	utilruntime.Must(metav1.AddMetaToScheme(internalScheme))
	apiextensionsinstall.Install(internalScheme)
}

const defaultIterations = 1000

func Fuzz(t *testing.T, scheme *runtime.Scheme, fuzzer *gofuzz.Fuzzer, crd *apiextensions.CustomResourceDefinition) {
	gk := schema.GroupKind{
		Group: crd.Spec.Group,
		Kind:  crd.Spec.Names.Kind,
	}

	for _, vers := range crd.Spec.Versions {
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

func DecodeFile(t *testing.T, path string) *apiextensions.CustomResourceDefinition {
	serializer := apijson.NewSerializerWithOptions(apijson.DefaultMetaFactory, internalScheme, internalScheme, apijson.SerializerOptions{
		Yaml: true,
	})
	convertor := runtime.UnsafeObjectConvertor(internalScheme)
	codec := versioning.NewCodec(serializer, serializer, convertor, internalScheme, internalScheme, internalScheme, runtime.InternalGroupVersioner, runtime.InternalGroupVersioner, internalScheme.Name())

	data, err := ioutil.ReadFile(path)
	if err != nil {
		t.Fatalf("Failed to read CRD input file %q: %v", path, err)
		return nil
	}

	crd := &apiextensions.CustomResourceDefinition{}
	if _, _, err := codec.Decode(data, nil, crd); err != nil {
		t.Fatalf("Failed to decode CRD data: %v", err)
		return nil
	}

	return crd
}
