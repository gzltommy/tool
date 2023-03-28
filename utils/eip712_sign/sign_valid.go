package eip712_sign

import (
	"encoding/json"
	"fmt"
	ethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/signer/core/apitypes"
	"github.com/storyicon/sigverify"
)

const typedDataExample = `
{
    "types": {
 		"EIP712Domain": [
            {
                "name": "name",
                "type": "string"
            },
			{
				"name": "version",
                "type": "string"
			},
            {
                "name": "chainId",
                "type": "uint256"
            }
        ],
	   "SignIn": [
			{ "name": "message", "type": "string"},
			{ "name": "Wallet address", "type": "address"}
		]
    },
    "domain": {
        "name": "DEGO",
		"version": "1",
		"chainId": "%d"
    },
    "primaryType": "%s",
    "message": %s
}
`

func CreateSignMessage(chainId int, message, walletAddress string) string {
	msg := `
		{
			 "message":"%s",
			 "Wallet address": "%s"
		}
	`
	return fmt.Sprintf(typedDataExample, chainId, "SignIn", fmt.Sprintf(msg, message, ethcommon.HexToAddress(walletAddress)))
}

func VerifyEIP712Signature(typedJsonData, address, signature string) (bool, error) {
	var typedData apitypes.TypedData
	if err := json.Unmarshal([]byte(typedJsonData), &typedData); err != nil {
		return false, err
	}
	valid, err := sigverify.VerifyTypedDataHexSignatureEx(
		ethcommon.HexToAddress(address),
		typedData,
		signature,
	)
	return valid, err
}
