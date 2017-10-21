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
	DBInit("DB_URI")
	S3Init(
		os.Getenv("S3_HOST"),
		os.Getenv("S3_ACCESS_ID"),
		os.Getenv("S3_SECRET_KEY"),
	)

	LDAPInit(
		os.Getenv("LDAP_HOST"),
		os.Getenv("LDAP_BIND_DN"),
		os.Getenv("LDAP_BIND_PW"),
	)

	// needs to be declared here not inline so provider is global XXX FIXME
	r := gin.Default()
	r.LoadHTMLGlob(os.Getenv("BASE_PATH") + "templates/*")

	csh := csh_auth.CSHAuth{}
	csh.Init(
		os.Getenv("csh_auth_client_id"),
		os.Getenv("csh_auth_client_secret"),
		os.Getenv("csh_auth_jwt_secret"),
		os.Getenv("csh_auth_state"),
		os.Getenv("csh_auth_server_host"),
		os.Getenv("csh_auth_redirect_uri"),
		"/auth/login",
	)
	r.GET("/auth/login", csh.AuthRequest)
	r.GET("/auth/redir", csh.AuthCallback)
	r.GET("/auth/logout", csh.AuthLogout)

	r.GET("/", csh.AuthWrapper(index))
	r.GET("/data", csh.AuthWrapper(action))
	r.GET("/upload", csh.AuthWrapper(upload_view))
	r.POST("/upload", csh.AuthWrapper(upload))

	r.GET("/admin", csh.AuthWrapper(get_pending_plugs))
	r.POST("/admin", csh.AuthWrapper(plug_approval))

	r.Run()
}
