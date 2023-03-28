package aws_s3

import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/gabriel-vasile/mimetype"
	"io/ioutil"
	"mime/multipart"
	"net"
	"net/http"
	"net/url"
	"path"
	"path/filepath"
	"regexp"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/endpoints"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/sts"

	"tool-attendance/log"
)

type S3Session struct {
	Sess   *session.Session
	Bucket string
}

var s3Sess *S3Session

func Init(accessKey, secretKey, bucket string) error {
	s3Sess = new(S3Session)
	creds := credentials.NewStaticCredentials(accessKey, secretKey, "")
	s, err := session.NewSession(&aws.Config{
		Credentials: creds,
		Region:      aws.String(endpoints.UsEast2RegionID),
	})
	if err != nil {
		log.Log.Error("aws_s3 Init:", err)
		return err
	}
	s3Sess.Sess = s
	s3Sess.Bucket = bucket
	return nil
}

func InitTempUser(accessKey, secretKey, token, bucket string) error {
	s3Sess = new(S3Session)
	creds := credentials.NewStaticCredentials(accessKey, secretKey, token)
	s, err := session.NewSession(&aws.Config{
		Credentials: creds,
		Region:      aws.String(endpoints.ApSoutheast1RegionID),
	})
	if err != nil {
		log.Log.Error("aws_s3 Init:", err)
		return err
	}
	s3Sess.Sess = s
	s3Sess.Bucket = bucket
	return nil
}

func GetAwsS3Session() *S3Session {
	return s3Sess
}

var svgReg = regexp.MustCompile(".svg")
var mp4Reg = regexp.MustCompile(".mp4")
var contentTypeReg = regexp.MustCompile("(video|image|audio)/.+")

func UploadImage(imagePath string, fileName string, proxy *url.URL) (string, error) {
	httpClient := &http.Client{
		Transport: &http.Transport{
			Proxy: nil,
		},
		Timeout: 120 * time.Second,
	}
	request, err := http.NewRequest("GET", imagePath, nil)
	if err != nil {
		return "", err
	}
	resp, respError := httpClient.Do(request)
	// 默认没有请求到则使用代理进行请求
	if (respError != nil || resp == nil || resp.StatusCode/100 > 3) && proxy != nil {
		proxyURL := http.ProxyURL(proxy)
		httpClientWithProxy := &http.Client{
			Transport: &http.Transport{
				Proxy: proxyURL,
				Dial: (&net.Dialer{
					Timeout:   60 * time.Second,
					Deadline:  time.Now().Add(6 * time.Second),
					KeepAlive: 4 * time.Second,
				}).Dial,
				TLSHandshakeTimeout: 4 * time.Second,
			},
			Timeout: 120 * time.Second,
		}
		request, err = http.NewRequest("GET", imagePath, nil)
		if err != nil {
			return "", err
		}
		resp, respError = httpClientWithProxy.Do(request)
	}

	if respError != nil {
		return "", respError
	}
	if resp == nil || resp.StatusCode/100 > 3 {
		return "", errors.New(fmt.Sprintf("Get Resource Error,StatusCode:%d", resp.StatusCode))
	}
	contentType := resp.Header.Get("content-type")
	if svgReg.MatchString(imagePath) {
		contentType = "image/svg+xml"
	} else if mp4Reg.MatchString(imagePath) {
		contentType = "video/mp4"
	}

	if !contentTypeReg.MatchString(contentType) {
		return "", errors.New("Wrong ContentType:" + contentType)
	}

	defer resp.Body.Close()

	data, err := ioutil.ReadAll(resp.Body)

	s := s3Sess
	var objBody *bytes.Reader
	objBody = bytes.NewReader(data)

	contentLength := resp.ContentLength
	if contentLength == -1 {
		contentLength = objBody.Size()
	}

	_, err = s3.New(s.Sess).PutObject(&s3.PutObjectInput{
		Bucket:      aws.String(s.Bucket),
		Key:         aws.String(fileName),
		Body:        objBody,
		ContentType: aws.String(contentType),
		//ContentLength: aws.Int64(contentLength),
		//ContentDisposition: aws.String("attachment"),
	})
	if err != nil {
		return "", err
	}
	return contentType, nil
}

