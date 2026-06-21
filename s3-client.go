package main

import (
	"bytes"
	"context"
	"errors"
	"io"
	"mime"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/smithy-go"
)

type S3Client struct {
	Client *s3.Client
}

func NewS3Client() *S3Client {
	ctx := context.Background()
	awsConfig, err := config.LoadDefaultConfig(ctx,
		config.WithRegion(Cfg.Region),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(Cfg.AccessKey, Cfg.SecretKey, "")),
	)
	if err != nil {
		Logoutput("Unable to load AWS configuration", "error")
		return nil
	}
	s3Client := s3.NewFromConfig(awsConfig, func(o *s3.Options) {
		o.UsePathStyle = true
		if Cfg.Endpoint != "" {
			o.BaseEndpoint = aws.String(Cfg.Endpoint)
		}
	})
	s3conf := "AccessKey: " + Cfg.AccessKey + "\nSecretKey: " + Cfg.SecretKey + "\nBucketName: " + Cfg.BucketName + "\nRegion: " + Cfg.Region + "\nEndpoint: " + Cfg.Endpoint
	if _, err := s3Client.ListBuckets(ctx, &s3.ListBucketsInput{}); err != nil {
		Logoutput("Cannot Create S3 Client, Please check the S3 configuration;\nCurrent configuration: "+s3conf+"\n Error:"+err.Error(), "error")
		return nil
	}
	Logoutput("S3 Client created with configuration: "+s3conf, "info")
	return &S3Client{
		Client: s3Client,
	}
}

func (s *S3Client) ListObjects(key string) (*s3.ListObjectsV2Output, error) {
	Logoutput("ListObjects: "+key, "debug")
	input := &s3.ListObjectsV2Input{
		Bucket:    aws.String(Cfg.BucketName),
		Prefix:    aws.String(key),
		Delimiter: aws.String("/"),
	}
	return s.Client.ListObjectsV2(context.Background(), input)
}

func (s *S3Client) GetObject(key string) (*s3.GetObjectOutput, error) {
	Logoutput("GetObject: "+key, "debug")
	input := &s3.GetObjectInput{
		Bucket: aws.String(Cfg.BucketName),
		Key:    aws.String(key),
	}
	return s.Client.GetObject(context.Background(), input)
}

func (s *S3Client) GetObjectWithFallback(key, fallbackKey string) (*s3.GetObjectOutput, string, error) {
	result, err := s.GetObject(key)
	if err == nil || fallbackKey == "" || fallbackKey == key || !isS3NotFound(err) {
		return result, key, err
	}

	result, fallbackErr := s.GetObject(fallbackKey)
	if fallbackErr != nil {
		return nil, key, err
	}
	return result, fallbackKey, nil
}

func (s *S3Client) PutObject(key string, body io.Reader, providedContentType string) (*s3.PutObjectOutput, error) {
	data, err := io.ReadAll(body)
	if err != nil {
		Logoutput("Unable to read Body for Put Requests", "info")
		return nil, err
	}
	Logoutput("PutObject: "+key, "debug")
	contentType := objectContentType(key, providedContentType, data)
	input := &s3.PutObjectInput{
		Bucket:      aws.String(Cfg.BucketName),
		Key:         aws.String(key),
		Body:        bytes.NewReader(data),
		ContentType: aws.String(contentType),
	}
	if isImageContentType(contentType) {
		input.ContentDisposition = aws.String("inline")
	}
	return s.Client.PutObject(context.Background(), input)
}

func (s *S3Client) DeleteObject(key string) (*s3.DeleteObjectOutput, error) {
	Logoutput("DeleteObject: "+key, "debug")
	input := &s3.DeleteObjectInput{
		Bucket: aws.String(Cfg.BucketName),
		Key:    aws.String(key),
	}
	return s.Client.DeleteObject(context.Background(), input)
}

func (s *S3Client) CopyObject(src, dest string) (*s3.CopyObjectOutput, error) {
	Logoutput("CopyObject: "+src+" to "+dest, "debug")
	input := &s3.CopyObjectInput{
		Bucket:     aws.String(Cfg.BucketName),
		CopySource: aws.String(Cfg.BucketName + "/" + src),
		Key:        aws.String(dest),
	}
	return s.Client.CopyObject(context.Background(), input)
}

func (s *S3Client) MoveObject(src, dest string) (*s3.CopyObjectOutput, error) {
	Logoutput("MoveObject: "+src+" to "+dest, "debug")
	_, err := s.CopyObject(src, dest)
	if err != nil {
		Logoutput("Unable to copy object From Move Requsts", "info")
		return nil, err
	}
	_, err = s.DeleteObject(src)
	if err != nil {
		Logoutput("Unable to delete object From Move Requets", "info")
		return nil, err
	}
	return nil, nil
}

func isS3NotFound(err error) bool {
	if err == nil {
		return false
	}
	var apiErr smithy.APIError
	if errors.As(err, &apiErr) {
		switch apiErr.ErrorCode() {
		case "NoSuchKey", "NotFound":
			return true
		}
	}
	return false
}

func objectContentType(key, providedContentType string, data []byte) string {
	if !isGenericContentType(providedContentType) {
		return providedContentType
	}
	return inferContentType(key, data)
}

func inferContentType(key string, data []byte) string {
	extensionType := contentTypeByExtension(key)
	if len(data) > 0 {
		sniffedType := http.DetectContentType(data)
		if !isGenericContentType(sniffedType) {
			if isSVGContentType(extensionType) && isXMLLikeContentType(sniffedType) {
				return extensionType
			}
			return sniffedType
		}
	}
	if extensionType != "" {
		return extensionType
	}
	return "application/octet-stream"
}

func contentTypeByExtension(key string) string {
	extension := strings.ToLower(filepath.Ext(key))
	if extension == "" {
		return ""
	}
	if contentType := mime.TypeByExtension(extension); contentType != "" {
		return contentType
	}
	switch extension {
	case ".png":
		return "image/png"
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".gif":
		return "image/gif"
	case ".webp":
		return "image/webp"
	case ".svg":
		return "image/svg+xml"
	case ".avif":
		return "image/avif"
	case ".bmp":
		return "image/bmp"
	case ".ico":
		return "image/x-icon"
	case ".tif", ".tiff":
		return "image/tiff"
	case ".heic":
		return "image/heic"
	case ".heif":
		return "image/heif"
	default:
		return ""
	}
}

func isGenericContentType(contentType string) bool {
	mediaType := strings.TrimSpace(strings.ToLower(contentType))
	if parsedType, _, err := mime.ParseMediaType(mediaType); err == nil {
		mediaType = parsedType
	}
	return mediaType == "" || mediaType == "application/octet-stream" || mediaType == "binary/octet-stream"
}

func isImageContentType(contentType string) bool {
	mediaType := strings.TrimSpace(strings.ToLower(contentType))
	if parsedType, _, err := mime.ParseMediaType(mediaType); err == nil {
		mediaType = parsedType
	}
	return strings.HasPrefix(mediaType, "image/")
}

func isSVGContentType(contentType string) bool {
	mediaType := strings.TrimSpace(strings.ToLower(contentType))
	if parsedType, _, err := mime.ParseMediaType(mediaType); err == nil {
		mediaType = parsedType
	}
	return mediaType == "image/svg+xml"
}

func isXMLLikeContentType(contentType string) bool {
	mediaType := strings.TrimSpace(strings.ToLower(contentType))
	if parsedType, _, err := mime.ParseMediaType(mediaType); err == nil {
		mediaType = parsedType
	}
	return mediaType == "text/xml" || mediaType == "application/xml" || mediaType == "text/plain"
}
