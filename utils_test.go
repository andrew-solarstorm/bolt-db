package bolt_db

import (
	"testing"
)

type TestData struct {
	Name  string
	Value int
	Items []string
}

func TestDefaultEnc_Success(t *testing.T) {
	data := TestData{
		Name:  "test",
		Value: 42,
		Items: []string{"a", "b", "c"},
	}

	encoded, err := defaultEnc(&data)
	if err != nil {
		t.Fatalf("Encoding failed: %v", err)
	}

	if len(encoded) == 0 {
		t.Error("Encoded data should not be empty")
	}
}

func TestDefaultDec_Success(t *testing.T) {
	original := TestData{
		Name:  "test",
		Value: 99,
		Items: []string{"x", "y"},
	}

	encoded, err := defaultEnc(&original)
	if err != nil {
		t.Fatalf("Encoding failed: %v", err)
	}

	var decoded TestData
	err = defaultDec(encoded, &decoded)
	if err != nil {
		t.Fatalf("Decoding failed: %v", err)
	}

	if decoded.Name != original.Name {
		t.Errorf("Name mismatch: expected %s, got %s", original.Name, decoded.Name)
	}

	if decoded.Value != original.Value {
		t.Errorf("Value mismatch: expected %d, got %d", original.Value, decoded.Value)
	}

	if len(decoded.Items) != len(original.Items) {
		t.Errorf("Items length mismatch: expected %d, got %d", len(original.Items), len(decoded.Items))
	}
}

func TestDefaultDec_EmptyData(t *testing.T) {
	var decoded TestData
	err := defaultDec([]byte{}, &decoded)
	if err == nil {
		t.Error("Expected error for empty data")
	}
}

func TestDefaultDec_InvalidData(t *testing.T) {
	var decoded TestData
	err := defaultDec([]byte{1, 2, 3, 4, 5}, &decoded)
	if err == nil {
		t.Error("Expected error for invalid data")
	}
}

func TestEncodeDecode_RoundTrip(t *testing.T) {
	testCases := []TestData{
		{Name: "simple", Value: 1, Items: nil},
		{Name: "with items", Value: 100, Items: []string{"a", "b"}},
		{Name: "", Value: 0, Items: []string{}},
	}

	for _, tc := range testCases {
		encoded, err := defaultEnc(&tc)
		if err != nil {
			t.Fatalf("Encoding failed for %+v: %v", tc, err)
		}

		var decoded TestData
		err = defaultDec(encoded, &decoded)
		if err != nil {
			t.Fatalf("Decoding failed for %+v: %v", tc, err)
		}

		if decoded.Name != tc.Name || decoded.Value != tc.Value {
			t.Errorf("Round trip failed: expected %+v, got %+v", tc, decoded)
		}
	}
}

func TestDefaultEnc_ComplexStruct(t *testing.T) {
	type ComplexStruct struct {
		Map    map[string]int
		Nested *TestData
		Slice  []byte
	}

	complex := ComplexStruct{
		Map: map[string]int{"a": 1, "b": 2},
		Nested: &TestData{
			Name:  "nested",
			Value: 10,
			Items: []string{"item"},
		},
		Slice: []byte{1, 2, 3, 4, 5},
	}

	encoded, err := defaultEnc(&complex)
	if err != nil {
		t.Fatalf("Encoding complex struct failed: %v", err)
	}

	var decoded ComplexStruct
	err = defaultDec(encoded, &decoded)
	if err != nil {
		t.Fatalf("Decoding complex struct failed: %v", err)
	}

	if decoded.Map["a"] != 1 || decoded.Map["b"] != 2 {
		t.Error("Map decoding failed")
	}

	if decoded.Nested.Name != "nested" {
		t.Error("Nested struct decoding failed")
	}

	if len(decoded.Slice) != 5 {
		t.Error("Slice decoding failed")
	}
}
