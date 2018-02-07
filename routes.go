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
	"strings"
	"time"
)

func protectedProfile(c *gin.Context) {
	claims, ok := c.Value(csh_auth.AuthKey).(csh_auth.CSHClaims)
	if !ok {
		log.Fatal("error finding claims")
		return
	}
	c.String(http.StatusOK, "uid %s email %s name %s uuid %s", claims.UserInfo.Username, claims.UserInfo.Email, claims.UserInfo.FullName, claims.UserInfo.Subject)
}

func index(c *gin.Context) {
	c.Redirect(http.StatusFound, "/upload")
}

func action(c *gin.Context) {
	plug := GetPlug()
	url := S3PresignPlug(plug)

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
	AddLog(13, c.GetHeader("Referer"))
	c.Redirect(http.StatusFound, url.String())
}

func upload(c *gin.Context) {
	plug := Plug{}

	claims, ok := c.Value(csh_auth.AuthKey).(csh_auth.CSHClaims)
	if !ok {
		log.Fatal("error finding claims")
		return
	}

	plug.Owner = claims.UserInfo.Username
	plug.ViewsRemaining = 1000

	file, err := c.FormFile("file")
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
		mime := getMime(data)
		data.Seek(0, 0)

		if !DecrementCredits(plug.Owner, 1) {
			c.String(http.StatusPaymentRequired, "Get More Credits!")
			return
		}

		plug.S3ID = time.Now().Format("2006/01/02/150405") + "-" + plug.Owner + "-" + file.Filename
		S3AddFile(plug, data, mime)

		MakePlug(plug)
	} else {
		log.Error("invalid file dimensions")
		c.String(http.StatusBadRequest, "Please upload a 728x200 pixel image!")
		return
	}
	AddLog(1, "uid: "+plug.Owner+"uploaded plug s3id"+plug.S3ID)
	c.Data(http.StatusOK, "text/html", []byte(`
<html>

<head>
    <meta http-equiv="content-type" content="text/html; charset=UTF-8">
    <link rel="stylesheet" href="https://s3.csh.rit.edu/csh-material-bootstrap/4.0.0-beta.3/dist/csh-material-bootstrap.min.css" media="screen">
    <style>
        html {
            position: relative;
            min-height: 100%;
        }

        body {
            margin-bottom: 60px;
            /* Margin bottom by footer height */
        }

        .footer {
            position: absolute;
            bottom: 0;
            width: 100%;
            height: 60px;
            /* Set the fixed height of the footer here */
            line-height: 60px;
            /* Vertically center the text there */
            background-color: #f5f5f5;
        }
    </style>
</head>

<body>
    <form action="/admin" method="POST">

        <nav class="navbar navbar-expand-lg navbar-dark bg-dark">
            <div class="container">
                <a class="navbar-brand" href="/upload">Plug</a>
                <ul class="navbar-nav mr-auto">
                    <li class="nav-item active">
                        <a class="nav-link" href="/upload">Upload</a>
                    </li>
                    <li class="nav-item">
                        <a class="nav-link" href="/admin">Admin <span class="sr-only">(current)</span></a>
                    </li>
                </ul>
            </div>
        </nav>
        <div class="container">
            <div class="row justify-content-center">
                <div class="col-lg-7">
                    <div class="card mb-3">
                    </div>
                </div>
            </div>

            <div class="row justify-content-center">
                <div class="col-lg-7">
                    <div class="card mb-3">
                        <h3 class="card-header">Success!</h3>
                        <img style="width: 100%; display: block;" src="`+S3PresignPlug(plug).String()+`" alt="Card image">

                        <div class="card-footer text-muted">
                            This is how your Plug will appear on CSH sites. (This does not count towards the views for your Plug.)<br><br>Your Plug must be approved before it will appear for viewing. Any member of the following groups (drink, eboard, rtp) can do so via the admin page.
                        </div>
                    </div>
                </div>
            </div>
        </div>

    </form>

    <footer class="footer">
        <div class="container">
            <span class="text-muted">CSH Plug on <a href="https://github.com/liam-middlebrook/csh-plug">GitHub</a></span>
        </div>
    </footer>
</body>

</html>
	`))
	log.WithFields(log.Fields{
		"uid":       claims.UserInfo.Username,
		"plug_id":   plug.ID,
		"plug_s3id": plug.S3ID,
	}).Info("Uploaded new Plug!")
}

