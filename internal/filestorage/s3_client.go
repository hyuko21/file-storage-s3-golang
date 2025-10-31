package filestorage

import (
	"context"

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
