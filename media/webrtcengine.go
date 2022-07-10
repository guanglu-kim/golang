package media

import (
	"center/util"
	"github.com/pion/webrtc/v2"
	"io"

	"github.com/pion/rtcp"
	"github.com/pion/rtp"
	"github.com/pion/rtp/codecs"
	"github.com/pion/webrtc/v2/pkg/media/samplebuilder"
)

// 默认配置 ICEServers 应该是没用 实际用的时候清空了
var defaultPeerCfg = webrtc.Configuration{
	ICEServers: []webrtc.ICEServer{
		{
			URLs: []string{"stun:stun.stunprotocol.org:3478"},
		},
	},
}

const (
	//一个媒体传送单元是1400 分成7个包 每侦所需要的RTP包的个数
	averageRtpPacketsPerFrame = 7
)

type WebRTCEngine struct {
	cfg webrtc.Configuration // 保存一套配置 用于传参

	mediaEngine webrtc.MediaEngine // mediaEngin 用于构造 api

	api *webrtc.API // api 是最后调用 pion 的入口
}

func NewWebRTCEngine() *WebRTCEngine {
	urls := []string{} //conf.SFU.Ices//[]string{"stun:stun.stunprotocol.org:3478"};//conf.SFU.Ices

	// sdp Session Description Protoco
	w := &WebRTCEngine{
		mediaEngine: webrtc.MediaEngine{},
		cfg: webrtc.Configuration{
			SDPSemantics: webrtc.SDPSemanticsUnifiedPlanWithFallback, // SDP 风格 媒体协商的标准
			ICEServers: []webrtc.ICEServer{
				{
					URLs: urls,
				},
			},
		},
	}

	// 注册编码译码器  VP8 是谷歌提供的视频编解码器
	w.mediaEngine.RegisterCodec(webrtc.NewRTPVP8Codec(webrtc.DefaultPayloadTypeVP8, 90000))
	//Opus 有损声音编码格式
	w.mediaEngine.RegisterCodec(webrtc.NewRTPOpusCodec(webrtc.DefaultPayloadTypeOpus, 48000))

	// 主要通过api 对象操作
	w.api = webrtc.NewAPI(webrtc.WithMediaEngine(w.mediaEngine))
	return w
}

// 发送者 PC-B PC-C 发送数据
func (s WebRTCEngine) CreateSender(offer webrtc.SessionDescription, pc **webrtc.PeerConnection, addVideoTrack, addAudioTrack **webrtc.Track, stop chan int) (answer webrtc.SessionDescription, err error) {

	// 创建一个 pc
	*pc, err = s.api.NewPeerConnection(s.cfg)
	util.Infof("WebRTCEngine.CreateSender pc=%p", *pc)
	if err != nil {
		return webrtc.SessionDescription{}, err
	}

	// 加入 音视频流 这里实现的给数据
	if *addVideoTrack != nil && *addAudioTrack != nil {
		(*pc).AddTrack(*addVideoTrack)
		(*pc).AddTrack(*addAudioTrack)
		err = (*pc).SetRemoteDescription(offer)
		if err != nil {
			return webrtc.SessionDescription{}, err
		}
	}

	//创建应答Answer
	answer, err = (*pc).CreateAnswer(nil)
	//设置本地SDP
	err = (*pc).SetLocalDescription(answer)
	util.Infof("WebRTCEngine.CreateReceiver ok")
	return answer, err

}

