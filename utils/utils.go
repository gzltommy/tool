package utils

import (
	"bytes"
	"crypto/md5"
	"crypto/sha1"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"math"
	"math/big"
	"math/rand"
	"net"
	"net/http"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unsafe"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	"github.com/shopspring/decimal"
	excelizeV2 "github.com/xuri/excelize/v2"
)

const letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

var src = rand.NewSource(time.Now().UnixNano())

const (
	letterIdxBits = 6                    // 6 bits to represent a letter index
	letterIdxMask = 1<<letterIdxBits - 1 // All 1-bits, as many as letterIdxBits
	letterIdxMax  = 63 / letterIdxBits   // # of letter indices fitting in 63 bits
)

func GenLenAlphabetic(n int) string {
	b := make([]byte, n)
	// A src.Int63() generates 63 random bits, enough for letterIdxMax characters!
	for i, cache, remain := n-1, src.Int63(), letterIdxMax; i >= 0; {
		if remain == 0 {
			cache, remain = src.Int63(), letterIdxMax
		}
		if idx := int(cache & letterIdxMask); idx < len(letterBytes) {
			b[i] = letterBytes[idx]
			i--
		}
		cache >>= letterIdxBits
		remain--
	}

	return *(*string)(unsafe.Pointer(&b))
}

type SignatureResp struct {
	Code int      `json:"code"`
	Data []string `json:"data"`
}

func VerifySignature(address, msg, sign string) bool {
	// 签名的字符串进行hash处理
	if !verifySignatureOffline(address, msg, sign) {
		jsonParam := fmt.Sprintf(`{"sig":"%s","message":"%s"}`, sign, msg)
		resp, err := HttpPostJsonRequest("http://verify-server/verify-signature", jsonParam)
		if err != nil {
			return false
		}
		var respOjb SignatureResp
		err = json.Unmarshal([]byte(resp), &respOjb)
		if err != nil || respOjb.Code != 0 {
			return false
		}
		for _, s := range respOjb.Data {
			if strings.EqualFold(s, address) {
				return true
			}
		}
		return false
	}
	return true
}

func verifySignatureOffline(address, msg, sign string) bool {
	signAddress := common.HexToAddress(address)
	message := fmt.Sprintf("\x19Ethereum Signed Message:\n%d%s", len(msg), msg)
	data := []byte(message)
	hash := crypto.Keccak256Hash(data)

	signature := hexutil.MustDecode(sign)
	if signature[64] != 27 && signature[64] != 28 {
		return false
	}
	signature[64] -= 27

	sigPublicKey, err := crypto.SigToPub(hash.Bytes(), signature)
	if err != nil {
		return false
	}

	sigPublicKeyAddr := crypto.PubkeyToAddress(*sigPublicKey)

	return signAddress == sigPublicKeyAddr
}

// HttpPostJsonRequest 发送Post请求，json格式数据
func HttpPostJsonRequest(url string, jsonStr string) (string, error) {
	request, err := http.NewRequest("POST", url, bytes.NewBuffer([]byte(jsonStr)))
	if nil != err {
		return err.Error(), err
	}
	request.Header.Set("Content-Type", "application/json")
	httpClient := &http.Client{}

	response, err := httpClient.Do(request)
	if err != nil {
		return err.Error(), err
	}
	if response != nil {
		defer response.Body.Close()
	}
	request.Close = true
	body, err := ioutil.ReadAll(response.Body)
	if nil != err {
		return err.Error(), err
	}
	return string(body), nil
}

