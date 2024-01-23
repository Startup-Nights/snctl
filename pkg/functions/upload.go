package functions

import (
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/pkg/errors"
)

type SpacesConfig struct {
	Bucket string
	Region string
	Secret string
	Key    string
}

// Upload all files from a directory to digitalocean spaces. This assumes that
// the files already have already a same name.
func Upload(cfg SpacesConfig, basedir, dir string) error {
	config := &aws.Config{
		Credentials: credentials.NewStaticCredentials(cfg.Key, cfg.Secret, ""),
		Endpoint:    aws.String(fmt.Sprintf("%s.digitaloceanspaces.com:443", strings.TrimSpace(cfg.Region))),
		Region:      aws.String(cfg.Region),
	}

	session, err := session.NewSession(config)
	if err != nil {
		log.Fatal(err)
	}

	client := s3.New(session)

	fmt.Println("=> uploading to digitalocean:")

	return filepath.WalkDir(basedir, func(path string, d fs.DirEntry, err error) error {
		if d.IsDir() {
			return nil
		}

		return upload(client, cfg.Bucket, path, dir)
	})
}

func upload(client *s3.S3, bucket, filename, dir string) error {
	data, err := os.ReadFile(filename)
	if err != nil {
		return errors.Wrap(err, "read file")
	}

	object := s3.PutObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(filepath.Join(dir, filepath.Base(filename))),
		Body:   strings.NewReader(string(data)),
		ACL:    aws.String("public-read"),
	}

	_, err = client.PutObject(&object)
	if err != nil {
		return errors.Wrap(err, "upload to spaces")
	}

	fmt.Println(filepath.Join(dir, filepath.Base(filename)))

	return nil
}
