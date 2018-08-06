package main

import (
	"flag"
	"github.com/gin-gonic/gin"
	csh_auth "github.com/liam-middlebrook/csh-auth"
	log "github.com/sirupsen/logrus"
	"os"
)

type PlugApplication struct {

	// Internal
	// --------

	// Service Connections
	db     DBConnection
	ldap   LDAPConnection
	s3     S3Connection
	router *gin.Engine
	auth   csh_auth.CSHAuth

	// Service Connection Credentials
	base_path        string
	auth_login_route string
}

func (a PlugApplication) Init(
	db_uri,
	s3_host,
	s3_access_id,
	s3_secret_key,
	ldap_host,
	ldap_bind_dn,
	ldap_bind_pw,
	base_path,
	auth_client_id,
	auth_client_secret,
	auth_jwt_secret,
	auth_state,
	auth_server_host,
	auth_redirect_uri,
	auth_login_route string) {

	// Database Connection
	a.db.Init(&a, db_uri)

	// S3 Connection
	a.s3.Init(s3_host,
		s3_access_id,
		s3_secret_key)

	// LDAP connection
	a.ldap.Init(&a, ldap_host, ldap_bind_dn, ldap_bind_pw)

	a.base_path = base_path
	a.router = a.createGinEngine()

	a.auth.Init(
		auth_client_id,
		auth_client_secret,
		auth_jwt_secret,
		auth_state,
		auth_server_host,
		auth_redirect_uri,
		auth_login_route,
	)
	a.auth_login_route = auth_login_route
}

func (a PlugApplication) createGinEngine() *gin.Engine {
	var r *gin.Engine
	r = gin.Default()

	// TODO we should probably look into a different templating solution that
	// allows for inheritance so we can have a navigation and base page layout
	// not be repeated.
	r.LoadHTMLGlob(a.base_path + "templates/*")
	r.Static("/static", a.base_path+"static")

	return r
}

func main() {
	flag.Parse()

	var app PlugApplication

	app.Init(
		os.Getenv("DB_URI"),
		os.Getenv("S3_HOST"),
		os.Getenv("S3_ACCESS_ID"),
		os.Getenv("S3_SECRET_KEY"),
		os.Getenv("LDAP_HOST"),
		os.Getenv("LDAP_BIND_DN"),
		os.Getenv("LDAP_BIND_PW"),
		os.Getenv("BASE_PATH"),
		os.Getenv("csh_auth_client_id"),
		os.Getenv("csh_auth_client_secret"),
		os.Getenv("csh_auth_jwt_secret"),
		os.Getenv("csh_auth_state"),
		os.Getenv("csh_auth_server_host"),
		os.Getenv("csh_auth_redirect_uri"),
		"/auth/login",
	)

	log.Info("Starting server...")

	var r PlugRoutes
	r.app = &app

	app.router.Static("/static", os.Getenv("BASE_PATH")+"static")

	app.router.GET(app.auth_login_route, app.auth.AuthRequest)
	app.router.GET("/auth/redir", app.auth.AuthCallback)
	app.router.GET("/auth/logout", app.auth.AuthLogout)

	app.router.GET("/", app.auth.AuthWrapper(r.index))
	app.router.GET("/data", app.auth.AuthWrapper(r.action))
	app.router.GET("/upload", app.auth.AuthWrapper(r.upload_view))
	app.router.POST("/upload", app.auth.AuthWrapper(r.upload))

	app.router.GET("/admin", app.auth.AuthWrapper(r.get_pending_plugs))
	app.router.POST("/admin", app.auth.AuthWrapper(r.plug_approval))
	app.router.POST("/admin/delete/:id", app.auth.AuthWrapper(r.plug_deletion))

	app.router.Run()
}
