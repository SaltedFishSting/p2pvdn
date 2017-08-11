// main
package main

import (
	"bufio"
	"bytes"
	"ebase"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"os/signal"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/prometheus/client_golang/prometheus/push"

	"gopkg.in/yaml.v2"
)

type GWConfig struct {
	Gw struct {
		Addr           string `yaml:"addr"`
		HttpListenPort int    `yaml:"httpListenPort"`
	}
	Output struct {
		Prometheus      bool   `yaml:"prometheus"`
		Telegraf        bool   `yaml:"telegraf"`
		TelegrafAddr    string `yaml:"telegrafAddr"`
		PushGateway     bool   `yaml:"pushGateway"`
		PushGatewayAddr string `yaml:"pushGatewayAddr"`
		JobName         string `yaml:"jobName"`
	}
	Rest struct {
		Test   bool   `yaml:"test"`
		Vdn    string `yaml:"vdn"`
		Period int    `yaml:"period"`
	}
	Logger struct {
		Filename   string `yaml:"filename"`
		MaxSize    int    `yaml:"maxSize"`
		MaxBackups int    `yaml:"maxBackups"`
		MaxAge     int    `yaml:"maxAge"`
	}
	Fileaddress struct {
		Server_sumary  string `yaml:"server_sumary"`
		User_statistic string `yaml:"user_statistic"`
		Call_statistic string `yaml:"call_statistic"`
		Host_info      string `yaml:"host_info"`
		Relay          string `yaml:"relay"`
		Bootstrap      string `yaml:"bootstrap"`
		Dht            string `yaml:"dht"`
		Sps            string `yaml:"sps"`
		Ps             string `yaml:"ps"`
		Callmgr        string `yaml:"callmgr"`
	}
}

var globeCfg *GWConfig

/*
statistic.serverSummary.action // 读取服务运行概要信息，供全局使用
  时间 节点ID 服务器类型 IP port 所属hostID *是否发布 *是否健康
statistic.userStatistic.action // 系统总体概况/用户在线统计
  时间 *在线用户数 *匿名用户数 *可激活用户数 *最近三分钟登录用户数 *最近三分钟登出用户数 分类终端在线用户数
statistic.callStatistic.action // 系统总体概况/通话统计
  时间 *当前通话并发总数 *当前视频通话总数 *当前音频通话总数 *最近3分钟通话量 *最近3分钟未接通数 *最近3分钟正常挂断数 *最近3分钟异常挂断数
TODO: statistic.acd.action     // 系统总体概况/ACD分配统计
TODO: statistic.im.action      // 系统总体概况/IM消息统计
statistic.host.action          // 设备运行情况/HOST服务
  时间 Host节点ID Host IP Host Port Host是否健康 *额定用户数 *在线用户数 *坐席在线个数 *匿名在线用户数 *工作线程未处理任务数 *最近3分钟登录次数 *最近3分钟登出次数 *最近3分钟登录用户数 *最近3分钟登出用户数 *最近3分钟查询被叫次数 *最近3分钟查询被叫本地命中次数 *最近3分钟查询被叫DHT查询次数 *最近3分钟转发消息次数 *最近3分钟转发消息CAHCE命中次数 *最近3分钟转发消息DHT查询次数 *最近3分钟转发消息本地命中次数 *最近3分钟发送坐席状态消息次数 *最近3分钟发送用户排队位置消息次数 *最近3分钟向APNS通道推送次数 *最近3分钟向静默通道推送次数 在线用户设备分布列表
statistic.relay.action         // 设备运行情况/RELAY服务
  时间 relay节点id relay IP relay Port *并发通话数 *接入|落地用户数 *最近3分钟短链保活消息数 *最近3分钟转发建路包数 *最近3分钟转发媒体包数 *最近3分钟无效消息数据 *最近3分钟通话建立次数 *最近3分钟通话结束次数 *最近3分钟平均媒体转发上行流量 *最近3分钟平均媒体转发下行流量
statistic.bootstrap.action     // 设备运行情况/BootStrap服务
  时间 Bootstrap节点ID Bootstrap IP Bootstrap Port *3分钟查询次数 *当前健康HOST数 *当前HOST总数 *路由表长度
statistic.DHT.action           // 设备运行情况/DHT服务
  时间 DHT节点id 所属Host节点ID DHT的KAD IP	 DHt的KAD Port *DHT连接状态 *DHT是否健康 *路由表个数 *在线信息用户数 *ANPS离线信息用户数 *有静默通道用户数 *Connect应用总数 Host列表 *最近3分钟内GetValue次数 *最近3分钟内SetValue次数 *GetValue响应速度列表
statistic.SPS.action           // 设备运行情况/SPS服务
  时间 SPS节点id 所属Host节点ID SPS IP SPS Port *通道[信令双通道+静默通道]连接数 *最近3分钟发送消息总数 *最近3分钟双通道发送给Host消息数 8最近3分钟双通道给客发送客户端消息数 *最近3分钟静默通道推送次数 *最近3分钟静默通道推送成功次数
statistic.ANPS.action          // 设备运行情况/ANPS服务
  时间 PS节点id 所属Host节点ID PS IP PS Port *与APNS连接成功通道数 *待推送的任务数 *最近3分钟推送总数 *最近3分钟推送成功次数 *最近3分钟推送失败次数
statistic.CM.action            // 设备运行情况/CM服务
  时间 CallMgr节点ID 所属Host节点ID CallMgr IP CallMgr port *当前通话数 *最近3分钟视频通数 *最近3分钟音频话数 *最近3分钟正常挂断通话数 *最近3分钟异常挂断通话数 *最近3分钟系统原因未接通数 *最近3分钟人为原因未接通数 *最近3分钟被叫不在线未接通数
TODO: statistic.rc.action      // 设备运行情况/RC服务
*/

var ( //statistic.serverSummary.action
	//*是否发布
	serverSummary_published = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "p2p",
			Subsystem: "serverSummary",
			Name:      "published",
			Help:      "is service published.",
		},
		[]string{
			"SvcType",
			"NodeID",
			"IP",
			"Port",
			"HostID",
		},
	)

	//*是否健康
	serverSummary_healthy = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "p2p",
			Subsystem: "serverSummary",
			Name:      "healthy",
			Help:      "is service healthy.",
		},
		[]string{
			"SvcType",
			"NodeID",
			"IP",
			"Port",
			"HostID",
		},
	)
)

func regServerSummary() {
	prometheus.MustRegister(serverSummary_published)
	prometheus.MustRegister(serverSummary_healthy)
}

func extractServerSummary(svrSum []string) error {
	//时间 节点ID 服务器类型 IP port 所属hostID *是否发布 *是否健康
	svcType := svrSum[2]
	nodeID := svrSum[1]
	ip := svrSum[3]
	port := svrSum[4]
	hostID := svrSum[5]
	published, _ := strconv.Atoi(svrSum[6])
	healthy, _ := strconv.Atoi(svrSum[7])
	if globeCfg.Output.Prometheus || globeCfg.Output.PushGateway {
		serverSummary_published.WithLabelValues(svcType, nodeID, ip, port, hostID).Set(float64(published))
		serverSummary_healthy.WithLabelValues(svcType, nodeID, ip, port, hostID).Set(float64(healthy))
	}

	if globeCfg.Output.Telegraf {
		mx := fmt.Sprintf("p2p_serverSummary,svcType=%s,nodeId=%s,addr=%s:%s,hostId=%s published=%d,healthy=%d\n",
			svcType,
			nodeID,
			ip, port,
			hostID,
			published,
			healthy)
		collectBuf.WriteString(mx)
	}
	return nil
}

var ( //statistic.userStatistic.action
	//*在线用户数

	userStatistic_online = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Namespace: "p2p",
			Subsystem: "userStatistic",
			Name:      "online",
			Help:      "sum of online user",
		},
	)
	//*匿名用户数
	userStatistic_anonym = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Namespace: "p2p",
			Subsystem: "userStatistic",
			Name:      "anonym",
			Help:      "sum of online anonym-user",
		},
	)

	//*可激活用户数
	userStatistic_activable = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Namespace: "p2p",
			Subsystem: "userStatistic",
			Name:      "activable",
			Help:      "sum of activable user",
		},
	)

	//*最近三分钟登录用户数
	userStatistic_login = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Namespace: "p2p",
			Subsystem: "userStatistic",
			Name:      "new_login",
			Help:      "increased login user.",
		},
	)

	//*最近三分钟登出用户数
	userStatistic_logout = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Namespace: "p2p",
			Subsystem: "userStatistic",
			Name:      "new_logout",
			Help:      "decreased login user.",
		},
	)
)

