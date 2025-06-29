package main

import (
	"io"
	"bytes"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
)

type S3Client struct {
	Client *s3.S3
}

func NewS3Client() *S3Client {
	awsConfig := &aws.Config{
		Region: aws.String(Cfg.Region),
		Endpoint: aws.String(Cfg.Endpoint),
		Credentials: credentials.NewStaticCredentials(Cfg.AccessKey, Cfg.SecretKey, ""),
	}
	awsConfig.S3ForcePathStyle = aws.Bool(true)

	sess, err := session.NewSession(awsConfig)
	if err != nil {
		Logoutput("Unable to create S3 session", "error")
		return nil
	}
	s3Client := s3.New(sess)
	s3conf := "AccessKey: "+Cfg.AccessKey+"\nSecretKey: "+Cfg.SecretKey+"\nBucketName: "+Cfg.BucketName+"\nRegion: "+Cfg.Region+"\nEndpoint: "+Cfg.Endpoint
	if _, err := s3Client.ListBuckets(nil); err != nil {
		Logoutput("Cannot Create S3 Client, Please check the S3 configuration;\nCurrent configuration: "+s3conf, "error")
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
		Bucket: aws.String(Cfg.BucketName),
		Prefix: aws.String(key),
		Delimiter: aws.String("/"),
	}
	return s.Client.ListObjectsV2(input)
}

func (s *S3Client) GetObject(key string) (*s3.GetObjectOutput, error) {
	Logoutput("GetObject: "+key, "debug")
	input := &s3.GetObjectInput{
		Bucket: aws.String(Cfg.BucketName),
		Key: aws.String(key),
	}
	return s.Client.GetObject(input)
}

func (s *S3Client) PutObject(key string, body io.Reader) (*s3.PutObjectOutput, error) {
	data, err := io.ReadAll(body)
	if err != nil {
		Logoutput("Unable to read Body for Put Requests", "info")
		return nil, err
	}
	Logoutput("PutObject: "+key, "debug")
	input := &s3.PutObjectInput{
		Bucket: aws.String(Cfg.BucketName),
		Key:    aws.String(key),
		Body:   bytes.NewReader(data),
	}
	return s.Client.PutObject(input)
}

func (s *S3Client) DeleteObject(key string) (*s3.DeleteObjectOutput, error) {
	Logoutput("DeleteObject: "+key, "debug")
	input := &s3.DeleteObjectInput{
		Bucket: aws.String(Cfg.BucketName),
		Key: aws.String(key),
	}
	return s.Client.DeleteObject(input)
}

func (s *S3Client) CopyObject(src, dest string) (*s3.CopyObjectOutput, error) {
	Logoutput("CopyObject: "+src+" to "+dest, "debug")
	input := &s3.CopyObjectInput{
		Bucket: aws.String(Cfg.BucketName),
		CopySource: aws.String(Cfg.BucketName + "/" + src),
		Key: aws.String(dest),
	}
	return s.Client.CopyObject(input)
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