func HttpGetRequest(strURL string, params map[string]interface{}) (string, error) {
	var httpClient *http.Client
	httpClient = &http.Client{
		Transport: &http.Transport{
			Proxy: http.ProxyFromEnvironment,
			Dial: (&net.Dialer{
				Timeout:   10 * time.Second,
				Deadline:  time.Now().Add(6 * time.Second),
				KeepAlive: 4 * time.Second,
			}).Dial,
			TLSHandshakeTimeout: 4 * time.Second,
		},
		Timeout: 10 * time.Second,
	}

	var strRequestURL string
	if nil == params {
		strRequestURL = strURL
	} else {
		strParams := Map2UrlQuery(params)
		strRequestURL = strURL + "?" + strParams
	}

	request, err := http.NewRequest("GET", strRequestURL, nil)
	if nil != err {
		return err.Error(), err
	}
	if request.Header.Get("Content-Type") == "" {
		request.Header.Add("Content-Type", "application/json")
	}
	request.Close = true
	response, err := httpClient.Do(request)
	if err != nil {
		return err.Error(), err
	}
	if response != nil {
		defer response.Body.Close()
	}
	body, err := ioutil.ReadAll(response.Body)
	if nil != err {
		return err.Error(), err
	}
	return string(body), nil
}
func HttpGetRequestWithHeader(strURL string, params map[string]interface{}, headers map[string]string) (body []byte, err error) {
	var httpClient *http.Client
	httpClient = &http.Client{
		Transport: &http.Transport{
			Proxy: http.ProxyFromEnvironment,
			Dial: (&net.Dialer{
				Timeout:   10 * time.Second,
				Deadline:  time.Now().Add(6 * time.Second),
				KeepAlive: 4 * time.Second,
			}).Dial,
			TLSHandshakeTimeout: 4 * time.Second,
		},
		Timeout: 10 * time.Second,
	}

	var strRequestURL string
	if nil == params {
		strRequestURL = strURL
	} else {
		strParams := Map2UrlQuery(params)
		strRequestURL = strURL + "?" + strParams
	}

	request, err := http.NewRequest("GET", strRequestURL, nil)
	if nil != err {
		return body, err
	}
	for k, v := range headers {
		request.Header.Add(k, v)
	}
	if request.Header.Get("Content-Type") == "" {
		request.Header.Add("Content-Type", "application/json")
	}
	request.Close = true
	response, err := httpClient.Do(request)
	if err != nil {
		return body, err
	}
	if response != nil {
		defer response.Body.Close()
	}
	return ioutil.ReadAll(response.Body)
}

// Map2UrlQuery map to url query
func Map2UrlQuery(params map[string]interface{}) string {
	var strParams string
	for key, value := range params {
		strParams += fmt.Sprintf("%v=%v&", key, value)
	}
	if 0 < len(strParams) {
		bm := []rune(strParams)
		strParams = string(bm[:len(bm)-1])
	}
	return strParams
}

func IsValidAddress(iaddress interface{}) bool {
	re := regexp.MustCompile("^0x[0-9a-fA-F]{40}$")
	switch v := iaddress.(type) {
	case string:
		return re.MatchString(v)
	case common.Address:
		return re.MatchString(v.Hex())
	default:
		return false
	}
}

func SigRSV(isig interface{}) ([32]byte, [32]byte, uint8) {
	var sig []byte
	switch v := isig.(type) {
	case []byte:
		sig = v
	case string:
		sig, _ = hexutil.Decode(v)
	}

	sigstr := common.Bytes2Hex(sig)
	rS := sigstr[0:64]
	sS := sigstr[64:128]
	R := [32]byte{}
	S := [32]byte{}
	copy(R[:], common.FromHex(rS))
	copy(S[:], common.FromHex(sS))
	vStr := sigstr[128:130]
	vI, _ := strconv.Atoi(vStr)
	V := uint8(vI + 27)

	return R, S, V
}

func ToDecimal(ivalue interface{}, decimals int) decimal.Decimal {
	value := new(big.Int)
	switch v := ivalue.(type) {
	case string:
		value.SetString(v, 10)
	case *big.Int:
		value = v
	}

	mul := decimal.NewFromFloat(float64(10)).Pow(decimal.NewFromFloat(float64(decimals)))
	num, _ := decimal.NewFromString(value.String())
	result := num.Div(mul)

	return result
}

// ToWei decimals to wei
func ToWei(iamount interface{}, decimals int) *big.Int {
	amount := decimal.NewFromFloat(0)
	switch v := iamount.(type) {
	case string:
		amount, _ = decimal.NewFromString(v)
	case float64:
		amount = decimal.NewFromFloat(v)
	case int64:
		amount = decimal.NewFromFloat(float64(v))
	case decimal.Decimal:
		amount = v
	case *decimal.Decimal:
		amount = *v
	}

	mul := decimal.NewFromFloat(float64(10)).Pow(decimal.NewFromFloat(float64(decimals)))
	result := amount.Mul(mul)

	wei := new(big.Int)
	wei.SetString(result.String(), 10)

	return wei
}

