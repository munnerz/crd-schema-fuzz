package fuzz

import (
	"io/ioutil"
	"testing"

	gofuzz "github.com/google/gofuzz"
	"k8s.io/apiextensions-apiserver/pkg/apis/apiextensions"
	apiextensionsinstall "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/install"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
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

func Fuzz(t *testing.T, scheme *runtime.Scheme, fuzzer *gofuzz.Fuzzer, crd *apiextensionsv1.CustomResourceDefinition) {
	gk := schema.GroupKind{
		Group: crd.Spec.Group,
		Kind:  crd.Spec.Names.Kind,
	}

	internalcrd := &apiextensions.CustomResourceDefinition{}
	if err := internalScheme.Convert(crd, internalcrd, runtime.InternalGroupVersioner); err != nil {
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

func DecodeFile(t *testing.T, path string) *apiextensionsv1.CustomResourceDefinition {
	groupVersioner := schema.GroupVersions([]schema.GroupVersion{apiextensionsv1.SchemeGroupVersion})
	serializer := apijson.NewSerializerWithOptions(apijson.DefaultMetaFactory, internalScheme, internalScheme, apijson.SerializerOptions{
		Yaml: true,
	})
	convertor := runtime.UnsafeObjectConvertor(internalScheme)
	codec := versioning.NewCodec(serializer, serializer, convertor, internalScheme, internalScheme, internalScheme, groupVersioner, runtime.InternalGroupVersioner, internalScheme.Name())

	data, err := ioutil.ReadFile(path)
	if err != nil {
		t.Fatalf("Failed to read CRD input file %q: %v", path, err)
		return nil
	}

	crd := &apiextensionsv1.CustomResourceDefinition{}
	if _, _, err := codec.Decode(data, nil, crd); err != nil {
		t.Fatalf("Failed to decode CRD data: %v", err)
		return nil
	}

	return crd
}