func regUserStatistic() {
	prometheus.MustRegister(userStatistic_online)
	prometheus.MustRegister(userStatistic_anonym)
	prometheus.MustRegister(userStatistic_activable)
	prometheus.MustRegister(userStatistic_login)
	prometheus.MustRegister(userStatistic_logout)
}

func extractUserStatistic(userS []string) error {
	//时间 *在线用户数 *匿名用户数 *可激活用户数 *最近三分钟登录用户数 *最近三分钟登出用户数 分类终端在线用户数
	onlineUserNum, _ := strconv.Atoi(userS[1])
	anonymUserNum, _ := strconv.Atoi(userS[2])
	activableUserNum, _ := strconv.Atoi(userS[3])
	loginUserNum, _ := strconv.Atoi(userS[4])
	logoutUserNum, _ := strconv.Atoi(userS[5])

	if globeCfg.Output.Prometheus || globeCfg.Output.PushGateway {
		userStatistic_online.Set(float64(onlineUserNum))
		userStatistic_anonym.Set(float64(anonymUserNum))
		userStatistic_activable.Set(float64(activableUserNum))
		userStatistic_login.Set(float64(loginUserNum))
		userStatistic_logout.Set(float64(logoutUserNum))
	}

	if globeCfg.Output.Telegraf {
		mx := fmt.Sprintf("p2p_userStatistic,host=all online=%d,anonym=%d,activable=%d,loginX=%d,logoutX=%d\n",
			onlineUserNum,
			anonymUserNum,
			activableUserNum,
			loginUserNum,
			logoutUserNum)
		collectBuf.WriteString(mx)
	}
	return nil
}

var ( //statistic.callStatistic.action
	//*当前通话并发总数
	callStatistic_onphone = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Namespace: "p2p",
			Subsystem: "callStatistic",
			Name:      "onphone",
			Help:      "sum of onphone user",
		},
	)
	//*当前视频通话总数
	callStatistic_onphoneV = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Namespace: "p2p",
			Subsystem: "callStatistic",
			Name:      "onphone_video",
			Help:      "sum of onphone video user",
		},
	)
	//*当前音频通话总数
	callStatistic_onphoneA = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Namespace: "p2p",
			Subsystem: "callStatistic",
			Name:      "onphone_audio",
			Help:      "sum of onphone audio user",
		},
	)
	//*最近3分钟通话量
	callStatistic_callTraffic = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Namespace: "p2p",
			Subsystem: "callStatistic",
			Name:      "new_traffic",
			Help:      "increased sum of call traffic",
		},
	)
	//*最近3分钟未接通数
	callStatistic_blockedCall = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Namespace: "p2p",
			Subsystem: "callStatistic",
			Name:      "new_blocked_call",
			Help:      "increased sum of blocked-call",
		},
	)
	//*最近3分钟正常挂断数
	callStatistic_releasedCall = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Namespace: "p2p",
			Subsystem: "callStatistic",
			Name:      "new_released_call",
			Help:      "increased sum of released-call",
		},
	)
	//*最近3分钟异常挂断数
	callStatistic_breakedCall = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Namespace: "p2p",
			Subsystem: "callStatistic",
			Name:      "new_broken_call",
			Help:      "increased sum of broken-call",
		},
	)
)

func regCallStatistic() {
	prometheus.MustRegister(callStatistic_onphone)
	prometheus.MustRegister(callStatistic_onphoneV)
	prometheus.MustRegister(callStatistic_onphoneA)
	prometheus.MustRegister(callStatistic_callTraffic)
	prometheus.MustRegister(callStatistic_blockedCall)
	prometheus.MustRegister(callStatistic_releasedCall)
	prometheus.MustRegister(callStatistic_breakedCall)

}

func extractCallStatistic(callS []string) error {
	//  时间 *当前通话并发总数 *当前视频通话总数 *当前音频通话总数 *最近3分钟通话量 *最近3分钟未接通数 *最近3分钟正常挂断数 *最近3分钟异常挂断数
	onphoneNum, _ := strconv.Atoi(callS[1])
	onphoneVNum, _ := strconv.Atoi(callS[2])
	onphoneANum, _ := strconv.Atoi(callS[3])
	callTrafficNum, _ := strconv.Atoi(callS[4])
	blockCallNum, _ := strconv.Atoi(callS[5])
	callReleasedNum, _ := strconv.Atoi(callS[6])
	callBreakedNum, _ := strconv.Atoi(callS[7])

	if globeCfg.Output.Prometheus || globeCfg.Output.PushGateway {
		callStatistic_onphone.Set(float64(onphoneNum))

		callStatistic_onphoneV.Set(float64(onphoneVNum))
		callStatistic_onphoneA.Set(float64(onphoneANum))
		callStatistic_callTraffic.Set(float64(callTrafficNum))
		callStatistic_blockedCall.Set(float64(blockCallNum))
		callStatistic_releasedCall.Set(float64(callReleasedNum))
		callStatistic_breakedCall.Set(float64(callBreakedNum))
	}

	if globeCfg.Output.Telegraf {
		mx := fmt.Sprintf("p2p_callStatistic,host=all onphone=%d,onvideo=%d,onaudio=%d,trafficX=%d,blockedX=%d,releasedX=%d,brokenX=%d\n",
			onphoneNum,
			onphoneVNum,
			onphoneANum,
			callTrafficNum,
			blockCallNum,
			callReleasedNum,
			callBreakedNum)
		collectBuf.WriteString(mx)
	}
	return nil
}

//TODO:statistic.acd.action
//TODO:statistic.im.action

