package media

import (
	"center/util"
	"time"

	"github.com/pion/webrtc/v2"
)

var (
	webrtcEngine *WebRTCEngine
)

// 初始化一个 WebRTCEngine 便于后续调用
func init() {
	webrtcEngine = NewWebRTCEngine()
}

// 永远是应答方
type WebRTCPeer struct {
	ID         string
	PC         *webrtc.PeerConnection // SFU端 应答的PeerConnection
	VideoTrack *webrtc.Track          // 视轨
	AudioTrack *webrtc.Track          // 音轨
	stop       chan int               // 关闭通道
	pli        chan int               // 关键帧 丢包重传 通道
}

// 创建对象 建立 stop 通道 建立 pli 通道 ID 的生成规则是 UserID 这个规则是最大的疑问  接收者不是多个么？
func NewWebRTCPeer(id string) *WebRTCPeer {
	return &WebRTCPeer{
		ID:   id,
		stop: make(chan int),
		pli:  make(chan int),
	}
}

// 关闭两个通道 stop pli
func (p *WebRTCPeer) Stop() {
	close(p.stop)
	close(p.pli)
}

// 应答 发布者  A->SFU->B  A 就是发布者
func (p *WebRTCPeer) AnswerSender(offer webrtc.SessionDescription) (answer webrtc.SessionDescription, err error) {

	//创建接收
	return webrtcEngine.CreateReceiver(offer, &p.PC, &p.VideoTrack, &p.AudioTrack, p.stop, p.pli)
}

//应答 接收方   A->SFU->B B 就是接收者  响应接收方的时候 要传递 audio track / video track
func (p *WebRTCPeer) AnswerReceiver(offer webrtc.SessionDescription, addVideoTrack **webrtc.Track, addAudioTrack **webrtc.Track) (answer webrtc.SessionDescription, err error) {
	util.Infof("WebRTCPeer.AnswerReceiver")

	//创建发送
	return webrtcEngine.CreateSender(offer, &p.PC, addVideoTrack, addAudioTrack, p.stop)
}

// 如果不加这块 接收方会接收不到数据 调用的地方不在这里
func (p *WebRTCPeer) SendPLI() {
	go func() {
		defer func() {
			//恢复
			if r := recover(); r != nil {
				util.Errorf("%v", r)
				return
			}
		}()
		ticker := time.NewTicker(time.Second)
		i := 0
		for {
			select {
			case <-ticker.C:
				p.pli <- 1
				if i > 3 {
					return
				}
				i++
			case <-p.stop:
				return
			}
		}
	}()
}
