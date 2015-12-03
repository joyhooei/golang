package format

import (
	"testing"
)

func TestParseKV(t *testing.T) {
	data := "a=1\r\nb = 4"
	values, err := ParseKV(data, "\r\n")
	if err != nil {
		t.Errorf(err.Error())
	}
	if values["a"] != "1" {
		t.Errorf("expect a=1, but is a=%s", values["a"])
	}
	if values["b "] != " 4" {
		t.Errorf("expect b = 4, but is b =%s", values["b "])
	}
}

func TestParseKVGroup(t *testing.T) {
	data := "a=1\r\nb = 4\r\n\r\na=2\r\nb=3"
	values, err := ParseKVGroup(data, "\r\n", "\r\n\r\n")
	if err != nil {
		t.Errorf(err.Error())
	}
	if values[0]["a"] != "1" {
		t.Errorf("expect a=1, but is a=%s", values[0]["a"])
	}
	if values[0]["b "] != " 4" {
		t.Errorf("expect b = 4, but is b =%s", values[0]["b "])
	}
	if values[1]["a"] != "2" {
		t.Errorf("expect a=2, but is a=%s", values[1]["a"])
	}
	if values[1]["b"] != "3" {
		t.Errorf("expect b =3, but is b =%s", values[1]["b "])
	}
}