var (
	allowFileExt = map[string]int{
		".png": 1, ".PNG": 1, ".jpg": 1,
		".JPG": 1, ".jpeg": 1, ".JPEG": 1,
		".gif": 1, ".GIF": 1, ".svg": 1,
		".res": 1, ".ab": 1, ".json": 1,
		".glb": 1}
	allowFileExt2 = map[string]int{
		".png": 1, ".PNG": 1, ".jpg": 1,
		".JPG": 1, ".jpeg": 1, ".JPEG": 1,
		".gif": 1, ".GIF": 1, ".svg": 1,
		".res": 1, ".ab": 1, ".json": 1,
		".glb": 1, ".mp4": 1}
	allowAdminFileExt = map[string]int{
		".png": 1, ".PNG": 1, ".jpg": 1,
		".JPG": 1, ".jpeg": 1, ".JPEG": 1,
		".gif": 1, ".GIF": 1, ".mp4": 1,
		".svg": 1, ".res": 1, ".ab": 1,
		".json": 1, ".zip": 1, ".xlsx": 1,
		".csv": 1}
	NotAllowExt = errors.New("not allow ext")
)

func ApiUploadImage(file multipart.File, fileHeader *multipart.FileHeader, dir string) (string, error) {
	originFilename := filepath.Base(fileHeader.Filename)
	ext := path.Ext(originFilename)
	if _, ok := allowFileExt[ext]; !ok {
		return "", NotAllowExt
	}

	//size := fileHeader.Size
	buffer := make([]byte, fileHeader.Size)
	file.Read(buffer)
	return uploadFileToAws(buffer, dir, ext)
}

func ApiUploadAnimation(file multipart.File, fileHeader *multipart.FileHeader, dir string) (string, error) {
	originFilename := filepath.Base(fileHeader.Filename)
	ext := path.Ext(originFilename)
	if _, ok := allowFileExt2[ext]; !ok {
		return "", NotAllowExt
	}

	//size := fileHeader.Size
	buffer := make([]byte, fileHeader.Size)
	file.Read(buffer)
	return uploadFileToAws(buffer, dir, ext)
}

// nft 头像有些本来就没有后缀名
func UploadNftImage(data []byte, originFilename, dir string) (string, error) {
	ext := path.Ext(originFilename)
	return uploadFileToAws(data, dir, ext)
}

func uploadFileToAws(data []byte, dir, ext string) (string, error) {
	sh := md5.New()
	sh.Write(data)
	imageNameHash := hex.EncodeToString(sh.Sum([]byte("")))

	s := s3Sess
	fileName := fmt.Sprintf("images/%s/%s%s", dir, imageNameHash, ext)
	_, err := s3.New(s.Sess).PutObject(&s3.PutObjectInput{
		Bucket:        aws.String(s.Bucket),
		Key:           aws.String(fileName),
		Body:          bytes.NewReader(data),
		ContentType:   aws.String(mimetype.Detect(data).String()),
		ContentLength: aws.Int64(int64(len(data))),
	})
	if err != nil {
		return "", err
	}
	return fileName, nil
}

func UploadEditorSpaceConfigFile(file []byte, spaceId int64) (string, error) {
	s := s3Sess
	fileName := fmt.Sprintf("space_editor/%d/config.json", spaceId)
	_, err := s3.New(s.Sess).PutObject(&s3.PutObjectInput{
		Bucket:        aws.String(s.Bucket),
		Key:           aws.String(fileName),
		Body:          bytes.NewReader(file),
		ContentType:   aws.String("application/json"),
		ContentLength: aws.Int64(int64(len(file))),
	})
	if err != nil {
		return "", err
	}
	return fileName, nil
}

