package gcs

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"strings"

	"google.golang.org/api/iterator"

	gcs "cloud.google.com/go/storage"
	"github.com/drone/drone-cache-lib/storage"
	"github.com/dustin/go-humanize"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/option"

	log "github.com/sirupsen/logrus"
)

// Options contains configuration for the GCS connection.
type Options struct {
	JSONKey string
}

type gcStorage struct {
	client *gcs.Client
	opts   *Options
}

// New method creates an implementation of Storage with GCS as the backend.
func New(opts *Options) (storage.Storage, error) {
	ctx := context.Background()

	creds, err := google.CredentialsFromJSON(ctx, []byte(opts.JSONKey), gcs.ScopeReadWrite)
	if err != nil {
		return nil, err
	}

	client, err := gcs.NewClient(ctx, option.WithCredentials(creds))
	if err != nil {
		return nil, err
	}

	return &gcStorage{
		client: client,
		opts:   opts,
	}, nil
}

func (s *gcStorage) Get(p string, dst io.Writer) error {
	ctx := context.Background()
	bucketName, key := splitBucket(p)

	if len(bucketName) == 0 || len(key) == 0 {
		return fmt.Errorf("Invalid path %s", p)
	}

	log.Infof("Retrieving file in %s at %s", bucketName, key)

	bkt := s.client.Bucket(bucketName)
	if _, err := bkt.Attrs(ctx); err != nil {
		return err
	}

	r, err := s.client.Bucket(bucketName).Object(key).NewReader(ctx)
	if err != nil {
		return err
	}
	defer r.Close()

	log.Infof("Copying object from the server")

	numBytes, err := io.Copy(dst, r)

	if err != nil {
		return err
	}

	log.Infof("Downloaded %s from server", humanize.Bytes(uint64(numBytes)))

	return nil
}

func (s *gcStorage) Put(p string, src io.Reader) error {
	ctx := context.Background()
	bucketName, key := splitBucket(p)

	log.Infof("Uploading to bucket %s at %s", bucketName, key)

	if len(bucketName) == 0 || len(key) == 0 {
		ioutil.ReadAll(src)
		return fmt.Errorf("Invalid path %s", p)
	}

	bkt := s.client.Bucket(bucketName)
	if _, err := bkt.Attrs(ctx); err != nil {
		ioutil.ReadAll(src)
		return err
	}

	log.Infof("Putting file in %s at %s", bucketName, key)

	obj := s.client.Bucket(bucketName).Object(key)

	w := obj.NewWriter(ctx)

	numBytes, err := io.Copy(w, src)
	if err != nil {
		ioutil.ReadAll(src)
		return err
	}

	if err := w.Close(); err != nil {
		return err
	}

	attrsToUpdate := gcs.ObjectAttrsToUpdate{ContentType: "application/tar"}
	_, err = obj.Update(ctx, attrsToUpdate)
	if err != nil {
		return err
	}

	log.Infof("Uploaded %s to server", humanize.Bytes(uint64(numBytes)))

	return nil
}

func (s *gcStorage) List(p string) ([]storage.FileEntry, error) {
	ctx := context.Background()
	bucketName, key := splitBucket(p)

	log.Infof("Retrieving objects in bucket %s at %s", bucketName, key)

	if len(bucketName) == 0 || len(key) == 0 {
		return nil, fmt.Errorf("Invalid path %s", p)
	}

	bkt := s.client.Bucket(bucketName)
	if _, err := bkt.Attrs(ctx); err != nil {
		return nil, err
	}

	var objects []storage.FileEntry
	it := s.client.Bucket(bucketName).Objects(ctx, nil)
	for {
		attr, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("Failed to retrieve object %s: %s", attr.Name, err)
		}
		path := bucketName + "/" + attr.Name
		objects = append(objects, storage.FileEntry{
			Path:         path,
			Size:         attr.Size,
			LastModified: attr.Updated,
		})
		log.Debugf("Found object %s: Path=%s Size=%s LastModified=%s", attr.Name, path, attr.Size, attr.Updated)
	}

	log.Infof("Found %d objects in bucket %s at %s", len(objects), bucketName, key)

	return objects, nil
}

func (s *gcStorage) Delete(p string) error {
	ctx := context.Background()
	bucketName, key := splitBucket(p)

	log.Infof("Deleting object in bucket %s at %s", bucketName, key)

	if len(bucketName) == 0 || len(key) == 0 {
		return fmt.Errorf("Invalid path %s", p)
	}

	bkt := s.client.Bucket(bucketName)
	if _, err := bkt.Attrs(ctx); err != nil {
		return err
	}

	o := s.client.Bucket(bucketName).Object(key)
	err := o.Delete(ctx)
	return err
}

func splitBucket(p string) (string, string) {
	// Remove initial forward slash
	full := strings.TrimPrefix(p, "/")

	// Get first index
	i := strings.Index(full, "/")

	if i != -1 && len(full) != i+1 {
		// Bucket names need to be all lower case for the key it doesnt matter
		return strings.ToLower(full[0:i]), full[i+1:]
	}

	return "", ""
}