var ( //statistic.host.action
	//*Host是否健康
	host_healthy = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "p2p",
			Subsystem: "host",
			Name:      "heathy",
			Help:      "heathy or not",
		},
		[]string{
			"HostID",
			"IP",
			"Port",
		},
	)
	//*额定用户数
	host_fixedUser = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "p2p",
			Subsystem: "host",
			Name:      "fixed_user",
			Help:      "sum of fixed user",
		},
		[]string{
			"HostID",
			"IP",
			"Port",
		},
	)
	//*在线用户数
	host_onlineUser = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "p2p",
			Subsystem: "host",
			Name:      "online_user",
			Help:      "sum of online user",
		},
		[]string{
			"HostID",
			"IP",
			"Port",
		},
	)
	//*坐席在线个数
	host_onlineSeat = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "p2p",
			Subsystem: "host",
			Name:      "online_seat",
			Help:      "sum of online seat",
		},
		[]string{
			"HostID",
			"IP",
			"Port",
		},
	)
	//*匿名在线用户数
	host_onlineAnonym = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "p2p",
			Subsystem: "host",
			Name:      "online_anonym",
			Help:      "sum of online anonym user",
		},
		[]string{
			"HostID",
			"IP",
			"Port",
		},
	)
	//*工作线程未处理任务数
	host_untreatedTask = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "p2p",
			Subsystem: "host",
			Name:      "untreated_task",
			Help:      "sum of untreated task",
		},
		[]string{
			"HostID",
			"IP",
			"Port",
		},
	)
	//*最近3分钟登录次数
	host_login = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "p2p",
			Subsystem: "host",
			Name:      "new_login",
			Help:      "increased sum of login",
		},
		[]string{
			"HostID",
			"IP",
			"Port",
		},
	)
	//*最近3分钟登出次数
	host_logout = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "p2p",
			Subsystem: "host",
			Name:      "new_logout",
			Help:      "increased sum of logout",
		},
		[]string{
			"HostID",
			"IP",
			"Port",
		},
	)
	//*最近3分钟登录用户数
	host_loginUser = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "p2p",
			Subsystem: "host",
			Name:      "new_login_user",
			Help:      "increased sum of login-user",
		},
		[]string{
			"HostID",
			"IP",
			"Port",
		},
	)
	//*最近3分钟登出用户数
	host_logoutUser = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "p2p",
			Subsystem: "host",
			Name:      "new_logout_user",
			Help:      "increased sum of logout-user",
		},
		[]string{
			"HostID",
			"IP",
			"Port",
		},
	)
	//*最近3分钟查询被叫次数
	host_queryCalled = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "p2p",
			Subsystem: "host",
			Name:      "new_query_called",
			Help:      "increased sum of querying called",
		},
		[]string{
			"HostID",
			"IP",
			"Port",
		},
	)
	//*最近3分钟查询被叫本地命中次数
	host_queryCalledSuc = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "p2p",
			Subsystem: "host",
			Name:      "new_query_called_success",
			Help:      "increased sum of success to query called",
		},
		[]string{
			"HostID",
			"IP",
			"Port",
		},
	)
	//*最近3分钟查询被叫DHT查询次数
	host_queryCalledDHT = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "p2p",
			Subsystem: "host",
			Name:      "new_query_called_DHT",
			Help:      "increased sum of query called DHT",
		},
		[]string{
			"HostID",
			"IP",
			"Port",
		},
	)
	//*最近3分钟转发消息次数
	host_relayMsg = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "p2p",
			Subsystem: "host",
			Name:      "new_relay_msg",
			Help:      "increased sum of relay message",
		},
		[]string{
			"HostID",
			"IP",
			"Port",
		},
	)
	//*最近3分钟转发消息CAHCE命中次数
	host_relayMsgCAHCESuc = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "p2p",
			Subsystem: "host",
			Name:      "new_relay_msg_CAHCE_success",
			Help:      "increased sum of succes to relay CAHCE message",
		},
		[]string{
			"HostID",
			"IP",
			"Port",
		},
	)
	//*最近3分钟转发消息DHT查询次数
	host_relayMsgQueryDHT = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "p2p",
			Subsystem: "host",
			Name:      "new_relay_msg_query_DHT",
			Help:      "increased sum of query DHT for relay message",
		},
		[]string{
			"HostID",
			"IP",
			"Port",
		},
	)
	//*最近3分钟转发消息本地命中次数
	host_relayMsgLocalSuc = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "p2p",
			Subsystem: "host",
			Name:      "new_relay_msg_local_success",
			Help:      "increased sum of succes to relay local message",
		},
		[]string{
			"HostID",
			"IP",
			"Port",
		},
	)
	//*最近3分钟发送坐席状态消息次数
	host_relaySeatMsg = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "p2p",
			Subsystem: "host",
			Name:      "new_relay_seat_msg",
			Help:      "increased sum of relay seat message",
		},
		[]string{
			"HostID",
			"IP",
			"Port",
		},
	)
	//*最近3分钟发送用户排队位置消息次数
	host_relayUserQueuePos = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "p2p",
			Subsystem: "host",
			Name:      "new_relay_user_pos_msg",
			Help:      "increased sum of relay user queue pos message",
		},
		[]string{
			"HostID",
			"IP",
			"Port",
		},
	)
	//*最近3分钟向APNS通道推送次数
	host_pushAPNS = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "p2p",
			Subsystem: "host",
			Name:      "new_push_APNS",
			Help:      "increased sum of push APNS",
		},
		[]string{
			"HostID",
			"IP",
			"Port",
		},
	)
	//*最近3分钟向静默通道推送次数
	host_pushSilent = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "p2p",
			Subsystem: "host",
			Name:      "new_push_silent",
			Help:      "increased sum of push silent",
		},
		[]string{
			"HostID",
			"IP",
			"Port",
		},
	)
)

func regHost() {
	prometheus.MustRegister(host_healthy)
	prometheus.MustRegister(host_fixedUser)
	prometheus.MustRegister(host_onlineUser)
	prometheus.MustRegister(host_onlineSeat)
	prometheus.MustRegister(host_onlineAnonym)
	prometheus.MustRegister(host_untreatedTask)
	prometheus.MustRegister(host_login)
	prometheus.MustRegister(host_logout)
	prometheus.MustRegister(host_loginUser)
	prometheus.MustRegister(host_logoutUser)
	prometheus.MustRegister(host_queryCalled)
	prometheus.MustRegister(host_queryCalledSuc)
	prometheus.MustRegister(host_queryCalledDHT)
	prometheus.MustRegister(host_relayMsg)
	prometheus.MustRegister(host_relayMsgCAHCESuc)
	prometheus.MustRegister(host_relayMsgQueryDHT)
	prometheus.MustRegister(host_relayMsgLocalSuc)
	prometheus.MustRegister(host_relaySeatMsg)
	prometheus.MustRegister(host_relayUserQueuePos)
	prometheus.MustRegister(host_pushAPNS)
	prometheus.MustRegister(host_pushSilent)
}

func extractHost(hs []string) error {
	//  时间 Host节点ID Host IP Host Port *Host是否健康 *额定用户数 *在线用户数 *坐席在线个数 *匿名在线用户数 *工作线程未处理任务数 *最近3分钟登录次数 *最近3分钟登出次数 *最近3分钟登录用户数 *最近3分钟登出用户数 *最近3分钟查询被叫次数 *最近3分钟查询被叫本地命中次数 *最近3分钟查询被叫DHT查询次数 *最近3分钟转发消息次数 *最近3分钟转发消息CAHCE命中次数 *最近3分钟转发消息DHT查询次数 *最近3分钟转发消息本地命中次数 *最近3分钟发送坐席状态消息次数 *最近3分钟发送用户排队位置消息次数 *最近3分钟向APNS通道推送次数 *最近3分钟向静默通道推送次数 在线用户设备分布列表
	hostId := hs[1]
	ip := hs[2]
	port := hs[3]
	healthy, _ := strconv.Atoi(hs[4])
	fixedUser, _ := strconv.Atoi(hs[5])
	onlineUser, _ := strconv.Atoi(hs[6])
	onlineSeat, _ := strconv.Atoi(hs[7])
	onlineAnonym, _ := strconv.Atoi(hs[8])
	untreatedTask, _ := strconv.Atoi(hs[9])
	login, _ := strconv.Atoi(hs[10])
	logout, _ := strconv.Atoi(hs[11])
	loginUser, _ := strconv.Atoi(hs[12])
	logoutUser, _ := strconv.Atoi(hs[13])
	queryCalled, _ := strconv.Atoi(hs[14])
	queryCalledSuc, _ := strconv.Atoi(hs[15])
	queryCalledDHT, _ := strconv.Atoi(hs[16])
	relayMsg, _ := strconv.Atoi(hs[17])
	relayMsgCAHCESuc, _ := strconv.Atoi(hs[18])
	relayMsgQueryDHT, _ := strconv.Atoi(hs[19])
	relayMsgLocalSuc, _ := strconv.Atoi(hs[20])
	relaySeatMsg, _ := strconv.Atoi(hs[21])
	relayUserQueuePos, _ := strconv.Atoi(hs[22])
	pushAPNS, _ := strconv.Atoi(hs[23])
	pushSilent, _ := strconv.Atoi(hs[24])

	if globeCfg.Output.Prometheus || globeCfg.Output.PushGateway {
		host_healthy.WithLabelValues(hostId, ip, port).Set(float64(healthy))
		host_fixedUser.WithLabelValues(hostId, ip, port).Set(float64(fixedUser))
		host_onlineUser.WithLabelValues(hostId, ip, port).Set(float64(onlineUser))
		host_onlineSeat.WithLabelValues(hostId, ip, port).Set(float64(onlineSeat))
		host_onlineAnonym.WithLabelValues(hostId, ip, port).Set(float64(onlineAnonym))
		host_untreatedTask.WithLabelValues(hostId, ip, port).Set(float64(untreatedTask))
		host_login.WithLabelValues(hostId, ip, port).Set(float64(login))
		host_logout.WithLabelValues(hostId, ip, port).Set(float64(logout))
		host_loginUser.WithLabelValues(hostId, ip, port).Set(float64(loginUser))
		host_logoutUser.WithLabelValues(hostId, ip, port).Set(float64(logoutUser))
		host_queryCalled.WithLabelValues(hostId, ip, port).Set(float64(queryCalled))
		host_queryCalledSuc.WithLabelValues(hostId, ip, port).Set(float64(queryCalledSuc))
		host_queryCalledDHT.WithLabelValues(hostId, ip, port).Set(float64(queryCalledDHT))
		host_relayMsg.WithLabelValues(hostId, ip, port).Set(float64(relayMsg))
		host_relayMsgCAHCESuc.WithLabelValues(hostId, ip, port).Set(float64(relayMsgCAHCESuc))
		host_relayMsgQueryDHT.WithLabelValues(hostId, ip, port).Set(float64(relayMsgQueryDHT))
		host_relayMsgLocalSuc.WithLabelValues(hostId, ip, port).Set(float64(relayMsgLocalSuc))
		host_relaySeatMsg.WithLabelValues(hostId, ip, port).Set(float64(relaySeatMsg))
		host_relayUserQueuePos.WithLabelValues(hostId, ip, port).Set(float64(relayUserQueuePos))
		host_pushAPNS.WithLabelValues(hostId, ip, port).Set(float64(pushAPNS))
		host_pushSilent.WithLabelValues(hostId, ip, port).Set(float64(pushSilent))
	}

	if globeCfg.Output.Telegraf {
		mx := fmt.Sprintf("p2p_host,id=%s,addr=%s:%s healthy=%d,fixed_user=%d,online_user=%d,online_seat=%d,online_anonym=%d,untreated_task=%d,loginX=%d,logoutX=%d,login_userX=%d,logout_userX=%d,query_calledX=%d,query_called_okX=%d,query_called_DHT_X=%d,relay_msgX=%d,relay_msg_CAHCE_okX=%d,relay_msg_query_DHT_X=%d,relay_msg_local_okX=%d,relay_seat_msgX=%d,relay_user_queue_posX=%d,push_APNS_X=%d,push_silentX=%d\n",
			hostId,
			ip,
			port,
			healthy,
			fixedUser,
			onlineUser,
			onlineSeat,
			onlineAnonym,
			untreatedTask,
			login,
			logout,
			loginUser,
			logoutUser,
			queryCalled,
			queryCalledSuc,
			queryCalledDHT,
			relayMsg,
			relayMsgCAHCESuc,
			relayMsgQueryDHT,
			relayMsgLocalSuc,
			relaySeatMsg,
			relayUserQueuePos,
			pushAPNS,
			pushSilent)
		collectBuf.WriteString(mx)
	}
	return nil
}

