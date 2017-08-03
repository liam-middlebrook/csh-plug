package main

import (
    "net/url"
    "time"
    log "github.com/sirupsen/logrus"
    "github.com/minio/minio-go"
)

var s3 *minio.Client

func S3Init(host, access, secret string) {
    var err error
    s3, err = minio.NewV2(host, access, secret, true)
    if err != nil {
        log.Fatal(err)
    }
}

func S3PresignPlug(plug Plug) *url.URL {
    presignedURL, err := s3.PresignedGetObject("plugs", plug.S3ID, time.Duration(60)*time.Second, make(url.Values))
    if err != nil {
        log.Fatal(err)
    }

    return presignedURL
}