func upload_view(c *gin.Context) {
	c.Data(http.StatusOK, "text/html", []byte(`
	<html>

<head>
    <meta http-equiv="content-type" content="text/html; charset=windows-1252">
    <link rel="stylesheet" href="https://s3.csh.rit.edu/csh-material-bootstrap/4.0.0-beta.3/dist/csh-material-bootstrap.min.css" media="screen">
    <style>
        html {
            position: relative;
            min-height: 100%;
        }

        body {
            margin-bottom: 60px;
            /* Margin bottom by footer height */
        }

        .footer {
            position: absolute;
            bottom: 0;
            width: 100%;
            height: 60px;
            /* Set the fixed height of the footer here */
            line-height: 60px;
            /* Vertically center the text there */
            background-color: #f5f5f5;
        }
    </style>
</head>

<body>
    <nav class="navbar navbar-expand-lg navbar-dark bg-dark">
        <div class="container">
            <a class="navbar-brand" href="/upload">Plug</a>
            <ul class="navbar-nav mr-auto">
                <li class="nav-item active">
                    <a class="nav-link" href="/upload">Upload</a>
                </li>
                <li class="nav-item">
                    <a class="nav-link" href="/admin">Admin <span class="sr-only">(current)</span></a>
                </li>
            </ul>
        </div>
    </nav>

    <div class="jumbotron">
        <div class="row justify-content-center">
            <div class="col-lg-7">
                <div class="jumbotron" style="max-width: 800px">
                    <h1 class="display-3">Upload a Plug!</h1>
                    <p class="lead">You will lose 1 drink credit in exchange for a 1000 view-limit of your plug.<br> Plugs must be 728x200 pixels and in PNG, or JPG format</p>
                    <hr class="my-4">

                    <div class="form-group">
                        <input class="form-control-file" id="exampleInputFile" aria-describedby="fileHelp" type="file">
                        <small id="fileHelp" class="form-text text-muted">Your Plug must be approved before it will appear for viewing. Any member of the following groups (drink, eboard, rtp) can do so via the admin page.</small>
                    </div>
                    <div class="float-right">
                        <form action="/upload" method="post" enctype="multipart/form-data">
                            <input class="btn btn-primary btn-lg" href="/upload" role="button" value="Upload" name="submit" type="submit"> </form>
                    </div>
                </div>
            </div>
        </div>
    </div>

    <div class="modal" id="agreementModal">
        <div class="modal-dialog" role="document">
            <div class="modal-content">
                <div class="modal-header">
                    <h5 class="modal-title">User Agreement</h5>
                </div>
                <div class="modal-body">
                    <p>The CSH Code Of Conduct Section 8 prohibits the sending of content that may harass others.</p>
                    <p>Please review the <a href="http://latex.aslushnikov.com/compile?url=https%3A%2F%2Fraw.githubusercontent.com%2FComputerScienceHouse%2FCodeOfConduct%2Fmaster%2Fcsh-coc.tex" target="_blank">CSH Code Of Conduct</a> before uploading content to Plug.</p>
                </div>
                <div class="modal-footer">
                    <button type="button" class="btn btn-primary" data-dismiss="modal">Agree</button>
                </div>
            </div>
        </div>
    </div>




    <div>

    </div>

    <footer class="footer">
        <div class="container">
            <span class="text-muted">CSH Plug on <a href="https://github.com/liam-middlebrook/csh-plug">GitHub</a></span>
        </div>
    </footer>


    <script src="https://code.jquery.com/jquery-3.2.1.slim.min.js" integrity="sha384-KJ3o2DKtIkvYIK3UENzmM7KCkRr/rE9/Qpg6aAZGJwFDMVNA/GpGFF93hXpG5KkN" crossorigin="anonymous"></script>
    <script src="https://cdnjs.cloudflare.com/ajax/libs/popper.js/1.12.9/umd/popper.min.js" integrity="sha384-ApNbgh9B+Y1QKtv3Rn7W3mgPxhU9K/ScQsAP7hUibX39j7fakFPskvXusvfa0b4Q" crossorigin="anonymous"></script>
    <script src="https://maxcdn.bootstrapcdn.com/bootstrap/4.0.0-beta.3/js/bootstrap.min.js" integrity="sha384-a5N7Y/aK3qNeh15eJKGWxsqtnX/wWdSZSKp+81YjTmS15nvnvxKHuzaWwXHDli+4" crossorigin="anonymous"></script>
    <script>
        <!--        $('#agreementModal').modal('show') -->
    </script>

</body>

</html>
	`))
}

func get_pending_plugs(c *gin.Context) {
	claims, ok := c.Value(csh_auth.AuthKey).(csh_auth.CSHClaims)
	if !ok {
		log.Fatal("error finding claims")
		return
	}

	if !CheckIfAdmin(claims.UserInfo.Username) {
		c.Redirect(http.StatusFound, "/")
		return
	}
	plugs := GetPendingPlugs()
	var out_plugs []Plug

	for _, plug := range plugs {
		new := plug
		new.PresignedURL = S3PresignPlug(plug).String()
		out_plugs = append(out_plugs, new)
	}
	c.HTML(http.StatusOK, "view_plugs.tmpl", gin.H{
		"plugs": out_plugs,
	})
}

func plug_approval(c *gin.Context) {
	claims, ok := c.Value(csh_auth.AuthKey).(csh_auth.CSHClaims)
	if !ok {
		log.Fatal("error finding claims")
		return
	}

	if !CheckIfAdmin(claims.UserInfo.Username) {
		c.Redirect(http.StatusFound, "/")
		return
	}

	var plugList PlugList
	c.Bind(&plugList)

	log.WithFields(log.Fields{
		"uid":            claims.UserInfo.Username,
		"plugs_approved": strings.Join(plugList.Data, ","),
	}).Info("Changed Approved Plug List")

	AddLog(1, "uid: "+claims.UserInfo.Username+"approved: "+strings.Join(plugList.Data, ","))

	SetPendingPlugs(plugList.Data)
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