var ( //statistic.relay.action
	//*并发通话数
	relay_onphone = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "p2p",
			Subsystem: "relay",
			Name:      "onphone",
			Help:      "sum of onphone link",
		},
		[]string{
			"RelayId",
			"IP",
			"Port",
		},
	)
	//*接入|落地用户数
	relay_onconnect = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "p2p",
			Subsystem: "relay",
			Name:      "onconnect",
			Help:      "sum of onconnect link",
		},
		[]string{
			"RelayId",
			"IP",
			"Port",
		},
	)
	//*最近3分钟短链保活消息数
	relay_shortLiveMsg = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "p2p",
			Subsystem: "relay",
			Name:      "new_short_living_msg",
			Help:      "increased sum of short living msg",
		},
		[]string{
			"RelayId",
			"IP",
			"Port",
		},
	)
	//*最近3分钟转发建路包数
	relay_buildingMsg = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "p2p",
			Subsystem: "relay",
			Name:      "new_building_msg",
			Help:      "increased sum of building msg.",
		},
		[]string{
			"RelayId",
			"IP",
			"Port",
		},
	)
	//*最近3分钟转发媒体包数
	relay_media = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "p2p",
			Subsystem: "relay",
			Name:      "new_media_packet",
			Help:      "increased sum of media packet",
		},
		[]string{
			"RelayId",
			"IP",
			"Port",
		},
	)
	//*最近3分钟无效消息数据
	relay_invalidMsg = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "p2p",
			Subsystem: "relay",
			Name:      "new_invalid_msg",
			Help:      "increased sum of invalid message",
		},
		[]string{
			"RelayId",
			"IP",
			"Port",
		},
	)
	//*最近3分钟通话建立次数
	relay_callBeg = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "p2p",
			Subsystem: "relay",
			Name:      "new_call_setup",
			Help:      "increased sum of call setup",
		},
		[]string{
			"RelayId",
			"IP",
			"Port",
		},
	)
	//*最近3分钟通话结束次数
	relay_callEnd = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "p2p",
			Subsystem: "relay",
			Name:      "new_call_end",
			Help:      "increased sum of call end",
		},
		[]string{
			"RelayId",
			"IP",
			"Port",
		},
	)
	//*最近3分钟平均媒体转发上行流量
	relay_upStream = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "p2p",
			Subsystem: "relay",
			Name:      "new_up_stream",
			Help:      "increased sum of up stream",
		},
		[]string{
			"RelayId",
			"IP",
			"Port",
		},
	)
	//*最近3分钟平均媒体转发下行流量
	relay_downStream = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "p2p",
			Subsystem: "relay",
			Name:      "new_down_stream",
			Help:      "increased sum of down stream",
		},
		[]string{
			"RelayId",
			"IP",
			"Port",
		},
	)
)

func regRelay() {
	prometheus.MustRegister(relay_onphone)
	prometheus.MustRegister(relay_onconnect)
	prometheus.MustRegister(relay_shortLiveMsg)
	prometheus.MustRegister(relay_buildingMsg)
	prometheus.MustRegister(relay_media)
	prometheus.MustRegister(relay_invalidMsg)
	prometheus.MustRegister(relay_callBeg)
	prometheus.MustRegister(relay_callEnd)
	prometheus.MustRegister(relay_upStream)
	prometheus.MustRegister(relay_downStream)
}

func extractRelay(ri []string) error {
	//  时间 relay节点id relay IP relay Port *并发通话数 *接入|落地用户数 *最近3分钟短链保活消息数 *最近3分钟转发建路包数 *最近3分钟转发媒体包数 *最近3分钟无效消息数据 *最近3分钟通话建立次数 *最近3分钟通话结束次数 *最近3分钟平均媒体转发上行流量 *最近3分钟平均媒体转发下行流量
	relayId := ri[1]
	ip := ri[2]
	port := ri[3]
	onphone, _ := strconv.Atoi(ri[4])
	onconnect, _ := strconv.Atoi(ri[5])
	shortLiveMsg, _ := strconv.Atoi(ri[6])
	buildingMsg, _ := strconv.Atoi(ri[7])
	media, _ := strconv.Atoi(ri[8])
	invalidMsg, _ := strconv.Atoi(ri[9])
	callSetup, _ := strconv.Atoi(ri[10])
	callEnd, _ := strconv.Atoi(ri[11])
	upStream, _ := strconv.Atoi(ri[12])
	downStream, _ := strconv.Atoi(ri[13])
	if globeCfg.Output.Prometheus || globeCfg.Output.PushGateway {
		relay_onphone.WithLabelValues(relayId, ip, port).Set(float64(onphone))
		relay_onconnect.WithLabelValues(relayId, ip, port).Set(float64(onconnect))
		relay_shortLiveMsg.WithLabelValues(relayId, ip, port).Set(float64(shortLiveMsg))
		relay_buildingMsg.WithLabelValues(relayId, ip, port).Set(float64(buildingMsg))
		relay_media.WithLabelValues(relayId, ip, port).Set(float64(media))
		relay_invalidMsg.WithLabelValues(relayId, ip, port).Set(float64(invalidMsg))
		relay_callBeg.WithLabelValues(relayId, ip, port).Set(float64(callSetup))
		relay_callEnd.WithLabelValues(relayId, ip, port).Set(float64(callEnd))
		relay_upStream.WithLabelValues(relayId, ip, port).Set(float64(upStream))
		relay_downStream.WithLabelValues(relayId, ip, port).Set(float64(downStream))
	}

	if globeCfg.Output.Telegraf {
		mx := fmt.Sprintf("p2p_relay,id=%s,addr=%s:%s onphone=%d,onconnect=%d,short_live_msgX=%d,building_msgX=%d,media_packetX=%d,invalid_msgX=%d,call_setupX=%d,call_endX=%d,up_streamX=%d,down_streamX=%d\n",
			relayId,
			ip,
			port,
			onphone,
			onconnect,
			shortLiveMsg,
			buildingMsg,
			media,
			invalidMsg,
			callSetup,
			callEnd,
			upStream,
			downStream)
		collectBuf.WriteString(mx)
	}

	return nil
}

