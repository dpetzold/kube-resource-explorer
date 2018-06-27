package kube

import (
	"fmt"
	"testing"

	"k8s.io/apimachinery/pkg/api/resource"
)

func TestMemoryResource(t *testing.T) {
	r := resource.NewQuantity(2*1024*1024, resource.BinarySI)
	m := NewMemoryResource(2 * 1024 * 1024)

	if r.Value() != m.Value() {
		t.Errorf("error")
	}

	fmt.Printf("%v\n", m.ToQuantity())

	if r.Value() != m.ToQuantity().Value() {
		t.Errorf("error")
	}

}
