package document

import (
	"context"
	"errors"
	"io"
	"net/url"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type S3Config struct {
	Bucket, Region, Endpoint, AccessKeyID, SecretAccessKey string
	PathStyle                                              bool
}

type S3Blob struct {
	client *s3.Client
	bucket string
}

func NewS3Blob(config S3Config) (*S3Blob, error) {
	endpoint, err := url.Parse(config.Endpoint)
	if err != nil || endpoint.Scheme != "https" || endpoint.Host == "" || config.Bucket == "" || config.Region == "" || config.AccessKeyID == "" || config.SecretAccessKey == "" {
		return nil, errors.New("invalid private object storage configuration")
	}
	credentials := aws.NewCredentialsCache(aws.CredentialsProviderFunc(func(context.Context) (aws.Credentials, error) {
		return aws.Credentials{AccessKeyID: config.AccessKeyID, SecretAccessKey: config.SecretAccessKey}, nil
	}))
	client := s3.New(s3.Options{Region: config.Region, BaseEndpoint: aws.String(config.Endpoint), Credentials: credentials, UsePathStyle: config.PathStyle})
	return &S3Blob{client: client, bucket: config.Bucket}, nil
}

func (s *S3Blob) Put(ctx context.Context, key string, body io.Reader, size int64) error {
	_, err := s.client.PutObject(ctx, &s3.PutObjectInput{Bucket: &s.bucket, Key: &key, Body: body, ContentLength: &size})
	return err
}

func (s *S3Blob) Get(ctx context.Context, key string) (io.ReadCloser, error) {
	result, err := s.client.GetObject(ctx, &s3.GetObjectInput{Bucket: &s.bucket, Key: &key})
	if err != nil {
		return nil, err
	}
	return result.Body, nil
}

func (s *S3Blob) Delete(ctx context.Context, key string) error {
	_, err := s.client.DeleteObject(ctx, &s3.DeleteObjectInput{Bucket: &s.bucket, Key: &key})
	return err
}