var ( //statistic.bootstrap.action
	//*3分钟查询次数
	bootstrap_query = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "p2p",
			Subsystem: "bootstrap",
			Name:      "new_query",
			Help:      "increased sum of query",
		},
		[]string{
			"BootstrapId",
			"IP",
			"Port",
		},
	)
	//*当前健康HOST数
	bootstrap_heathyHost = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "p2p",
			Subsystem: "bootstrap",
			Name:      "heathy_host",
			Help:      "sum of heathy host",
		},
		[]string{
			"BootstrapId",
			"IP",
			"Port",
		},
	)
	//*当前HOST总数
	bootstrap_Host = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "p2p",
			Subsystem: "bootstrap",
			Name:      "host",
			Help:      "sum of host",
		},
		[]string{
			"BootstrapId",
			"IP",
			"Port",
		},
	)
	//*路由表长度
	bootstrap_Route = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "p2p",
			Subsystem: "bootstrap",
			Name:      "route_table_len",
			Help:      "route table len",
		},
		[]string{
			"BootstrapId",
			"IP",
			"Port",
		},
	)
)

func regBootstrap() {
	prometheus.MustRegister(bootstrap_query)
	prometheus.MustRegister(bootstrap_heathyHost)
	prometheus.MustRegister(bootstrap_Host)
	prometheus.MustRegister(bootstrap_Route)
}

func extractBootstrap(ps []string) error {
	//  时间 Bootstrap节点ID Bootstrap IP Bootstrap Port *3分钟查询次数 *当前健康HOST数 *当前HOST总数 *路由表长度
	bootstrapId := ps[1]
	ip := ps[2]
	port := ps[3]
	query, _ := strconv.Atoi(ps[4])
	heathyHost, _ := strconv.Atoi(ps[5])
	host, _ := strconv.Atoi(ps[6])
	route, _ := strconv.Atoi(ps[7])
	if globeCfg.Output.Prometheus || globeCfg.Output.PushGateway {
		bootstrap_query.WithLabelValues(bootstrapId, ip, port).Set(float64(query))
		bootstrap_heathyHost.WithLabelValues(bootstrapId, ip, port).Set(float64(heathyHost))
		bootstrap_Host.WithLabelValues(bootstrapId, ip, port).Set(float64(host))
		bootstrap_Route.WithLabelValues(bootstrapId, ip, port).Set(float64(route))
	}
	if globeCfg.Output.Telegraf {
		mx := fmt.Sprintf("p2p_bootstrap,id=%s,addr=%s:%s queryX=%d,heathy_host=%d,host=%d,route_len=%d\n",
			bootstrapId,
			ip,
			port,
			query,
			heathyHost,
			host,
			route)
		collectBuf.WriteString(mx)
	}
	return nil
}

var ( //statistic.DHT.action
	//*DHT连接状态
	dht_status = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "p2p",
			Subsystem: "dht",
			Name:      "status",
			Help:      "connected status",
		},
		[]string{
			"DhtId",
			"HostId",
			"IP",
			"Port",
		},
	)
	//*DHT是否健康
	dht_heathy = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "p2p",
			Subsystem: "dht",
			Name:      "heathy",
			Help:      "heathy or not",
		},
		[]string{
			"DhtId",
			"HostId",
			"IP",
			"Port",
		},
	)
	//*路由表个数
	dht_route = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "p2p",
			Subsystem: "dht",
			Name:      "route_table",
			Help:      "sum of route table",
		},
		[]string{
			"DhtId",
			"HostId",
			"IP",
			"Port",
		},
	)
	//*在线信息用户数
	dht_online = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "p2p",
			Subsystem: "dht",
			Name:      "online",
			Help:      "sum of online user",
		},
		[]string{
			"DhtId",
			"HostId",
			"IP",
			"Port",
		},
	)
	//*ANPS离线信息用户数
	dht_offline = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "p2p",
			Subsystem: "dht",
			Name:      "offline",
			Help:      "sum of offline user",
		},
		[]string{
			"DhtId",
			"HostId",
			"IP",
			"Port",
		},
	)
	//*有静默通道用户数
	dht_silent = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "p2p",
			Subsystem: "dht",
			Name:      "silent",
			Help:      "sum of silent user",
		},
		[]string{
			"DhtId",
			"HostId",
			"IP",
			"Port",
		},
	)
	//*Connect应用总数
	dht_connect = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "p2p",
			Subsystem: "dht",
			Name:      "connect",
			Help:      "sum of connect",
		},
		[]string{
			"DhtId",
			"HostId",
			"IP",
			"Port",
		},
	)
	//Host列表
	//*最近3分钟内GetValue次数
	dht_getvalue = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "p2p",
			Subsystem: "dht",
			Name:      "getvalue",
			Help:      "sum of getvalue",
		},
		[]string{
			"DhtId",
			"HostId",
			"IP",
			"Port",
		},
	)
	//*最近3分钟内SetValue次数
	dht_setvalue = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "p2p",
			Subsystem: "dht",
			Name:      "setvalue",
			Help:      "sum of setvalue",
		},
		[]string{
			"DhtId",
			"HostId",
			"IP",
			"Port",
		},
	)

//GetValue响应速度列表
)

func regDHT() {
	prometheus.MustRegister(dht_status)
	prometheus.MustRegister(dht_heathy)
	prometheus.MustRegister(dht_route)
	prometheus.MustRegister(dht_online)
	prometheus.MustRegister(dht_offline)
	prometheus.MustRegister(dht_silent)
	prometheus.MustRegister(dht_connect)
	prometheus.MustRegister(dht_getvalue)
	prometheus.MustRegister(dht_setvalue)
}

func extractDHT(ps []string) error {
	//  时间 DHT节点id 所属Host节点ID DHT的KAD IP	 DHt的KAD Port *DHT连接状态 *DHT是否健康 *路由表个数 *在线信息用户数 *ANPS离线信息用户数 *有静默通道用户数 *Connect应用总数 Host列表 *最近3分钟内GetValue次数 *最近3分钟内SetValue次数 *GetValue响应速度列表
	dhtId := ps[1]
	hostId := ps[2]
	ip := ps[3]
	port := ps[4]
	status, _ := strconv.Atoi(ps[5])
	heathy, _ := strconv.Atoi(ps[6])
	route, _ := strconv.Atoi(ps[7])
	online, _ := strconv.Atoi(ps[8])
	offline, _ := strconv.Atoi(ps[9])
	silent, _ := strconv.Atoi(ps[10])
	connect, _ := strconv.Atoi(ps[11])
	getvalue, _ := strconv.Atoi(ps[13])
	setvalue, _ := strconv.Atoi(ps[14])
	if globeCfg.Output.Prometheus || globeCfg.Output.PushGateway {
		dht_status.WithLabelValues(dhtId, hostId, ip, port).Set(float64(status))
		dht_heathy.WithLabelValues(dhtId, hostId, ip, port).Set(float64(heathy))
		dht_route.WithLabelValues(dhtId, hostId, ip, port).Set(float64(route))
		dht_online.WithLabelValues(dhtId, hostId, ip, port).Set(float64(online))
		dht_offline.WithLabelValues(dhtId, hostId, ip, port).Set(float64(offline))
		dht_silent.WithLabelValues(dhtId, hostId, ip, port).Set(float64(silent))
		dht_connect.WithLabelValues(dhtId, hostId, ip, port).Set(float64(connect))
		//Host列表
		dht_getvalue.WithLabelValues(dhtId, hostId, ip, port).Set(float64(getvalue))
		dht_setvalue.WithLabelValues(dhtId, hostId, ip, port).Set(float64(setvalue))
		//GetValue响应速度列表
	}

	if globeCfg.Output.Telegraf {
		mx := fmt.Sprintf("p2p_dht,id=%s,hostId=%s,addr=%s:%s status=%d,heathy=%d,host=%d,route_table=%d,online=%d,offline=%d,silent=%d,connect=%d,getvalue=%d,setvalue=%d\n",
			dhtId,
			hostId,
			ip,
			port,
			status,
			heathy,
			route,
			online,
			offline,
			silent,
			connect,
			getvalue,
			setvalue)
		collectBuf.WriteString(mx)
	}

	return nil
}

