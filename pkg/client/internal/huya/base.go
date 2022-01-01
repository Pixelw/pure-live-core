package huya

import (
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"github.com/TarsCloud/TarsGo/tars/protocol/codec"
	"github.com/gorilla/websocket"
	"github.com/iyear/pure-live-core/model"
	"github.com/iyear/pure-live-core/pkg/client/internal/abstract"
	"github.com/iyear/pure-live-core/pkg/client/internal/huya/internal/tars/danmaku"
	"github.com/iyear/pure-live-core/pkg/client/internal/huya/internal/tars/online"
	"github.com/iyear/pure-live-core/pkg/client/internal/huya/internal/tars/push_msg"
	"github.com/iyear/pure-live-core/pkg/client/internal/huya/internal/tars/ws_cmd"
	"github.com/iyear/pure-live-core/pkg/conf"
	"github.com/iyear/pure-live-core/pkg/util"
	"net/url"
	"strings"
)

const hb = "00031d0000690000006910032c3c4c56086f6e6c696e657569660f4f6e557365724865617274426561747d00003c0800010604745265711d00002f0a0a0c1600260036076164725f77617046000b1203aef00f2203aef00f3c426d5202605c60017c82000bb01f9cac0b8c980ca80c20"

type Huya struct {
	*abstract.Client
}
type H map[string]interface{}

// NewHuya
func NewHuya() (model.Client, error) {
	return &Huya{}, nil
}

// Plat
func (h *Huya) Plat() string {
	return conf.PlatHuya
}

// GetPlayURL
func (h *Huya) GetPlayURL(room string, qn int) (*model.PlayURL, error) {
	liveLine := ""
	json, err := getRoomInfo(room)
	if err != nil {
		return nil, err
	}
	if liveLine = json.Get("roomProfile.liveLineUrl").String(); liveLine == "" {
		return nil, fmt.Errorf("no broadcast or live room")
	}
	b64, err := base64.StdEncoding.DecodeString(liveLine)
	if err != nil {
		return nil, err
	}
	link := strings.ReplaceAll(string(b64), "hls", "flv")
	link = strings.ReplaceAll(link, "m3u8", "flv")

	u, err := url.Parse(fmt.Sprintf("https:%s", link))
	if err != nil {
		return nil, err
	}
	query := u.Query()
	// 设置最高清晰度
	query.Set("ratio", "0")
	u.RawQuery = query.Encode()

	return &model.PlayURL{
		Qn:     qn,
		Desc:   util.Qn2Desc(qn),
		Origin: u.String(),
		CORS:   false,
		Type:   conf.StreamFlv,
	}, err
}

// GetRoomInfo
func (h *Huya) GetRoomInfo(room string) (*model.RoomInfo, error) {
	j, err := getRoomInfo(room)
	if err != nil {
		return nil, err
	}
	return &model.RoomInfo{
		Status: util.IF(j.Get("roomInfo.eLiveStatus").Int() == 2, 1, 0).(int),
		Room:   room,
		Upper:  j.Get("roomInfo.tProfileInfo.sNick").String(),
		Link:   fmt.Sprintf("https://www.huya.com/%s", room),
		Title:  j.Get("roomInfo.tLiveInfo.sIntroduction").String(),
	}, nil
}

// Host
func (h *Huya) Host() string {
	return "wss://cdnws.api.huya.com/"
}

// Enter
func (h *Huya) Enter(room string) (int, [][]byte, error) {
	j, err := getRoomInfo(room)
	if err != nil {
		return -1, nil, err
	}
	lYyid := j.Get("roomInfo.tLiveInfo.lYyid").Int()
	lChannelId := j.Get("roomInfo.tLiveInfo.tLiveStreamInfo.vStreamInfo.value.0.lChannelId").Int()
	lSubChannelId := j.Get("roomInfo.tLiveInfo.tLiveStreamInfo.vStreamInfo.value.0.lSubChannelId").Int()
	// fmt.Println(lYyid, lChannelId, lSubChannelId)
	return websocket.BinaryMessage, [][]byte{getEnterMsg(lYyid, lChannelId, lSubChannelId)}, nil
}

// HeartBeat
func (h *Huya) HeartBeat() (int, []byte, error) {
	msg, err := hex.DecodeString(hb)
	return websocket.BinaryMessage, msg, err
}

// Handle
func (h *Huya) Handle(tp int, msg []byte) ([]model.Msg, bool, error) {
	if tp != websocket.BinaryMessage {
		return nil, false, nil
	}
	cmd := ws_cmd.WebSocketCommand{}
	if err := cmd.ReadFrom(codec.NewReader(msg)); err != nil {
		return nil, false, err
	}
	switch cmd.ICmdType {
	case ewsCmdS2CMsgPushReq:
		return h.handleMsgPushReq(codec.FromInt8(cmd.VData))
	}
	return nil, false, nil
}

func (h *Huya) handleMsgPushReq(b []byte) ([]model.Msg, bool, error) {
	r := make([]model.Msg, 1)
	msg := push_msg.WSPushMessage{}
	if err := msg.ReadFrom(codec.NewReader(b)); err != nil {
		return nil, false, err
	}
	// fmt.Println(msg.EPushType, msg.IUri)
	switch msg.IUri {
	case 1400: // 弹幕
		d := danmaku.MessageNotice{}
		if err := d.ReadFrom(codec.NewReader(codec.FromInt8(msg.SMsg))); err != nil {
			return nil, false, err
		}
		// fmt.Println(d.SContent, d.TUserInfo.SNickName, d.IShowMode, d.TBulletFormat.IFontColor)
		r[0] = &model.MsgDanmaku{
			Content: d.SContent,
			Type:    0, // TODO 没找到虎牙弹幕mode的字段
			Color:   int64(util.IF(d.TBulletFormat.IFontColor == -1, int32(16777215), d.TBulletFormat.IFontColor).(int32)),
		}
		return r, true, nil

	case 8006: // 直播间热度
		on := online.AttendeeCountNotice{}
		if err := on.ReadFrom(codec.NewReader(codec.FromInt8(msg.SMsg))); err != nil {
			return nil, false, err
		}
		r[0] = &model.MsgHot{Hot: int64(on.IAttendeeCount)}
		return r, true, nil
	}
	return nil, false, nil
}

// SendDanmaku
func (h *Huya) SendDanmaku(room string, content string, tp int, color int64) error {
	_ = room
	_ = content
	_ = tp
	_ = color
	return fmt.Errorf("todo")
}

// Stop
func (h *Huya) Stop() {

}
