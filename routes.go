package main

import (
    "net/http"
    log "github.com/sirupsen/logrus"
    "github.com/gin-gonic/gin"
    csh_auth "github.com/liam-middlebrook/csh-auth"
)

func protectedProfile(c *gin.Context){
    claims, ok := c.Value(csh_auth.AuthKey).(csh_auth.CSHClaims)
    if !ok {
        log.Fatal("error finding claims")
        return
    }
    c.String(http.StatusOK, "uid %s email %s name %s uuid %s", claims.UserInfo.Username, claims.UserInfo.Email, claims.UserInfo.FullName, claims.UserInfo.Subject)
}

func index(c *gin.Context){
    c.Data(http.StatusOK, "text/html", []byte("<html><body><img src=\"/data\"></img><div><p>Upload Feature Coming Soon&trade;</p></div><div><a href=\"https://github.com/liam-middlebrook/csh-plug\">Fork me on GitHub!</a></div></body></html>"))
}

func action(c *gin.Context){
    plug := GetPlug()
    url := S3PresignPlug(plug)

    claims, ok := c.Value(csh_auth.AuthKey).(csh_auth.CSHClaims)
    if !ok {
        log.Fatal("error finding claims")
        return
    }
    log.WithFields(log.Fields{
        "uid": claims.UserInfo.Username,
        "plug_id": plug.ID,
        "plug_s3id": plug.S3ID,
        "presigned_uri": url.String(),
    }).Info("Presigned URI Generated")
    c.Redirect(http.StatusFound, url.String())
}
