package main

import "testing"

func TestTagValidation(t *testing.T) {
	if !validateTag("latest") {
		t.Fatalf("latest is not a valid tag")
	}
	if !validateTag("0.0.1") {
		t.Fatalf("0.0.0 is not a valid tag")
	}
	if validateTag(":") {
		t.Fatalf(": shouldn't be valid")
	}
	if validateTag("/") {
		t.Fatalf("/ shouldn't be valid")
	}
}

func TestDigestValidation(t *testing.T) {
	if !validateDigest("sha256:f91867a7769436d72e6f4bb68e4e3d240d93d5bc8cd59742298a2a2b3ccf11b7") {
		t.Fatalf("sha256:f91867a7769436d72e6f4bb68e4e3d240d93d5bc8cd59742298a2a2b3ccf11b7 is not a valid digest")
	}
	if validateDigest("0.0.1") {
		t.Fatalf("0.0.0 is a valid digest")
	}

}
