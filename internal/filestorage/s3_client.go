package filestorage

import (
	"context"
	"time"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

func MakeS3Client(region string) (c *s3.Client, err error) {
	cfg, err := config.LoadDefaultConfig(context.TODO(), config.WithRegion(region))
	if err != nil {
		return
	}
	c = s3.NewFromConfig(cfg)
	return
}

func GeneratePresignedURL(c *s3.Client, bucket, key string, expiry time.Duration) (string, error) {
	presignClient := s3.NewPresignClient(c)
	req, err := presignClient.PresignGetObject(context.TODO(), &s3.GetObjectInput{
		Bucket: &bucket,
		Key:    &key,
	}, s3.WithPresignExpires(expiry))
	return req.URL, err
}
