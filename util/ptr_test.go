package util

import "testing"

func TestPtrAndVal(t *testing.T) {
	s := "hello"
	sPtr := Ptr(s)
	if *sPtr != "hello" {
		t.Errorf("expected 'hello', got %s", *sPtr)
	}
	if Val(sPtr, "default") != "hello" {
		t.Errorf("expected 'hello'")
	}
	if Val[string](nil, "default") != "default" {
		t.Errorf("expected 'default'")
	}

	bTrue := true
	if !BoolVal(&bTrue, false) {
		t.Errorf("expected true")
	}
	if BoolVal(nil, true) != true {
		t.Errorf("expected fallback true")
	}
	if *BoolPtr(true) != true {
		t.Errorf("expected true")
	}
}
