package main

import (
	"testing"
)

func TestIsIPV4(t *testing.T) {
	if !isIPAddress("10.0.0.1") {
		t.Fail()
	}
}

func TestIsIPV6(t *testing.T) {
	if !isIPAddress("2001:db8::68") {
		t.Fail()
	}

	if !isIPAddress("[2001:db8::68]") {
		t.Fail()
	}
}

func TestInStrArray(t *testing.T) {
	a := []string{"test-1", "test-2", "test-3"}

	if inStrArray("test-4", a) {
		t.Fail()
	}

	if !inStrArray("test-2", a) {
		t.Fail()
	}
}

func TestStrArraySub(t *testing.T) {
	a1 := []string{"test-1", "test-2", "test-3"}
	a2 := []string{"test-5", "test-2", "test-4"}

	r := strArraySub(a1, a2)
	if len(r) != 2 && !inStrArray("test-3", r) && !inStrArray("test-1", r) {
		t.Fail()
	}
}
