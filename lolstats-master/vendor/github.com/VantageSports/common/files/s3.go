package files

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
)

type S3Provider struct {
	creds           *credentials.Credentials
	regionToClient  map[string]*s3.S3
	bucketsToRegion map[string]string
}

func NewS3Provider(creds *credentials.Credentials, region string) (*S3Provider, error) {
	s := session.New(&aws.Config{
		Credentials: creds,
		Region:      aws.String(region),
	})
	s3Client := s3.New(s)

	return &S3Provider{
		creds: creds,
		regionToClient: map[string]*s3.S3{
			region:    s3Client,
			"default": s3Client,
		},
		bucketsToRegion: map[string]string{},
	}, nil
}

func (s *S3Provider) DownloadTo(src, localDst string) error {
	b, k, err := BucketKey(src)
	if err != nil {
		return err
	}

	req := s3.GetObjectInput{
		Bucket: aws.String(b),
		Key:    aws.String(k),
	}

	client, err := s.client(b)
	if err != nil {
		return err
	}

	res, err := client.GetObject(&req)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	f, err := os.Create(localDst)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = io.Copy(f, res.Body)
	return err
}

func (s *S3Provider) UploadTo(localSrc, dst string, opts ...FileOption) error {
	b, k, err := BucketKey(dst)
	if err != nil {
		return err
	}

	f, err := os.Open(localSrc)
	if err != nil {
		return err
	}
	defer f.Close()

	client, err := s.client(b)
	if err != nil {
		return err
	}

	req := s3.PutObjectInput{
		Bucket:      aws.String(b),
		Key:         aws.String(k),
		ContentType: aws.String(detectContentType(localSrc, f)),
		Body:        f,
	}
	for _, opt := range opts {
		if err = opt.S3(&req); err != nil {
			return err
		}
	}

	_, err = client.PutObject(&req)
	return err
}

func (s *S3Provider) AllowPublicRead(dst string) (string, error) {
	b, k, err := BucketKey(dst)
	if err != nil {
		return "", err
	}

	client, err := s.client(b)
	if err != nil {
		return "", err
	}

	req := s3.PutObjectAclInput{
		Bucket: aws.String(b),
		Key:    aws.String(k),
		ACL:    aws.String("public-read"),
	}
	if _, err = client.PutObjectAcl(&req); err != nil {
		return "", err
	}

	return s.URLFor(dst)
}

func (s *S3Provider) URLFor(src string) (string, error) {
	bucket, key, err := BucketKey(src)
	if err != nil {
		return "", err
	}

	client, err := s.client(bucket)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("https://s3-%s.amazonaws.com/%s/%s", *client.Config.Region, bucket, key), err
}

func (s *S3Provider) Copy(src, dst string, opts ...FileOption) error {
	srcBucket, srcKey, err := BucketKey(src)
	if err != nil {
		return err
	}

	dstBucket, dstKey, err := BucketKey(dst)
	if err != nil {
		return err
	}

	client, err := s.client(srcBucket)
	if err != nil {
		return err
	}

	req := s3.CopyObjectInput{
		CopySource: aws.String(fmt.Sprintf("%s/%s", srcBucket, srcKey)),
		Bucket:     aws.String(dstBucket),
		Key:        aws.String(dstKey),
	}
	for _, opt := range opts {
		if err = opt.S3(&req); err != nil {
			return err
		}
	}
	_, err = client.CopyObject(&req)
	return err
}

func (s *S3Provider) Rename(src, dst string, opts ...FileOption) error {
	if err := s.Copy(src, dst, opts...); err != nil {
		return err
	}
	return s.Remove(src)
}

func (s *S3Provider) Read(src string) ([]byte, error) {
	b, k, err := BucketKey(src)
	if err != nil {
		return nil, err
	}

	req := s3.GetObjectInput{
		Bucket: aws.String(b),
		Key:    aws.String(k),
	}

	client, err := s.client(b)
	if err != nil {
		return nil, err
	}
	res, err := client.GetObject(&req)
	if err != nil {
		return nil, err
	}

	data, err := ioutil.ReadAll(res.Body)
	res.Body.Close()
	return data, err
}

func (s *S3Provider) Remove(path string) error {
	b, k, err := BucketKey(path)
	if err != nil {
		return err
	}

	req := s3.DeleteObjectInput{
		Bucket: aws.String(b),
		Key:    aws.String(k),
	}
	client, err := s.client(b)
	if err != nil {
		return err
	}
	_, err = client.DeleteObject(&req)
	return err
}

func (s *S3Provider) List(path string) ([]string, error) {
	b, k, err := BucketKey(path)
	if err != nil {
		return nil, err
	}
	if !strings.HasSuffix(k, "/") {
		k += "/"
	}

	client, err := s.client(b)
	if err != nil {
		return nil, err
	}

	req := s3.ListObjectsInput{Bucket: aws.String(b), Prefix: aws.String(k)}

	keys := []string{}
	err = client.ListObjectsPages(&req, func(page *s3.ListObjectsOutput, lastPage bool) bool {
		for i := range page.Contents {
			fullKey := fmt.Sprintf("s3://%s/%s", b, *page.Contents[i].Key)
			// Don't include the original 'directory' in the result.
			if fullKey != path && fullKey != path+"/" {
				keys = append(keys, fullKey)
			}
		}
		return true
	})
	return keys, err
}

func (s *S3Provider) Exists(path string) (bool, error) {
	b, k, err := BucketKey(path)
	if err != nil {
		return false, err
	}

	client, err := s.client(b)
	if err != nil {
		return false, err
	}

	req := s3.HeadObjectInput{
		Bucket: aws.String(b),
		Key:    aws.String(k),
	}
	out, err := client.HeadObject(&req)
	if err != nil {
		if reqerr, ok := err.(awserr.RequestFailure); ok && reqerr.StatusCode() == 404 {
			return false, nil
		}
	}
	return out != nil, err
}

func (s *S3Provider) client(bucket string) (*s3.S3, error) {
	region := s.bucketsToRegion[bucket]
	if region == "" {
		cl := s.regionToClient["default"]
		res, err := cl.GetBucketLocation(&s3.GetBucketLocationInput{
			Bucket: aws.String(bucket),
		})
		if err != nil {
			return nil, fmt.Errorf("unable to get bucket location for %s: %v", bucket, err)
		}
		region = *res.LocationConstraint
	}

	client := s.regionToClient[region]
	if client == nil {
		sess := session.New(&aws.Config{
			Credentials: s.creds,
			Region:      aws.String(region),
		})
		client = s3.New(sess)
		s.regionToClient[region] = client
	}

	return client, nil
}
