package aws_s3

import (
	"crypto/tls"
	"fmt"
	"golang.org/x/net/http2"
	"net"
	"net/http"
	"testing"
	"time"
)

func TestCreateTempAccessToken(t *testing.T) {
	accessKey := "AKIAZFT2W5KHWFZ7DMWD"
	secretKey := "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"
	bucket := "ynb-test-2"
	Init(accessKey, secretKey, bucket)
	key, screct, token, _, err := CreateSpaceEditorTempAccessToken("12")
	if err != nil {
		t.Error(err)
	}
	fmt.Println("key:", key, "secret:", screct, "token:", token)
}

func TestUploadImage(t *testing.T) {
	accessKey := "ASIAZFT2W5KHQVVKP45C"
	secretKey := "RkYKpPiFfcIlbxXygw9nS2ZV1L11PoasYvDnxq2H"
	token := "FwoGZXIvYXdzEP7//////////wEaDFk3CfssAiflgUfMqyKIA9CF3MytoN3Kk2Ny+1XpLmVo4U82EMBYNHUcI2yQihDsUMvr0waq2GB9nCCJ0saSC6lr/LEwQ+DZB/0UfgqAwpjR0U2C3u7zkFqUdOlSLpIjdkxjiyCNwclK4B0X0U5T+A8WpN3uBotXbyCM0WBTuP+r19JZi2yEiDGzAqFY2aX2zD376JeIKDWCfGhMt/r/90YwBmukOoXi5PcycWnaTgl4T9c1F1OXloERKSWSxUXYDCgMiH1NtIGx5sVeMp0r4aEHLDcyrBXJphwO4zGRtuWuJhz9CJpADJqCXLgGJqcKKwA9dEI07KbCAUPd9IDQ3hxIQtstlO2G9BmZcmEk5MCzcQCS2RaflcjJHRWdmOuO+7R2cadvYzGXAyNJFth3MVCNxVjDtV75RESVMwgrdWxOixCnc+8hXEHQzRJwgY2sPBuJ4Zhdg1Jp3MttJ8gofvhTuIdwy2UGE2sZ1A0LnPSGDmSzZTy6rUUVAiudBu/ILgcHeJlUE1Q3q5ZHqOvdl1tmUbxKyRQBKKTMgJUGMi3Z5rc8EiItALb9sWbixCZZQwv8+ieoeNSnfXyGS4c8R0oPi0W2/uTmOniVJ6E="
	bucket := "ynb-test-2"
	InitTempUser(accessKey, secretKey, token, bucket)
	contentType, err := UploadImage("https://images.pexels.com/photos/12092806/pexels-photo-12092806.jpeg?cs=srgb&dl=pexels-inchs-12092806.jpg&fm=jpg",
		"2/test2.jpg", nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Log(contentType)
	t.Log("success")
}

func TestSignature(t *testing.T) {
	/*funcSignature := []byte("test()")
	hash := crypto.Keccak256Hash(funcSignature)
	t.Log(hash.Hex())*/
	httpClient := http.Client{
		Transport: &http2.Transport{
			AllowHTTP: true,
			DialTLS: func(network, addr string, cfg *tls.Config) (net.Conn, error) {
				return net.Dial(network, addr)
			},
		},
		Timeout: 120 * time.Second,
	}
	request, err := http.NewRequest("GET", "https://ipfs.io/ipfs/QmYD9AtzyQPjSa9jfZcZq88gSaRssdhGmKqQifUDjGFfXm/sleepy.png", nil)
	if err != nil {
		t.Log(err)
		return
	}
	request.Header.Add("Host", "ipfs.io")
	request.Header.Add("Cache-Control", "no-cache")
	request.Header.Add("Postman-Token", "06468734-9e3c-4945-8438-a6b98861d88e")
	resp, respError := httpClient.Do(request)
	if respError != nil || resp == nil || resp.StatusCode/100 > 3 {
		t.Log(respError)
	}
}

/*func TestDeleteFileByKey(t *testing.T) {
	log.Init(nil)
	err := Init("AKIA54N4EXUREOUO773T", "+NdcgXrPTDC29TGvZ5ClpQMwd6NQsbMCXekIyzhM", "metaverses")
	t.Log(err)
	err = DeleteDirFiles("space_editor/369")
	t.Log(err)
}*/
