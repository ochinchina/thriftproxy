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