func InSliceString(s []string, v string) bool {
	for _, t := range s {
		if t == v {
			return true
		}
	}
	return false
}

func GetResourceTypeByContentType(content string) string {
	if content == "" {
		return ""
	}
	content = strings.ToLower(content)
	if strings.Contains(content, "image") {
		return "image"
	}
	if strings.Contains(content, "video") {
		return "video"
	}
	if strings.Contains(content, "audio") {
		return "audio"
	}
	return ""
}
func GetImageType(url string) string {
	url = strings.ToLower(url)
	if strings.HasSuffix(url, "png") || strings.HasSuffix(url, "jpg") || strings.HasSuffix(url, "jpeg") ||
		strings.HasSuffix(url, "svg") || strings.HasSuffix(url, "bmp") || strings.HasSuffix(url, "gif") {
		return "image"
	}
	if strings.HasSuffix(url, "mp4") || strings.HasSuffix(url, "webm") || strings.HasSuffix(url, "ogg") {
		return "video"
	}
	return "image"
}

func GetDataType(s interface{}) string {
	switch s.(type) {
	case string:
		return "string"
	case float64:
		return "number"
	default:
		return "string"
	}
}

func Md5(s string) string {
	h := md5.New()
	h.Write([]byte(s))
	return strings.ToLower(hex.EncodeToString(h.Sum(nil)))
}

func GetPageParams4Admin(c *gin.Context) (int, int) {
	pageNo, _ := strconv.Atoi(c.DefaultQuery("pageNo", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "20"))
	return pageNo, pageSize
}

func GetPageParams(c *gin.Context) (int, int) {
	pageNo, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	return pageNo, pageSize
}

func HideMailAddress(mail string) string {
	createXing := func(count int) string {
		s := ""
		for i := 0; i < count; i++ {
			s = s + "*"
		}
		return s
	}
	i := strings.Index(mail, "@")
	if i == -1 {
		return ""
	}
	prefix := mail[0:i]
	if i > 5 {
		prefix = prefix[0:3] + createXing(i-5) + prefix[i-2:]
	} else if i > 3 {
		prefix = prefix[0:2] + createXing(i-3) + prefix[i-1:]
	} else {
		prefix = prefix[0:1] + createXing(i-1)
	}
	return prefix + mail[i:]
}

func FormatDate(t time.Time) string {
	return t.Format("2006-01-02")
}

func FormatTime(t time.Time) string {
	return t.Format("2006-01-02 15:04:05")
}

func IsError(errs []error) bool {
	b := false
	for _, e := range errs {
		if e != nil {
			b = true
			break
		}
	}
	return b
}

func Int64ToBytes(i int64) []byte {
	var buf = make([]byte, 8)
	binary.BigEndian.PutUint64(buf, uint64(i))
	return buf
}

func BytesToInt64(buf []byte) int64 {
	return int64(binary.BigEndian.Uint64(buf))
}

// Float64ToByte Float64转byte
func Float64ToByte(float float64) []byte {
	bits := math.Float64bits(float)
	bytes := make([]byte, 8)
	binary.LittleEndian.PutUint64(bytes, bits)
	return bytes
}

// ByteToFloat64 byte转Float64
func ByteToFloat64(bytes []byte) float64 {
	bits := binary.LittleEndian.Uint64(bytes)
	return math.Float64frombits(bits)
}

func UUID() string {
	result, _ := uuid.NewV4()
	return result.String()
}

const salt = "iuakj72394273lsdH6(HF20hfiunKGG21bn9nah82l-N9hjfb0*n209N)2h9f9198^4hJ1ghd^hj2JHJGU%232js0ybFFGGchbq90ev2-wendlsa09ht54govrndlhHhj9a0&*n29naKL"

func ValidateSign(oriStr string, sign string, timestamp int64) bool {
	if math.Abs(float64(time.Now().Unix()-timestamp)) > 600 {
		return false
	}
	return Md5(oriStr+salt) == sign
}

func Sha1(oriStr string) string {
	sha1 := sha1.New()
	sha1.Write([]byte(oriStr))
	return strings.ToLower(hex.EncodeToString(sha1.Sum([]byte(""))))
}

func IsEqualFloat32(x, y float32) bool {
	return math.Abs(float64(x)-float64(y)) < 0.000001
}

