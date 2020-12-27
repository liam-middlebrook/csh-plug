package csh_auth

import (
	"errors"
	oidc "github.com/coreos/go-oidc"
	jwt "github.com/dgrijalva/jwt-go"
	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
	"golang.org/x/net/context"
	"golang.org/x/oauth2"
	"net/http"
	"time"
)

const AuthKey = "cshauth"
const CookieName = "Auth"
const ProviderURI = "https://sso.csh.rit.edu/auth/realms/csh"

// =================
//	  structs
// =================

type CSHAuth struct {
	clientID         string
	clientSecret     string
	secret           string
	state            string
	server_host      string
	redirect_uri     string
	authenticate_uri string

	config   oauth2.Config // this guy changes a bit, weird
	ctx      context.Context
	provider *oidc.Provider
}

type CSHClaims struct {
	Token    string      `json:"token"`
	UserInfo CSHUserInfo `"json:user_info"`
	jwt.StandardClaims
}

type CSHUserInfo struct {
	Subject       string `json:"uuid"`
	Profile       string `json:"profile"`
	Email         string `json:"email"`
	EmailVerified bool   `json:"email_verified"`
	// contains filtered or unexported fields
	Username string `json:"preferred_username"`
	FullName string `json:"name"`
}

// =================
//	auth helper
// =================

func (auth *CSHAuth) AuthWrapper(page gin.HandlerFunc) gin.HandlerFunc {
	return gin.HandlerFunc(func(c *gin.Context) {
		cookie, err := c.Cookie(CookieName)
		if err != nil || cookie == "" {
			log.Info("cookie not found")
			c.Redirect(http.StatusFound, auth.authenticate_uri+"?referer="+c.Request.URL.String())
			return
		}

		token, err := jwt.ParseWithClaims(cookie, &CSHClaims{}, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, errors.New("unexpected signing method")
			}
			return []byte(auth.secret), nil
		})
		if err != nil {
			log.Error("token failure")
			return
		}

		if claims, ok := token.Claims.(*CSHClaims); ok && token.Valid {
			// add in user info data
			c.Set(AuthKey, *claims)
			// call the wrapped func
			page(c)
		} else {
			log.Error("claim parsing failure")
		}
	})
}

func (auth *CSHAuth) AuthRequest(c *gin.Context) {
	// Thrash this so we don't get additive weirdness
	auth.config.RedirectURL = auth.redirect_uri + "?referer=" + c.Query("referer")
	c.Redirect(http.StatusFound, auth.config.AuthCodeURL(auth.state))
}

func (auth *CSHAuth) AuthCallback(c *gin.Context) {
	if c.Query("state") != auth.state {
		log.Error("state does not match")
		return
	}
	oauth2Token, err := auth.config.Exchange(auth.ctx, c.Query("code"))
	if err != nil {
		log.Error("failed to exchange token")
		return
	}
	userInfo := &CSHUserInfo{}
	oidcUserInfo, err := auth.provider.UserInfo(auth.ctx, oauth2.StaticTokenSource(oauth2Token))
	if err != nil {
		log.Error("failed to get userinfo")
	}
	oidcUserInfo.Claims(userInfo)
	if err != nil {
		log.Error("failed to marshal userinfo")
	}

	expireToken := time.Now().Add(time.Hour * 1).Unix()
	expireCookie := 3600
	claims := CSHClaims{
		oauth2Token.AccessToken,
		*userInfo,
		jwt.StandardClaims{
			ExpiresAt: expireToken,
			Issuer:    auth.server_host,
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signedToken, err := token.SignedString([]byte(auth.secret))

	c.SetCookie(CookieName, signedToken, int(expireCookie), "", "", false, true)
	c.Redirect(http.StatusFound, c.Query("referer"))
}

func (auth *CSHAuth) Init(clientID, clientSecret, secret, state, server_host, redirect_uri, auth_uri string) {
	auth.clientID = clientID
	auth.clientSecret = clientSecret
	auth.secret = secret
	auth.state = state
	auth.server_host = server_host
	auth.redirect_uri = redirect_uri
	auth.authenticate_uri = auth_uri

	var err error
	auth.ctx = context.Background()
	auth.provider, err = oidc.NewProvider(auth.ctx, ProviderURI)
	if err != nil {
		log.Error("Failed to Create oidc Provider")
	}
	log.Info(auth.authenticate_uri)
	auth.config = oauth2.Config{
		ClientID:     auth.clientID,
		ClientSecret: auth.clientSecret,
		Endpoint:     auth.provider.Endpoint(),
		RedirectURL:  auth.redirect_uri,
		Scopes:       []string{oidc.ScopeOpenID, "profile", "email"},
	}
}

func (auth *CSHAuth) AuthLogout(c *gin.Context) {
	c.SetCookie(CookieName, "", 0, "", "", false, true)
	c.Redirect(http.StatusFound, ProviderURI+"/protocol/openid-connect/logout?redirect_uri=http://"+auth.server_host+"/")
}
