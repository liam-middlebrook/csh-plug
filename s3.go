package main

import (
	"github.com/minio/minio-go"
	log "github.com/sirupsen/logrus"
	"io"
	"net/url"
	"time"
)

type S3Connection struct {
	con *minio.Client
}

func (c S3Connection) Init(host, access, secret string) {
	s3, err := minio.NewV2(host, access, secret, true)
	if err != nil {
		log.Fatal(err)
	}
	c.con = s3
}

func (c S3Connection) PresignPlug(plug Plug) *url.URL {
	presignedURL, err := c.con.PresignedGetObject("plugs", plug.S3ID, time.Duration(60)*time.Second, make(url.Values))
	if err != nil {
		log.Fatal(err)
	}

	return presignedURL
}

func (c S3Connection) AddFile(plug Plug, data io.Reader, mime string) {
	_, err := c.con.PutObject("plugs", plug.S3ID, data, mime)
	if err != nil {
		log.Error(err)
	}
}

func (c S3Connection) DelFile(plug Plug) {
	err := c.con.RemoveObject("plugs", plug.S3ID)
	if err != nil {
		log.Error(err)
	}
}
