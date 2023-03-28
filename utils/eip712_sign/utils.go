package eip712_sign

/*
【绑定邮箱】
Welcome to DEGO! Click to connect email. This request will not trigger a blockchain transaction or cost any gas fees.

【绑定钱包】
Welcome to DEGO! Click to connect wallet. This request will not trigger a blockchain transaction or cost any gas fees.

【创建秘钥】
Welcome to DEGO! Click to create the secret key. This request will not trigger a blockchain transaction or cost any gas fees.

【删除秘钥】
Welcome to DEGO! Click to delete the secret key. This request will not trigger a blockchain transaction or cost any gas fees.

【更新秘钥状态】
Welcome to DEGO! Click to change secret key status. This request will not trigger a blockchain transaction or cost any gas fees.
*/
func VerifySignature(signType string, chainId int, address, sign string) bool {
	var message string
	switch signType {
	case "login":
		message = "Welcome to DEGO! Click to sign in. This request will not trigger a blockchain transaction or cost any gas fees. Your authentication status will reset after 24 hours."
	case "bindAddress":
		message = "Welcome to DEGO! Click to connect wallet. This request will not trigger a blockchain transaction or cost any gas fees."
	case "bindEmail":
		message = "Welcome to DEGO! Click to connect email. This request will not trigger a blockchain transaction or cost any gas fees."
	case "createSecretKey":
		message = "Welcome to DEGO! Click to create the secret key. This request will not trigger a blockchain transaction or cost any gas fees."
	case "deleteSecretKey":
		message = "Welcome to DEGO! Click to delete the secret key. This request will not trigger a blockchain transaction or cost any gas fees."
	case "updateSecretKey":
		message = "Welcome to DEGO! Click to change secret key status. This request will not trigger a blockchain transaction or cost any gas fees."
	default:
		return false
	}
	msg := CreateSignMessage(chainId, message, address)
	result, err := VerifyEIP712Signature(msg, address, sign)
	if err != nil {
		return false
	}
	return result
}
