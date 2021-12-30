package v1

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/iyear/pure-live/global"
	"github.com/iyear/pure-live/pkg/ecode"
	"github.com/iyear/pure-live/pkg/format"
	"github.com/iyear/pure-live/service/svc_live"
	"go.uber.org/zap"
)

func GetPlayURL(c *gin.Context) {
	req := struct {
		Plat string `form:"plat" binding:"required,max=15" json:"plat"`
		Room string `form:"room" binding:"required" json:"room"`
	}{}
	if err := c.ShouldBind(&req); err != nil {
		format.HTTP(c, ecode.InvalidParams, nil, nil)
		return
	}
	url, err := svc_live.GetPlayURL(req.Plat, req.Room)
	if err != nil {
		format.HTTP(c, ecode.ErrorGetPlayURL, err, nil)
		zap.S().Warnw("failed to get play url", "error", err, "req", req)
		return
	}
	format.HTTP(c, ecode.Success, nil, url)
}
func GetRoomInfo(c *gin.Context) {
	req := struct {
		Plat string `form:"plat" binding:"required,max=15" json:"plat"`
		Room string `form:"room" binding:"required" json:"room"`
	}{}
	if err := c.ShouldBind(&req); err != nil {
		format.HTTP(c, ecode.InvalidParams, nil, nil)
		return
	}
	info, err := svc_live.GetRoomInfo(req.Plat, req.Room)
	if err != nil {
		format.HTTP(c, ecode.ErrorGetRoomInfo, err, nil)
		zap.S().Warnw("failed to get room info", "error", err, "req", req)
		return
	}
	format.HTTP(c, ecode.Success, nil, info)
}
func SendDanmaku(c *gin.Context) {
	req := struct {
		ID      string `form:"id" binding:"required,uuid"` // 服务端分发的uuid
		Content string `form:"content" binding:"required" json:"content"`
		Type    int    `form:"type" binding:"gte=0,lte=2" json:"type"` // 1:顶部 0:滚动 2:底部
		Color   int64  `form:"color" binding:"required" json:"color"`
	}{}
	if err := c.ShouldBind(&req); err != nil {
		format.HTTP(c, ecode.InvalidParams, nil, nil)
		return
	}

	conn, err := global.GetConn(req.ID)
	if err != nil {
		format.HTTP(c, ecode.UnknownError, fmt.Errorf("can not get global conn"), nil)
		return
	}

	if err = svc_live.SendDanmaku(conn.Client, conn.Room, req.Content, req.Type, req.Color); err != nil {
		zap.S().Warnw("failed to send danmaku", "error", err, "req", req)
		format.HTTP(c, ecode.ErrorSendDanmaku, err, nil)
		return
	}
	format.HTTP(c, ecode.Success, nil, nil)
}
