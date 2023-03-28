package eip712_sign

import "testing"

func TestVerifyEIP712Signature(t *testing.T) {
	msg := CreateSignMessage(97, "", "0x582a14D1dFE75cc53d25705Dd6EBEB6A1733BB62")
	t.Log(msg)
	result, err := VerifyEIP712Signature(msg, "0x582a14D1dFE75cc53d25705Dd6EBEB6A1733BB62", "0x45745117187b3593741b51d3ce1b1c57225124c1ee1d8b38f1a94bae5a29e45138ed3b7a1eb6fb40dd5ed50af18b59b6267c1f8e6ecf9d8d451c8e11f9ea36a31b")
	if err != nil {
		t.Error(err)
	}
	t.Log(result)
}
