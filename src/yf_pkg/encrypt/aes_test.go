package encrypt

import (
	"errors"
	"fmt"
	"testing"
)

func testHelper(raw string, code string) error {
	key := []byte(code)
	result, err := AesEncrypt([]byte(raw), key)
	if err != nil {
		return err
	}
	origData, err := AesDecrypt(result, key)
	if err != nil {
		return err
	}
	if string(origData) != raw {
		return errors.New(fmt.Sprintf("expect %s, but is %s", raw, origData))
	}
	return nil
}

func TestAes(t *testing.T) {
	key := "sfe023f_9fd&fwfl"
	raw := "test string"
	err := testHelper(raw, key)
	if err != nil {
		t.Errorf(err.Error())
	}
	key = "j&fwfl"
	raw = "test string"
	err = testHelper(raw, key)
	if err == nil {
		t.Errorf("block 16 error")
	}
}

func TestAes16(t *testing.T) {
	key := "test"
	data := "I am elife100"
	crypted, err := AesEncrypt16(data, key)
	if err != nil {
		t.Errorf(err.Error())
	}
	result, err := AesDecrypt16(crypted, key)
	if err != nil {
		t.Errorf(err.Error())
	}
	if result != data {
		t.Errorf("expected %s, but is %s", data, result)
	}
}
