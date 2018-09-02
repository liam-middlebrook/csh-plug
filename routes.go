package main

import (
	"github.com/gin-gonic/gin"
	csh_auth "github.com/liam-middlebrook/csh-auth"
	log "github.com/sirupsen/logrus"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type PlugRoutes struct {
	app *PlugApplication
}

func protectedProfile(c *gin.Context) {
	claims, ok := c.Value(csh_auth.AuthKey).(csh_auth.CSHClaims)
	if !ok {
		log.Fatal("error finding claims")
		return
	}
	c.String(http.StatusOK, "uid %s email %s name %s uuid %s", claims.UserInfo.Username, claims.UserInfo.Email, claims.UserInfo.FullName, claims.UserInfo.Subject)
}

func (r PlugRoutes) index(c *gin.Context) {
	c.Redirect(http.StatusFound, "/upload")
}

func (r PlugRoutes) action(c *gin.Context) {
	plug := r.app.db.GetPlug()
	url := r.app.s3.PresignPlug(plug)

	claims, ok := c.Value(csh_auth.AuthKey).(csh_auth.CSHClaims)
	if !ok {
		log.Fatal("error finding claims")
		return
	}
	log.WithFields(log.Fields{
		"uid":           claims.UserInfo.Username,
		"plug_id":       plug.ID,
		"plug_s3id":     plug.S3ID,
		"presigned_uri": url.String(),
	}).Info("Presigned URI Generated")
	r.app.db.AddLog(13, c.GetHeader("Referer"))
	c.Redirect(http.StatusFound, url.String())
}

func (r PlugRoutes) upload(c *gin.Context) {
	plug := Plug{}

	claims, ok := c.Value(csh_auth.AuthKey).(csh_auth.CSHClaims)
	if !ok {
		log.Fatal("error finding claims")
		return
	}

	plug.Owner = claims.UserInfo.Username

	file, err := c.FormFile("fileUpload")
	if err != nil {
		log.Error(err)
		c.String(http.StatusBadRequest, "Error Reading File")
		return
	}
	data, err := file.Open()
	if err != nil {
		log.Error(err)
		c.String(http.StatusBadRequest, "Error Reading File")
		return
	}
	defer data.Close()
	imageData, _, err := image.DecodeConfig(data)
	if err != nil {
		log.Error(err)
		c.String(http.StatusUnsupportedMediaType, "Please upload either a JPG or PNG!")
		return
	}
	data.Seek(0, 0)
	if imageData.Width == 728 && imageData.Height == 200 {
		numCredits, err := strconv.Atoi(c.PostForm("numCredits"))
		if err != nil {
			log.Error(err)
			c.String(http.StatusUnsupportedMediaType, "Specify numCredits")
			return
		}
		if numCredits < 0 {
			log.Error(err)
			c.String(http.StatusBadRequest, "Can't specify negative credits!")
		}
		mime := getMime(data)
		data.Seek(0, 0)

		if !r.app.ldap.DecrementCredits(plug.Owner, numCredits) {
			c.String(http.StatusPaymentRequired, "Get More Credits!")
			return
		}

		plug.ViewsRemaining = numCredits * 1000

		plug.S3ID = time.Now().Format("2006/01/02/150405") + "-" + plug.Owner + "-" + file.Filename
		r.app.s3.AddFile(plug, data, mime)

		r.app.db.MakePlug(plug)
	} else {
		log.Error("invalid file dimensions")
		c.String(http.StatusBadRequest, "Please upload a 728x200 pixel image!")
		return
	}
	r.app.db.AddLog(1, "uid: "+plug.Owner+"uploaded plug s3id"+plug.S3ID)
	c.HTML(http.StatusOK, "success.tmpl", gin.H{
		"plug_s3url": r.app.s3.PresignPlug(plug).String(),
	})
	log.WithFields(log.Fields{
		"uid":       claims.UserInfo.Username,
		"plug_id":   plug.ID,
		"plug_s3id": plug.S3ID,
	}).Info("Uploaded new Plug!")
}

func (r PlugRoutes) upload_view(c *gin.Context) {
	c.HTML(http.StatusOK, "upload.tmpl", gin.H{})
}

func (r PlugRoutes) get_pending_plugs(c *gin.Context) {
	claims, ok := c.Value(csh_auth.AuthKey).(csh_auth.CSHClaims)
	if !ok {
		log.Fatal("error finding claims")
		return
	}

	if !r.app.ldap.CheckIfAdmin(claims.UserInfo.Username) {
		c.Redirect(http.StatusFound, "/")
		return
	}
	plugs := r.app.db.GetPendingPlugs()
	var out_plugs []Plug

	for _, plug := range plugs {
		new := plug
		new.PresignedURL = r.app.s3.PresignPlug(plug).String()
		out_plugs = append(out_plugs, new)
	}
	c.HTML(http.StatusOK, "view_plugs.tmpl", gin.H{
		"plugs": out_plugs,
	})
}

func (r PlugRoutes) plug_approval(c *gin.Context) {
	claims, ok := c.Value(csh_auth.AuthKey).(csh_auth.CSHClaims)
	if !ok {
		log.Fatal("error finding claims")
		return
	}

	if !r.app.ldap.CheckIfAdmin(claims.UserInfo.Username) {
		c.Redirect(http.StatusFound, "/")
		return
	}

	var plugList PlugList
	c.Bind(&plugList)

	log.WithFields(log.Fields{
		"uid":            claims.UserInfo.Username,
		"plugs_approved": strings.Join(plugList.Data, ","),
	}).Info("Changed Approved Plug List")

	r.app.db.AddLog(1, "uid: "+claims.UserInfo.Username+"approved: "+strings.Join(plugList.Data, ","))

	r.app.db.SetPendingPlugs(plugList.Data)
	c.Redirect(http.StatusFound, "/admin")
}

func (r PlugRoutes) plug_deletion(c *gin.Context) {
	claims, ok := c.Value(csh_auth.AuthKey).(csh_auth.CSHClaims)
	if !ok {
		log.Fatal("error finding claims")
		return
	}

	if !r.app.ldap.CheckIfAdmin(claims.UserInfo.Username) {
		c.Redirect(http.StatusFound, "/")
		return
	}

	id, err := strconv.Atoi(c.Param("id"))

	if err != nil {
		log.Error(err)
	}

	r.app.db.DeletePlug(r.app.db.GetPlugById(id))

	c.Redirect(http.StatusFound, "/admin")
}

func getMime(data io.Reader) string {
	buffer := make([]byte, 512)
	n, err := data.Read(buffer)
	if err != nil {
		log.Error(err)
	}
	return http.DetectContentType(buffer[:n])
}
