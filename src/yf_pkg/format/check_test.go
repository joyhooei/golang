package format

import (
	"testing"
)

func TestEmail(t *testing.T) {
	data := "d322_rr@ggss.xx.yy"
	if !CheckEmail(data) {
		t.Errorf("check %s failed", data)
	}
}

func TestID(t *testing.T) {
	data := "610526198003300019"
	if !CheckIDCard(data) {
		t.Errorf("check %s failed", data)
	}
}

func TestCellphone(t *testing.T) {
	data := "13810592274"
	if !CheckCellphone(data) {
		t.Errorf("check %s failed", data)
	}
}

func TestPassword(t *testing.T) {
	//too short
	data := "111"
	if CheckPassword(data) {
		t.Errorf("check %s failed", data)
	}
	//too long
	data = "3xxdffsdfdffff111"
	if CheckPassword(data) {
		t.Errorf("check %s failed", data)
	}
	//has invalid characters
	data = "3xxdffsdfd f111"
	if CheckPassword(data) {
		t.Errorf("check %s failed", data)
	}
	//valid password
	data = "3xxdffsd_fd(f11"
	if !CheckPassword(data) {
		t.Errorf("check %s failed", data)
	}
	data = "312331"
	if !CheckPassword(data) {
		t.Errorf("check %s failed", data)
	}
	data = "xfsrfvvvvv312331"
	if !CheckPassword(data) {
		t.Errorf("check %s failed", data)
	}
}
