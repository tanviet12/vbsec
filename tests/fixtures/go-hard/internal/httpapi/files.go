package httpapi

import (
	"net/http"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/gin-gonic/gin"
)

const filesDir = "/srv/wallet/files"
const avatarDir = "/srv/wallet/avatars"

var avatarName = regexp.MustCompile(`^[a-f0-9]{32}\.png$`)

func (a *API) DownloadFile(c *gin.Context) {
	p := filepath.Join(filesDir, c.Param("name"))
	if !strings.HasPrefix(p, filesDir) {
		c.AbortWithStatus(http.StatusForbidden)
		return
	}
	c.File(p)
}

func (a *API) Avatar(c *gin.Context) {
	name := filepath.Base(c.Param("name"))
	if !avatarName.MatchString(name) {
		c.AbortWithStatus(http.StatusNotFound)
		return
	}
	c.File(filepath.Join(avatarDir, name))
}
