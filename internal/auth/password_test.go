package auth

import "testing"

func TestMalformedPasswordHashesAreRejected(t *testing.T) {
	for _, hash := range []string{
		"$argon2id$v=19$m=65536,t=0,p=4$c29tZXNhbHRoZXJlMTIzNA$YWJjZGVmZ2hpamtsbW5vcHFyc3R1dnd4eXowMTIzNDU",
		"$argon2id$v=19$m=65536,t=1,p=256$c29tZXNhbHRoZXJlMTIzNA$YWJjZGVmZ2hpamtsbW5vcHFyc3R1dnd4eXowMTIzNDU",
		"$argon2id$v=19$m=4294967295,t=1,p=4$c29tZXNhbHRoZXJlMTIzNA$YWJjZGVmZ2hpamtsbW5vcHFyc3R1dnd4eXowMTIzNDU",
		"$argon2id$v=16$m=65536,t=1,p=4$c29tZXNhbHRoZXJlMTIzNA$YWJjZGVmZ2hpamtsbW5vcHFyc3R1dnd4eXowMTIzNDU",
		"$argon2id$v=19$m=65536,t=1,p=4$$",
	} {
		if ok, err := VerifyArgon2(hash, "password"); err == nil || ok {
			t.Fatalf("accepted invalid hash %s", hash)
		}
	}
}