//创建 PC-A
func (s WebRTCEngine) CreateReceiver(offer webrtc.SessionDescription, pc **webrtc.PeerConnection, videoTrack, audioTrack **webrtc.Track, stop chan int, pli chan int) (answer webrtc.SessionDescription, err error) {

	// 先创建一个 PeerConnection
	*pc, err = s.api.NewPeerConnection(s.cfg)
	util.Infof("WebRTCEngine.CreateReceiver pc=%p", *pc)
	if err != nil {
		return webrtc.SessionDescription{}, err
	}

	// 添加视频接收器
	_, err = (*pc).AddTransceiver(webrtc.RTPCodecTypeVideo)
	if err != nil {
		return webrtc.SessionDescription{}, err
	}

	// 添加音频接收器
	_, err = (*pc).AddTransceiver(webrtc.RTPCodecTypeAudio)
	if err != nil {
		return webrtc.SessionDescription{}, err
	}

	//监听OnTrack事件  remoteTrack 就是 A 端推上来的流
	(*pc).OnTrack(func(remoteTrack *webrtc.Track, receiver *webrtc.RTPReceiver) {

		//视频处理
		if remoteTrack.PayloadType() == webrtc.DefaultPayloadTypeVP8 ||
			remoteTrack.PayloadType() == webrtc.DefaultPayloadTypeVP9 ||
			remoteTrack.PayloadType() == webrtc.DefaultPayloadTypeH264 {
			//根据remoteTrack创建一个VideoTrack赋值给videoTrack
			*videoTrack, err = (*pc).NewTrack(remoteTrack.PayloadType(), remoteTrack.SSRC(), "video", remoteTrack.Label())

			go func() {
				for {
					select {
					case <-pli:
						//PictureLossIndication 关键帧丢包重传,参考rfc4585  SSRC同步源标识符
						(*pc).WriteRTCP([]rtcp.Packet{&rtcp.PictureLossIndication{MediaSSRC: remoteTrack.SSRC()}})
					case <-stop:
						return
					}
				}
			}()
			//rtp解包 解的就是 videoTrack
			var pkt rtp.Depacketizer
			//判断视频编码
			if remoteTrack.PayloadType() == webrtc.DefaultPayloadTypeVP8 {
				//使用VP8编码
				pkt = &codecs.VP8Packet{}
			} else if remoteTrack.PayloadType() == webrtc.DefaultPayloadTypeVP9 {
				pkt = &codecs.VP8Packet{}
				util.Errorf("TODO codecs.VP9Packet")
			} else if remoteTrack.PayloadType() == webrtc.DefaultPayloadTypeH264 {
				pkt = &codecs.H264Packet{}
				util.Errorf("TODO codecs.H264Packet")
			}

			// 用于存放读取的视轨
			builder := samplebuilder.New(averageRtpPacketsPerFrame*5, pkt)
			for {
				select {

				case <-stop:
					return
				default:
					//读取RTP包  就是视轨
					rtp, err := remoteTrack.ReadRTP()
					if err != nil {
						if err == io.EOF {
							return
						}
						util.Errorf(err.Error())
					}

					//将RTP包放入builder对象里
					builder.Push(rtp)
					//迭代数据
					for s := builder.Pop(); s != nil; s = builder.Pop() {
						//向videoTrack里写入数据 这一句实现的将 PC-A 的流 推给 PC-B PC-C
						if err := (*videoTrack).WriteSample(*s); err != nil && err != io.ErrClosedPipe {
							util.Errorf(err.Error())
						}
					}
				}
			}
			//音频处理
		} else {
			*audioTrack, err = (*pc).NewTrack(remoteTrack.PayloadType(), remoteTrack.SSRC(), "audio", remoteTrack.Label())

			rtpBuf := make([]byte, 1400)
			for {
				select {
				case <-stop:
					return
				default:
					//读取音频数据
					i, err := remoteTrack.Read(rtpBuf)
					if err == nil {
						//将音频数据写入audioTrack
						(*audioTrack).Write(rtpBuf[:i])
					} else {
						util.Infof(err.Error())
					}
				}
			}
		}
	})

	//设置远端SDP
	err = (*pc).SetRemoteDescription(offer)
	if err != nil {
		return webrtc.SessionDescription{}, err
	}

	//创建应答Answer
	answer, err = (*pc).CreateAnswer(nil)
	//设置本地SDP
	err = (*pc).SetLocalDescription(answer)
	util.Infof("WebRTCEngine.CreateReceiver ok")
	return answer, err

}