var ( //statistic.SPS.action
	//*通道[信令双通道+静默通道]连接数
	sps_connect = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "p2p",
			Subsystem: "sps",
			Name:      "connect",
			Help:      "connected link",
		},
		[]string{
			"SpsId",
			"HostId",
			"IP",
			"Port",
		},
	)
	//*最近3分钟发送消息总数
	sps_msg = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "p2p",
			Subsystem: "sps",
			Name:      "new_send_msg",
			Help:      "sum of send message",
		},
		[]string{
			"SpsId",
			"HostId",
			"IP",
			"Port",
		},
	)
	//*最近3分钟双通道发送给Host消息数
	sps_hostMsg = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "p2p",
			Subsystem: "sps",
			Name:      "new_send_host_msg",
			Help:      "sum of send host message",
		},
		[]string{
			"SpsId",
			"HostId",
			"IP",
			"Port",
		},
	)
	//*最近3分钟双通道给客发送客户端消息数
	sps_clientMsg = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "p2p",
			Subsystem: "sps",
			Name:      "new_send_client_msg",
			Help:      "sum of send client message",
		},
		[]string{
			"SpsId",
			"HostId",
			"IP",
			"Port",
		},
	)
	//*最近3分钟静默通道推送次数
	sps_silentMsg = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "p2p",
			Subsystem: "sps",
			Name:      "new_send_silent_msg",
			Help:      "sum of send silent message",
		},
		[]string{
			"SpsId",
			"HostId",
			"IP",
			"Port",
		},
	)
	//*最近3分钟静默通道推送成功次数
	sps_silentMsgOk = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "p2p",
			Subsystem: "sps",
			Name:      "new_send_host_msg_ok",
			Help:      "sum of send host message successed",
		},
		[]string{
			"SpsId",
			"HostId",
			"IP",
			"Port",
		},
	)
)

func regSPS() {
	prometheus.MustRegister(sps_connect)
	prometheus.MustRegister(sps_msg)
	prometheus.MustRegister(sps_hostMsg)
	prometheus.MustRegister(sps_clientMsg)
	prometheus.MustRegister(sps_silentMsg)
	prometheus.MustRegister(sps_silentMsgOk)

}

func extractSPS(ps []string) error {
	//  时间 SPS节点id 所属Host节点ID SPS IP SPS Port *通道[信令双通道+静默通道]连接数 *最近3分钟发送消息总数 *最近3分钟双通道发送给Host消息数 8最近3分钟双通道给客发送客户端消息数 *最近3分钟静默通道推送次数 *最近3分钟静默通道推送成功次数
	spsId := ps[1]
	hostId := ps[2]
	ip := ps[3]
	port := ps[4]
	connect, _ := strconv.Atoi(ps[5])
	msg, _ := strconv.Atoi(ps[6])
	hostMsg, _ := strconv.Atoi(ps[7])
	clientMsg, _ := strconv.Atoi(ps[8])
	silentMsg, _ := strconv.Atoi(ps[9])
	silentMsgOk, _ := strconv.Atoi(ps[10])
	if globeCfg.Output.Prometheus || globeCfg.Output.PushGateway {
		sps_connect.WithLabelValues(spsId, hostId, ip, port).Set(float64(connect))
		sps_msg.WithLabelValues(spsId, hostId, ip, port).Set(float64(msg))
		sps_hostMsg.WithLabelValues(spsId, hostId, ip, port).Set(float64(hostMsg))
		sps_clientMsg.WithLabelValues(spsId, hostId, ip, port).Set(float64(clientMsg))
		sps_silentMsg.WithLabelValues(spsId, hostId, ip, port).Set(float64(silentMsg))
		sps_silentMsgOk.WithLabelValues(spsId, hostId, ip, port).Set(float64(silentMsgOk))
	}

	if globeCfg.Output.Telegraf {
		mx := fmt.Sprintf("p2p_sps,id=%s,hostId=%s,addr=%s:%s connect=%d,send_msgX=%d,send_host_msgX=%d,send_client_msgX=%d,send_silent_msgX=%d,send_silent_msg_okX=%d\n",
			spsId,
			hostId,
			ip,
			port,
			connect,
			msg,
			hostMsg,
			clientMsg,
			silentMsg,
			silentMsgOk)
		collectBuf.WriteString(mx)
	}

	return nil
}

var ( //statistic.ANPS.action
	//*与APNS连接成功通道数
	anps_connect = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "p2p",
			Subsystem: "apns",
			Name:      "connect",
			Help:      "connected link",
		},
		[]string{
			"ApnsId",
			"HostId",
			"IP",
			"Port",
		},
	)
	//*待推送的任务数
	anps_task = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "p2p",
			Subsystem: "apns",
			Name:      "task",
			Help:      "sum of task",
		},
		[]string{
			"ApnsId",
			"HostId",
			"IP",
			"Port",
		},
	)
	//*最近3分钟推送总数
	anps_pushed = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "p2p",
			Subsystem: "apns",
			Name:      "new_pushed",
			Help:      "sum of pushed msg",
		},
		[]string{
			"ApnsId",
			"HostId",
			"IP",
			"Port",
		},
	)
	//*最近3分钟推送成功次数
	anps_pushSucced = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "p2p",
			Subsystem: "apns",
			Name:      "new_push_succed",
			Help:      "sum of pushed msg succed",
		},
		[]string{
			"ApnsId",
			"HostId",
			"IP",
			"Port",
		},
	)
	//*最近3分钟推送失败次数
	anps_pushFailed = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "p2p",
			Subsystem: "apns",
			Name:      "new_push_failed",
			Help:      "sum of pushed msg failed",
		},
		[]string{
			"ApnsId",
			"HostId",
			"IP",
			"Port",
		},
	)
)

func regANPS() {
	prometheus.MustRegister(anps_connect)
	prometheus.MustRegister(anps_task)
	prometheus.MustRegister(anps_pushed)
	prometheus.MustRegister(anps_pushSucced)
	prometheus.MustRegister(anps_pushFailed)
}

func extractANPS(ps []string) error {
	//  时间 PS节点id 所属Host节点ID PS IP PS Port *与APNS连接成功通道数 *待推送的任务数 *最近3分钟推送总数 *最近3分钟推送成功次数 *最近3分钟推送失败次数
	anpsId := ps[1]
	hostId := ps[2]
	ip := ps[3]
	port := ps[4]
	connect, _ := strconv.Atoi(ps[5])
	task, _ := strconv.Atoi(ps[6])
	pushed, _ := strconv.Atoi(ps[7])
	pushSucced, _ := strconv.Atoi(ps[8])
	pushFailed, _ := strconv.Atoi(ps[9])
	if globeCfg.Output.Prometheus || globeCfg.Output.PushGateway {
		anps_connect.WithLabelValues(anpsId, hostId, ip, port).Set(float64(connect))
		anps_task.WithLabelValues(anpsId, hostId, ip, port).Set(float64(task))
		anps_pushed.WithLabelValues(anpsId, hostId, ip, port).Set(float64(pushed))
		anps_pushSucced.WithLabelValues(anpsId, hostId, ip, port).Set(float64(pushSucced))
		anps_pushFailed.WithLabelValues(anpsId, hostId, ip, port).Set(float64(pushFailed))
	}

	if globeCfg.Output.Telegraf {
		mx := fmt.Sprintf("p2p_anps,id=%s,hostId=%s,addr=%s:%s connect=%d,task=%d,pushedX=%d,push_okX=%d,push_nokX=%d\n",
			anpsId,
			hostId,
			ip,
			port,
			connect,
			task,
			pushed,
			pushSucced,
			pushFailed)
		collectBuf.WriteString(mx)
	}
	return nil
}

