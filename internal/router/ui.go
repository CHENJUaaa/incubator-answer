/*
 * Licensed to the Apache Software Foundation (ASF) under one
 * or more contributor license agreements.  See the NOTICE file
 * distributed with this work for additional information
 * regarding copyright ownership.  The ASF licenses this file
 * to you under the Apache License, Version 2.0 (the
 * "License"); you may not use this file except in compliance
 * with the License.  You may obtain a copy of the License at
 *
 *   http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing,
 * software distributed under the License is distributed on an
 * "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
 * KIND, either express or implied.  See the License for the
 * specific language governing permissions and limitations
 * under the License.
 */

package router

import (
	"embed"
	"fmt"
	"io/fs"
	"net/http"
	"os"

	"github.com/answerdev/answer/internal/controller"
	"github.com/answerdev/answer/internal/service/siteinfo_common"
	"github.com/answerdev/answer/pkg/htmltext"
	"github.com/answerdev/answer/ui"
	"github.com/gin-gonic/gin"
	"github.com/segmentfault/pacman/log"
)

const UIIndexFilePath = "build/index.html"
const UIRootFilePath = "build"
const UIStaticPath = "build/static"

// UIRouter is an interface that provides ui static file routers
type UIRouter struct {
	siteInfoController *controller.SiteInfoController
	siteInfoService    siteinfo_common.SiteInfoCommonService
}

// NewUIRouter creates a new UIRouter instance with the embed resources
func NewUIRouter(
	siteInfoController *controller.SiteInfoController,
	siteInfoService siteinfo_common.SiteInfoCommonService,
) *UIRouter {
	return &UIRouter{
		siteInfoController: siteInfoController,
		siteInfoService:    siteInfoService,
	}
}

// _resource is an interface that provides static file, it's a private interface
type _resource struct {
	fs embed.FS
}

// Open to implement the interface by http.FS required
func (r *_resource) Open(name string) (fs.File, error) {
	name = fmt.Sprintf(UIStaticPath+"/%s", name)
	log.Debugf("open static path %s", name)
	return r.fs.Open(name)
}

// Register a new static resource which generated by ui directory
func (a *UIRouter) Register(r *gin.Engine) {
	staticPath := os.Getenv("ANSWER_STATIC_PATH")

	// if ANSWER_STATIC_PATH is set and not empty, ignore embed resource
	if staticPath != "" {
		info, err := os.Stat(staticPath)

		if err != nil || !info.IsDir() {
			log.Error(err)
		} else {
			log.Debugf("registering static path %s", staticPath)

			r.LoadHTMLGlob(staticPath + "/*.html")
			r.Static("/static", staticPath+"/static")
			r.NoRoute(func(c *gin.Context) {
				c.HTML(http.StatusOK, "index.html", gin.H{})
			})

			// return immediately if the static path is set
			return
		}
	}

	// handle the static file by default ui static files
	r.StaticFS("/static", http.FS(&_resource{
		fs: ui.Build,
	}))

	// specify the not router for default routes and redirect
	r.NoRoute(func(c *gin.Context) {
		urlPath := c.Request.URL.Path
		filePath := ""
		switch urlPath {
		case "/favicon.ico":
			branding, err := a.siteInfoService.GetSiteBranding(c)
			if err != nil {
				log.Error(err)
			}
			if branding.Favicon != "" {
				c.String(http.StatusOK, htmltext.GetPicByUrl(branding.Favicon))
				return
			} else if branding.SquareIcon != "" {
				c.String(http.StatusOK, htmltext.GetPicByUrl(branding.SquareIcon))
				return
			} else {
				c.Header("content-type", "image/vnd.microsoft.icon")
				filePath = UIRootFilePath + urlPath

			}
		case "/manifest.json":
			// filePath = UIRootFilePath + urlPath
			a.siteInfoController.GetManifestJson(c)
			return
		case "/install":
			// if answer is running by run command user can not access install page.
			c.Redirect(http.StatusFound, "/")
			return
		default:
			filePath = UIIndexFilePath
			c.Header("content-type", "text/html;charset=utf-8")
			c.Header("X-Frame-Options", "DENY")
		}
		file, err := ui.Build.ReadFile(filePath)
		if err != nil {
			log.Error(err)
			c.Status(http.StatusNotFound)
			return
		}
		c.String(http.StatusOK, string(file))
	})
}
