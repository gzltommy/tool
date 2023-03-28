package randstr

import "testing"

func TestRandStr(t *testing.T) {
	for i := 0; i < 100; i++ {
		t.Log(RandStringBytesMaskImprSrc(64))
	}
}
