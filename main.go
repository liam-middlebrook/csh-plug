package main

import (
	"flag"
	"github.com/gin-gonic/gin"
	csh_auth "github.com/liam-middlebrook/csh-auth"
	log "github.com/sirupsen/logrus"
	"os"
)

func main() {
	flag.Parse()

	log.Info("Starting server...")
	DBInit(os.Getenv("DB_URI"))
	S3Init(
		os.Getenv("S3_HOST"),
		os.Getenv("S3_ACCESS_ID"),
		os.Getenv("S3_SECRET_KEY"),
	)
	// needs to be declared here not inline so provider is global XXX FIXME
	r := gin.Default()
	csh_auth.Init("/auth/login")
	r.GET("/auth/login", csh_auth.AuthRequest)
	r.GET("/auth/redir", csh_auth.AuthCallback)
	r.GET("/auth/logout", csh_auth.AuthLogout)

	r.GET("/", csh_auth.AuthWrapper(index))
	r.GET("/data", csh_auth.AuthWrapper(action))

	r.Run()
}