func UploadEditorSpaceFile(file multipart.File, fileHeader *multipart.FileHeader, spaceId int64, fileType string) (string, error) {
	originFilename := filepath.Base(fileHeader.Filename)
	ext := path.Ext(originFilename)
	if _, ok := allowFileExt[ext]; !ok {
		return "", NotAllowExt
	}
	size := fileHeader.Size
	buffer := make([]byte, size)
	file.Read(buffer)
	sh := md5.New()
	sh.Write(buffer)
	//fileHash := hex.EncodeToString(sh.Sum([]byte("")))
	s := s3Sess
	fileName := fmt.Sprintf("space_editor/%d/%s%s", spaceId, fileType, ext)
	_, err := s3.New(s.Sess).PutObject(&s3.PutObjectInput{
		Bucket:        aws.String(s.Bucket),
		Key:           aws.String(fileName),
		Body:          bytes.NewReader(buffer),
		ContentType:   aws.String(mimetype.Detect(buffer).String()),
		ContentLength: aws.Int64(size),
	})
	if err != nil {
		return "", err
	}
	return fileName, nil
}

func DeleteFileByKey(key string) error {
	_, err := s3.New(s3Sess.Sess).DeleteObject(&s3.DeleteObjectInput{
		Bucket: aws.String(s3Sess.Bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		log.Log.Errorf("Delete S3 File [%s] Error:%v", key, err)
	}
	return err
}

func DeleteDirFiles(dir string) error {
	ss := s3.New(s3Sess.Sess)
	out, err := ss.ListObjects(&s3.ListObjectsInput{
		Bucket: aws.String(s3Sess.Bucket),
		Prefix: aws.String(dir),
	})
	if err != nil {
		log.Log.Error("Get Dir files err:", err)
		return err
	}
	for _, obj := range out.Contents {
		_, err = ss.DeleteObject(&s3.DeleteObjectInput{
			Bucket: aws.String(s3Sess.Bucket),
			Key:    obj.Key,
		})
		if err != nil {
			log.Log.Errorf("Delete S3 File [%s] Error:%v", *obj.Key, err)
		}
	}
	return err
}

func GetFileByKey(key string) ([]byte, error) {
	out, err := s3.New(s3Sess.Sess).GetObject(&s3.GetObjectInput{Bucket: aws.String(s3Sess.Bucket), Key: aws.String(key)})
	if err != nil {
		return nil, err
	}
	return ioutil.ReadAll(out.Body)
}

func AdminUploadResource(file multipart.File, fileHeader *multipart.FileHeader, dir string, checkExt bool) (string, error) {
	originFilename := filepath.Base(fileHeader.Filename)
	ext := path.Ext(originFilename)
	if checkExt {
		if _, ok := allowAdminFileExt[ext]; !ok {
			return "", NotAllowExt
		}
	}
	size := fileHeader.Size
	buffer := make([]byte, size)
	file.Read(buffer)
	sh := md5.New()
	sh.Write(buffer)
	imageNameHash := hex.EncodeToString(sh.Sum([]byte("")))
	s := s3Sess
	if dir == "" {
		dir = "images/official"
	}
	fileName := fmt.Sprintf("%s/%s%s", dir, imageNameHash, ext)
	_, err := s3.New(s.Sess).PutObject(&s3.PutObjectInput{
		Bucket:        aws.String(s.Bucket),
		Key:           aws.String(fileName),
		Body:          bytes.NewReader(buffer),
		ContentType:   aws.String(mimetype.Detect(buffer).String()),
		ContentLength: aws.Int64(size),
	})
	if err != nil {
		return "", err
	}
	return fileName, nil
}

func UploadResource(fileBuffer []byte, size int64, ext, dir string) (string, error) {
	sh := md5.New()
	sh.Write(fileBuffer)
	imageNameHash := hex.EncodeToString(sh.Sum([]byte("")))
	s := s3Sess
	if dir == "" {
		dir = "resource/common"
	}
	//contentType := mimetype.Detect(fileBuffer).String()
	//if download {
	//	contentType = "octet-stream" //
	//}

	fileName := fmt.Sprintf("%s/%s%s", dir, imageNameHash, ext)
	_, err := s3.New(s.Sess).PutObject(&s3.PutObjectInput{
		Bucket:        aws.String(s.Bucket),
		Key:           aws.String(fileName),
		Body:          bytes.NewReader(fileBuffer),
		ContentType:   aws.String(mimetype.Detect(fileBuffer).String()),
		ContentLength: aws.Int64(size),
	})
	if err != nil {
		return "", err
	}
	return fileName, nil
}

func ResizeImage(imageUrl string, fileName string, width, height int) error {
	return nil
}

func UploadByteFile(data []byte, dir, ext string) (string, error) {
	sh := md5.New()
	sh.Write(data)
	imageNameHash := hex.EncodeToString(sh.Sum([]byte("")))

	s := s3Sess
	fileName := fmt.Sprintf("file/%s/%s%s", dir, imageNameHash, ext)
	_, err := s3.New(s.Sess).PutObject(&s3.PutObjectInput{
		Bucket:        aws.String(s.Bucket),
		Key:           aws.String(fileName),
		Body:          bytes.NewReader(data),
		ContentType:   aws.String(http.DetectContentType(data)),
		ContentLength: aws.Int64(int64(len(data))),
	})
	if err != nil {
		return "", err
	}
	return fileName, nil
}

func CreateSpaceEditorTempAccessToken(dir string) (string, string, string, int64, error) {
	uploadPath := fmt.Sprintf("%s/%s", s3Sess.Bucket, dir)
	policy := fmt.Sprintf("{\"Version\": \"2012-10-17\",\"Statement\": [{\"Sid\": \"VisualEditor0\",\"Effect\": \"Allow\",\"Action\": [\"s3:PutObject\",\"s3:GetObject\",\"s3:ListBucketMultipartUploads\",\"s3:AbortMultipartUpload\",\"s3:GetMultiRegionAccessPoint\",\"s3:DeleteMultiRegionAccessPoint\",\"s3:DeleteObject\",\"s3:CreateMultiRegionAccessPoint\",\"s3:ListMultipartUploadParts\"],\"Resource\": [\"arn:aws:s3:::%s\",\"arn:aws:s3:::%s/*\",\"arn:aws:s3::*:accesspoint/*\"]}]}", uploadPath, uploadPath)
	s := s3Sess
	svc := sts.New(s.Sess)
	input := &sts.AssumeRoleInput{
		DurationSeconds: aws.Int64(1800),
		Policy:          aws.String(policy),
		RoleArn:         aws.String("arn:aws:iam::014388475497:role/SpaceFileUploader"),
		RoleSessionName: aws.String("TempAssumeRoleSession"),
	}
	result, err := svc.AssumeRole(input)
	if err != nil {
		fmt.Println("Get TempAccessToken Err:", err.Error())
		return "", "", "", 0, err
	}
	expirTime := time.Now().Unix() + 1800
	return *result.Credentials.AccessKeyId, *result.Credentials.SecretAccessKey, *result.Credentials.SessionToken, expirTime, nil
}

func UploadByteImage(data []byte, fileName string) (string, error) {
	s := s3Sess
	resp, err := s3.New(s.Sess).PutObject(&s3.PutObjectInput{
		Bucket:        aws.String(s.Bucket),
		Key:           aws.String(fileName),
		Body:          bytes.NewReader(data),
		ContentType:   aws.String("image/png"),
		ContentLength: aws.Int64(int64(len(data))),
	})
	if err != nil {
		fmt.Printf("UploadByteImage error: %v\n", err)
		return "", err
	}
	fmt.Printf("UploadByteImage resp: %v\n", resp)
	return fileName, nil
}