func IsEqualFloat64(x, y float64) bool {
	return math.Abs(x-y) < 0.000001
}

func Decimal(value float64) float64 {
	return math.Trunc(value*1e2) * 1e-2
}

func Round(value float64) float64 {
	return math.Trunc(value*1e2+0.5) * 1e-2
}

// 获取最近一个周一UTC时间对应的时间戳，如果当天是周一，就是上周的周一
func GetLastMondayAtUTCTime(hour, min, sec int) int64 {
	now := time.Now()
	offset := int(time.Monday - now.Weekday())
	fmt.Println(int(time.Monday))
	if offset > 0 {
		offset = -6
	} else if offset == 0 {
		offset = -7
	}
	lastMonday := time.Date(now.Year(), now.Month(), now.Day(), hour, min, sec, 0, time.UTC).AddDate(0, 0, offset)
	return lastMonday.Unix()
}

func DownloadFile(url string) (string, []byte, error) {
	path := strings.Split(url, "/")
	var name string
	if len(path) > 1 {
		name = path[len(path)-1]
	} else {
		return "", nil, errors.New("invalid url")
	}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", nil, err
	}

	client := &http.Client{Timeout: time.Second * 15}
	resp, err := client.Do(req)
	if err != nil {
		return "", nil, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", nil, err
	}
	return name, body, nil
}

func ReadXlsx(r io.Reader) (res [][]string, err error) {
	f, err := excelizeV2.OpenReader(r)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return f.GetRows("Sheet1")
}

func InArray(val interface{}, array interface{}) (exists bool, index int) {
	exists = false
	index = -1
	switch reflect.TypeOf(array).Kind() {
	case reflect.Slice:
		s := reflect.ValueOf(array)
		for i := 0; i < s.Len(); i++ {
			if reflect.DeepEqual(val, s.Index(i).Interface()) == true {
				index = i
				exists = true
				return
			}
		}
	}
	return
}

// IsStandardNum 判断字符是否是一个标准的数字表示(.01 这种不计入标准数字)
func IsStandardNum(s string) bool {
	if strings.HasPrefix(s, ".") {
		return false
	}
	_, err := strconv.ParseFloat(s, 64)
	return err == nil
}

func JsonCleanCharacter(jsonStr string) string {
	jsonStrB := []byte(jsonStr)
	for i, ch := range jsonStrB {

		switch {
		case ch > '~':
			jsonStrB[i] = ' '
		case ch == '\r':
		case ch == '\n':
			jsonStrB[i] = ' '
		case ch == '\t':
		case ch < ' ':
			jsonStrB[i] = ' '
		}
	}
	return string(jsonStrB)
}

func GetFileUrl(baseUrl, srcUrl string) string {
	if srcUrl == "" {
		return ""
	}
	if strings.HasPrefix(srcUrl, "http") {
		return srcUrl
	} else {
		return fmt.Sprintf("%s/%s", strings.TrimRight(baseUrl, "/"), strings.TrimLeft(srcUrl, "/"))
	}
}

func GetFileUrlV2(baseUrl, srcUrl string) string {
	if srcUrl == "" {
		return ""
	}

	getUrl := func() string {
		if strings.HasPrefix(srcUrl, "http") {
			return srcUrl
		} else {
			return fmt.Sprintf("%s/%s", strings.TrimRight(baseUrl, "/"), strings.TrimLeft(srcUrl, "/"))
		}
	}
	return strings.Replace(getUrl(), "http:", "https:", 1)
}

func VerifySignatureSimple(address, msg, sign string) bool {

	signAddress := common.HexToAddress(address)

	message := fmt.Sprintf("\x19Ethereum Signed Message:\n%d%s", len(msg), msg)
	data := []byte(message)
	hash := crypto.Keccak256Hash(data)

	signature := hexutil.MustDecode(sign)
	if signature[64] != 27 && signature[64] != 28 {
		return false
	}
	signature[64] -= 27

	sigPublicKey, err := crypto.SigToPub(hash.Bytes(), signature)
	if err != nil {
		return false
	}

	sigPublicKeyAddr := crypto.PubkeyToAddress(*sigPublicKey)

	return signAddress == sigPublicKeyAddr
}