var ( //statistic.CM.action
	//*当前通话数
	cm_onphone = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "p2p",
			Subsystem: "cm",
			Name:      "onphone",
			Help:      "sum of onphone",
		},
		[]string{
			"CmId",
			"HostId",
			"IP",
			"Port",
		},
	)
	//*最近3分钟视频通数
	cm_onphoneV = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "p2p",
			Subsystem: "cm",
			Name:      "new_onphone_video",
			Help:      "increased sum of onphone video",
		},
		[]string{
			"CmId",
			"HostId",
			"IP",
			"Port",
		},
	)
	//*最近3分钟音频话数
	cm_onphoneA = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "p2p",
			Subsystem: "cm",
			Name:      "new_onphone_audio",
			Help:      "increased sum of onphone audio",
		},
		[]string{
			"CmId",
			"HostId",
			"IP",
			"Port",
		},
	)
	//*最近3分钟正常挂断通话数
	cm_hangup = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "p2p",
			Subsystem: "cm",
			Name:      "new_released",
			Help:      "increased sum of released",
		},
		[]string{
			"CmId",
			"HostId",
			"IP",
			"Port",
		},
	)
	//*最近3分钟异常挂断通话数
	cm_broken = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "p2p",
			Subsystem: "cm",
			Name:      "new_broken",
			Help:      "increased sum of broken call",
		},
		[]string{
			"CmId",
			"HostId",
			"IP",
			"Port",
		},
	)
	//*最近3分钟系统原因未接通数
	cm_blockBySys = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "p2p",
			Subsystem: "cm",
			Name:      "new_block_by_sys",
			Help:      "increased sum of call block by system",
		},
		[]string{
			"CmId",
			"HostId",
			"IP",
			"Port",
		},
	)
	//*最近3分钟人为原因未接通数
	cm_blockByOps = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "p2p",
			Subsystem: "cm",
			Name:      "new_block_by_man",
			Help:      "increased sum of call block by man",
		},
		[]string{
			"CmId",
			"HostId",
			"IP",
			"Port",
		},
	)
	//*最近3分钟被叫不在线未接通数
	cm_blockOffline = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "p2p",
			Subsystem: "cm",
			Name:      "new_block_called_offline",
			Help:      "increased sum of call block by called offline",
		},
		[]string{
			"CmId",
			"HostId",
			"IP",
			"Port",
		},
	)
)

func regCM() {
	prometheus.MustRegister(cm_onphone)
	prometheus.MustRegister(cm_onphoneV)
	prometheus.MustRegister(cm_onphoneA)
	prometheus.MustRegister(cm_hangup)
	prometheus.MustRegister(cm_broken)
	prometheus.MustRegister(cm_blockBySys)
	prometheus.MustRegister(cm_blockByOps)
	prometheus.MustRegister(cm_blockOffline)
}

func extractCM(ps []string) error {
	//  时间 CallMgr节点ID 所属Host节点ID CallMgr IP CallMgr port *当前通话数 *最近3分钟视频通数 *最近3分钟音频话数 *最近3分钟正常挂断通话数 *最近3分钟异常挂断通话数 *最近3分钟系统原因未接通数 *最近3分钟人为原因未接通数 *最近3分钟被叫不在线未接通数
	cmId := ps[1]
	hostId := ps[2]
	ip := ps[3]
	port := ps[4]
	onphone, _ := strconv.Atoi(ps[5])
	onphoneV, _ := strconv.Atoi(ps[6])
	onphoneA, _ := strconv.Atoi(ps[7])
	hangup, _ := strconv.Atoi(ps[8])
	broken, _ := strconv.Atoi(ps[9])
	blockBySys, _ := strconv.Atoi(ps[10])
	blockByOps, _ := strconv.Atoi(ps[11])
	blockOffline, _ := strconv.Atoi(ps[12])
	cm_onphone.WithLabelValues(cmId, hostId, ip, port).Set(float64(onphone))
	cm_onphoneV.WithLabelValues(cmId, hostId, ip, port).Set(float64(onphoneV))
	cm_onphoneA.WithLabelValues(cmId, hostId, ip, port).Set(float64(onphoneA))
	cm_hangup.WithLabelValues(cmId, hostId, ip, port).Set(float64(hangup))
	cm_broken.WithLabelValues(cmId, hostId, ip, port).Set(float64(broken))
	cm_blockBySys.WithLabelValues(cmId, hostId, ip, port).Set(float64(blockBySys))
	cm_blockByOps.WithLabelValues(cmId, hostId, ip, port).Set(float64(blockByOps))
	cm_blockOffline.WithLabelValues(cmId, hostId, ip, port).Set(float64(blockOffline))

	if globeCfg.Output.Telegraf {
		mx := fmt.Sprintf("p2p_cm,id=%s,hostId=%s,addr=%s:%s onphone=%d,onvideoX=%d,onaudioX=%d,releasedX=%d,brokenX=%d,sys_blockX=%d,ops_blockX=%d,offline_blockX=%d\n",
			cmId,
			hostId,
			ip,
			port,
			onphone,
			onphoneV,
			onphoneA,
			hangup,
			broken,
			blockBySys,
			blockByOps,
			blockOffline)
		collectBuf.WriteString(mx)
	}
	return nil
}

var ( //test
	call_vdn_err = prometheus.NewCounter(
		prometheus.CounterOpts{
			Namespace: "p2p",
			Subsystem: "ops",
			Name:      "access_vpn_error",
			Help:      "call vdn-rest error counter.",
		},
	)
)

var (
	collectBuf bytes.Buffer
	tcpConnect net.Conn
)

func init() {
	loadCfg()
	if globeCfg.Output.Prometheus || globeCfg.Output.PushGateway {
		regServerSummary()
		regUserStatistic()
		regCallStatistic()
		regHost()
		regRelay()
		regBootstrap()
		regDHT()
		regSPS()
		regANPS()
		regCM()
		prometheus.MustRegister(call_vdn_err)
	}

	if globeCfg.Output.Telegraf {
		ta := strings.Split(globeCfg.Output.TelegrafAddr, "://")
		if len(ta) != 2 {
			log.Fatal("invalid TelegrafAddr:", globeCfg.Output.TelegrafAddr)
		}
		var err error = nil
		tcpConnect, err = net.Dial(ta[0], ta[1])
		if err != nil {
			log.Fatal("Dial TelegrafAddr:", err)
		}
	}
}

type VMDExtractor func([]string) error
type VMDParser func(string, VMDExtractor) error

type VdnMonitorData struct {
	Data   []string `json:"data"`
	Desc   []string `json:"desc"`
	Result int      `json:"result"`
}

func parseVdnMonitorData(data string, extractor VMDExtractor) error {
	vmd := VdnMonitorData{}
	if err := json.Unmarshal([]byte(data), &vmd); err != nil {
		return err
	}

	if vmd.Result != 0 {
		return errors.New("vdn response:" + fmt.Sprintf("%d", vmd.Result))
	}

	for _, d := range vmd.Data {
		infos := strings.Split(d, "|")
		if len(vmd.Desc) != len(infos) {
			return errors.New("vdn response: error data.")
		}

		if err := extractor(infos); err != nil {
			return err
		}
	}

	return nil
}

