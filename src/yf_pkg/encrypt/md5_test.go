package encrypt

import (
	"testing"
)

func TestMD5(t *testing.T) {
	data := "test string"
	md5 := MD5Sum(data)
	expect := "6f8db599de986fab7a21625b7916589c"
	if md5 != expect {
		t.Errorf("expect %s, but is %s", expect, md5)
	}
}
