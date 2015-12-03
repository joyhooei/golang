package format

import (
	"testing"
)

func TestGenerateJSON(t *testing.T) {
	m := make(map[string]string)
	m["hello"] = "world"
	r := GenerateJSON(m)
	if r != "map" {
		t.Error("expect map, but is " + r)
	}
}