func main() {
	if globeCfg.Output.Prometheus {
		setupFakeServer()
	}

	go func() {
		apis := []struct {
			api       string
			extractor VMDExtractor
		}{
			//    获取数据的http端口
			//			{api: "statistic.serverSummary.action", extractor: extractServerSummary},
			//			{api: "statistic.userStatistic.action", extractor: extractUserStatistic},
			//			{api: "statistic.callStatistic.action", extractor: extractCallStatistic},
			//			//			//TODO:statistic.acd.action
			//			//			//TODO:statistic.im.action
			//			{api: "statistic.host.action", extractor: extractHost},
			//			{api: "statistic.relay.action", extractor: extractRelay},
			//			{api: "statistic.bootstrap.action", extractor: extractBootstrap},
			//			{api: "statistic.DHT.action", extractor: extractDHT},
			//			{api: "statistic.SPS.action", extractor: extractSPS},
			//			{api: "statistic.ANPS.action", extractor: extractANPS},
			//			{api: "statistic.CM.action", extractor: extractCM},
			//			//TODO:statistic.rc.action
			// 从本地文件获取数据
			{api: globeCfg.Fileaddress.Server_sumary, extractor: extractServerSummary},
			{api: globeCfg.Fileaddress.User_statistic, extractor: extractUserStatistic},
			{api: globeCfg.Fileaddress.Call_statistic, extractor: extractCallStatistic},
			{api: globeCfg.Fileaddress.Host_info, extractor: extractHost},
			//{api: globeCfg.Fileaddress.Relay, extractor: extractRelay}
			{api: globeCfg.Fileaddress.Bootstrap, extractor: extractBootstrap},
			{api: globeCfg.Fileaddress.Dht, extractor: extractDHT},
			{api: globeCfg.Fileaddress.Sps, extractor: extractSPS},
			{api: globeCfg.Fileaddress.Ps, extractor: extractANPS},
			{api: globeCfg.Fileaddress.Callmgr, extractor: extractCM},
		}
		for {

			if err := relayfileToPrometheus(globeCfg.Fileaddress.Relay, extractRelay); err != nil {
				log.Println("fileToPrometheus:", err)
				call_vdn_err.Inc()
			} else {
				if globeCfg.Output.Telegraf {
					fmt.Fprintf(tcpConnect, collectBuf.String())
					//fmt.Println(collectBuf.String())
					collectBuf.Reset()
				}
			}
			for _, api := range apis {
				//				log.Println(api.api)
				// 从本地文件获取数据
				if err := fileToPrometheus(api.api, api.extractor); err != nil {
					log.Println("fileToPrometheus:", err)
					call_vdn_err.Inc()
				} else {
					if globeCfg.Output.Telegraf {
						fmt.Fprintf(tcpConnect, collectBuf.String())
						//fmt.Println(collectBuf.String())
						collectBuf.Reset()
					}
				}

			}

			if globeCfg.Output.PushGateway {
				// Push registry, all good.
				if err := push.FromGatherer("p2p", push.HostnameGroupingKey(), globeCfg.Output.PushGatewayAddr, prometheus.DefaultGatherer); err != nil {
					log.Println("FromGatherer:", err)
				}
			}

			time.Sleep(time.Duration(globeCfg.Rest.Period) * time.Second)
		}
	}()

	if globeCfg.Output.Prometheus {
		go func() {
			http.Handle("/metrics", promhttp.Handler())
			log.Fatal(http.ListenAndServe(fmt.Sprintf("%s:%d", globeCfg.Gw.Addr, globeCfg.Gw.HttpListenPort), nil))
		}()
	}

	c := make(chan os.Signal, 1)
	signal.Notify(c)
	signal.Notify(c, os.Interrupt, os.Kill)
	s := <-c
	if fakeServer != nil {
		fakeServer.Close()
	}
	log.Println("exit.", s)
	//lgr.Rotate()
}

func loadCfg() {
	cfgbuf, err := ioutil.ReadFile("cfg.yaml")
	if err != nil {
		panic("not found cfg.yaml")
	}
	gwc := GWConfig{}
	err = yaml.Unmarshal(cfgbuf, &gwc)
	if err != nil {
		panic("invalid cfg.yaml")
	}
	globeCfg = &gwc

	lgr := &ebase.Logger{
		Filename:   gwc.Logger.Filename,
		MaxSize:    gwc.Logger.MaxSize,
		MaxBackups: gwc.Logger.MaxBackups,
		MaxAge:     gwc.Logger.MaxAge,
	}
	if gwc.Logger.Filename == "stdout" {
		log.SetOutput(os.Stdout)
	} else {
		log.SetOutput(lgr)
	}
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)

	log.Println("cfg:", gwc)
}

var fakeServer *httptest.Server = nil

func setupFakeServer() {
	if globeCfg.Rest.Test {
		fakeServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

			fmt.Println(r.Header)
			fmt.Println(r.URL.String())
			body, _ := ioutil.ReadAll(r.Body)
			fmt.Println(string(body))
			fmt.Fprintln(w, `{"resultCode":"200", "resultMsg":"成功"}`)
		}))
		globeCfg.Rest.Vdn = fakeServer.URL
	}
}

func timeInMilliseconds() int64 {
	return time.Now().UnixNano() / 1000000
}
func check(e error) error {
	if e != nil {
		return e
	}
	return nil
}

// param file：flag文件的路径 extractor：flag文件对应的promtheus函数
func fileToPrometheus(file string, extractor VMDExtractor) error {
	f, err := os.Open(file)
	check(err)
	defer f.Close()
	buf := bufio.NewReader(f)
	line, err := buf.ReadString('\n') //读取flag文件 找到当前写到的txt文件

	flag := strings.Split(string(line), "|")
	fmt.Println(flag)
	pathDir := filepath.Dir(file)
	flietxt := strings.TrimSpace(flag[0])
	txtf, err := os.Open(path.Join(pathDir, flietxt)) //读取txt文件
	check(err)
	defer txtf.Close()
	txtbuf := bufio.NewReader(txtf)

	txtstring, err := ioutil.ReadAll(txtbuf)
	check(err)
	lines := strings.Split(string(txtstring), "\n")
	i, err := strconv.Atoi(flag[1])
	check(err)
	j, err := strconv.Atoi(flag[2])
	check(err)
	//信息传给prometheus
	for i <= j {
		strss := string(lines[i-1])
		strss = strings.Replace(strss, "\r", "", -1)
		infos := strings.Split(strss, "|")
		fmt.Println(infos)
		if err := extractor(infos); err != nil {
			return err
		}
		i++
	}
	return nil
}

func relayfileToPrometheus(file string, extractor VMDExtractor) error {
	//relay 节点 map

	relaymap := make(map[string]int)
	relaymap["103.25.23.121|19"] = 0
	relaymap["103.25.23.122|20"] = 0
	relaymap["114.112.74.12|2"] = 0
	relaymap["122.13.78.226|5"] = 0
	relaymap["123.138.91.24|9"] = 0
	relaymap["124.116.176.115|10"] = 0
	relaymap["125.211.202.28|7"] = 0
	relaymap["125.88.254.159|6"] = 0
	relaymap["175.102.21.33|3"] = 0
	relaymap["175.102.8.227|4"] = 0
	relaymap["210.51.168.108|1"] = 0
	relaymap["220.249.119.217|13"] = 0
	relaymap["221.7.112.74|11"] = 0
	relaymap["222.171.242.142|8"] = 0
	relaymap["223.111.205.86|21"] = 0
	relaymap["223.111.205.90|23"] = 0
	relaymap["61.183.245.140|14"] = 0

	f, err := os.Open(file)
	check(err)
	defer f.Close()
	buf := bufio.NewReader(f)
	line, err := buf.ReadString('\n') //读取flag文件 找到当前写到的txt文件

	flag := strings.Split(string(line), "|")
	fmt.Println(flag)
	pathDir := filepath.Dir(file)
	flietxt := strings.TrimSpace(flag[0])
	txtf, err := os.Open(path.Join(pathDir, flietxt)) //读取txt文件
	check(err)
	defer txtf.Close()
	txtbuf := bufio.NewReader(txtf)

	txtstring, err := ioutil.ReadAll(txtbuf)
	check(err)
	lines := strings.Split(string(txtstring), "\n")
	i, err := strconv.Atoi(flag[1])
	check(err)
	j, err := strconv.Atoi(flag[2])
	check(err)
	//信息传给prometheus
	if j-i == 16 { //判断有无relay节点挂掉
		for i <= j {
			strss := string(lines[i-1])
			strss = strings.Replace(strss, "\r", "", -1)
			infos := strings.Split(strss, "|")

			if err := extractor(infos); err != nil {
				return err
			}
			i++
		}
	} else {

		for i <= j {
			strss := string(lines[i-1])
			strss = strings.Replace(strss, "\r", "", -1)
			infos := strings.Split(strss, "|")

			if err := extractor(infos); err != nil {
				return err
			}
			str := infos[2] + "|" + infos[1]
			relaymap[str] = 1
			i++
		}
		for k, v := range relaymap {
			if v == 0 {
				removerelay(k)
			}
			if v == 1 {
				v = 0
			}
		}
	}
	return nil
}
func removerelay(ip string) {
	relayinfos := strings.Split(ip, "|")
	relay_onphone.DeleteLabelValues(relayinfos[1], relayinfos[0], "9000")
}
