package eip712_sign

import (
	"crypto/ecdsa"
	"errors"
	"fmt"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/signer/core/apitypes"
)

func SignWithEip721(privateKey *ecdsa.PrivateKey, typedData *apitypes.TypedData) ([]byte, error) {
	if privateKey == nil || typedData == nil {
		return nil, errors.New("invalid parameter")
	}

	// 1、获取需要签名的数据的 Keccak-256 的哈希
	domainSeparator, err := typedData.HashStruct("EIP712Domain", typedData.Domain.Map())
	if err != nil {
		return nil, err
	}
	typedDataHash, err := typedData.HashStruct(typedData.PrimaryType, typedData.Message)
	if err != nil {
		return nil, err
	}
	rawData := []byte(fmt.Sprintf("\x19\x01%s%s", string(domainSeparator), string(typedDataHash)))
	sigHash := crypto.Keccak256(rawData)
	// 2、使用私钥签名哈希，得到签名
	signature, err := crypto.Sign(sigHash, privateKey)
	if err != nil {
		return nil, err
	}
	if signature[64] < 27 {
		signature[64] += 27
	}
	return signature, nil
}