func GetClientIP(request *http.Request) string {
	var ip string
	ipStr := request.Header.Get("X-Forwarded-For")
	ipArr := strings.Split(ipStr, ",")
	ip = ipArr[0]
	if strings.Contains(ip, "127.0.0.1") || ip == "" {
		ipStr1 := request.Header.Get("X-real-ip")
		ipArr1 := strings.Split(ipStr1, ",")
		ip = ipArr1[0]
	}
	if ip == "" {
		ip = "127.0.0.1"
	}
	return ip
}

func HttpPostRequest(strURL string, jsonParam interface{}, headers map[string]string) ([]byte, int, error) {
	httpClient := &http.Client{}
	param, _ := json.Marshal(jsonParam)
	payload := bytes.NewBuffer(param)
	request, err := http.NewRequest("POST", strURL, payload)
	if nil != err {
		return nil, 0, err
	}
	for k, v := range headers {
		request.Header.Add(k, v)
	}
	if request.Header.Get("Content-Type") == "" {
		request.Header.Add("Content-Type", "application/json")
	}
	response, err := httpClient.Do(request)
	if nil != err {
		return nil, 0, err
	}
	defer response.Body.Close()

	if response.StatusCode == http.StatusOK {
		body, err := io.ReadAll(response.Body)
		if nil != err {
			return nil, response.StatusCode, err
		}

		return body, response.StatusCode, nil
	} else {
		return nil, response.StatusCode, nil
	}
}

func TrimAllSpace(str string) string {
	str = strings.Replace(str, " ", "", -1)  // 去除空格
	str = strings.Replace(str, "\n", "", -1) // 去除换行符
	return str
}

func SkipBOM(buf []byte) (nBuf []byte, err error) {
	if len(buf) >= 4 && isUTF32BigEndianBOM4(buf) {
		return buf[4:], nil
	}
	if len(buf) >= 4 && isUTF32LittleEndianBOM4(buf) {
		return buf[4:], nil
	}
	if len(buf) > 2 && isUTF8BOM3(buf) {
		return buf[3:], nil
	}
	if len(buf) == 2 && isUTF16BigEndianBOM2(buf) {
		return buf[2:], nil
	}
	if len(buf) == 2 && isUTF16LittleEndianBOM2(buf) {
		return buf[2:], nil
	}
	return buf, nil
}

func isUTF32BigEndianBOM4(buf []byte) bool {
	if len(buf) < 4 {
		return false
	}
	return buf[0] == 0x00 && buf[1] == 0x00 && buf[2] == 0xFE && buf[3] == 0xFF
}

func isUTF32LittleEndianBOM4(buf []byte) bool {
	if len(buf) < 4 {
		return false
	}
	return buf[0] == 0xFF && buf[1] == 0xFE && buf[2] == 0x00 && buf[3] == 0x00
}

func isUTF8BOM3(buf []byte) bool {
	if len(buf) < 3 {
		return false
	}
	return buf[0] == 0xEF && buf[1] == 0xBB && buf[2] == 0xBF
}

func isUTF16BigEndianBOM2(buf []byte) bool {
	if len(buf) < 2 {
		return false
	}
	return buf[0] == 0xFE && buf[1] == 0xFF
}

func isUTF16LittleEndianBOM2(buf []byte) bool {
	if len(buf) < 2 {
		return false
	}
	return buf[0] == 0xFF && buf[1] == 0xFE
}

func NDay(lastTimeSec int64, nowTime time.Time) int64 {
	lastTime := time.Unix(lastTimeSec, 0)
	lastTime = time.Date(lastTime.Year(), lastTime.Month(), lastTime.Day(), 0, 0, 0, 0, time.UTC)
	nowTime = time.Date(nowTime.Year(), nowTime.Month(), nowTime.Day(), 0, 0, 0, 0, time.UTC)
	return (nowTime.Unix() - lastTime.Unix()) / (24 * 60 * 60)
}

func NDayX(lastTime, nowTime time.Time) int64 {
	lastTime = time.Date(lastTime.Year(), lastTime.Month(), lastTime.Day(), 0, 0, 0, 0, time.UTC)
	nowTime = time.Date(nowTime.Year(), nowTime.Month(), nowTime.Day(), 0, 0, 0, 0, time.UTC)
	return (nowTime.Unix() - lastTime.Unix()) / (24 * 60 * 60)
}
