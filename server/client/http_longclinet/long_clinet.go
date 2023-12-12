package httplongclinet

import (
	"crypto/tls"
	"net"
	"net/http"
	"sync"
	"time"

	"crypto/x509"
	"fmt"
	"github.com/link1st/go-stress-testing/model"
	"github.com/quic-go/quic-go"
	"github.com/quic-go/quic-go/http3"
	"golang.org/x/net/http2"
	"io"
)

var (
	mutex   sync.RWMutex
	clients = make(map[uint64]*http.Client, 0)
)

// NewClient new
func NewClient(i uint64, request *model.Request) *http.Client {
	client := getClient(i)
	if client != nil {
		return client
	}
	return setClient(i, request)
}

func getClient(i uint64) *http.Client {
	mutex.RLock()
	defer mutex.RUnlock()
	return clients[i]
}

func setClient(i uint64, request *model.Request) *http.Client {
	mutex.Lock()
	defer mutex.Unlock()
	client := createLangHttpClient(request)
	clients[i] = client
	return client
}

// createLangHttpClient 初始化长连接客户端参数
func createLangHttpClient(request *model.Request) *http.Client {
	var roundTri http.RoundTripper

	switch request.HttpVersion {
	case "2":
		// 使用真实证书 验证证书 模拟真实请求
		tr := &http.Transport{
			DialContext: (&net.Dialer{
				Timeout:   30 * time.Second,
				KeepAlive: 30 * time.Second,
			}).DialContext,
			MaxIdleConns:        0,                // 最大连接数,默认0无穷大
			MaxIdleConnsPerHost: request.MaxCon,   // 对每个host的最大连接数量(MaxIdleConnsPerHost<=MaxIdleConns)
			IdleConnTimeout:     90 * time.Second, // 多长时间未使用自动关闭连接
			TLSClientConfig:     &tls.Config{InsecureSkipVerify: false},
		}
		_ = http2.ConfigureTransport(tr)
		roundTri = tr
		break
	case "3":
		pool, err := x509.SystemCertPool()
		if err != nil {
			fmt.Println("请求失败:", err)
		}
		var keyLog io.Writer
		var qconf quic.Config
		//用短连接发送请求，避免长链接带来的不稳定
		qconf.KeepAlivePeriod = 0
		roundTrpper := &http3.RoundTripper{
			TLSClientConfig: &tls.Config{
				RootCAs:            pool,
				InsecureSkipVerify: true,
				KeyLogWriter:       keyLog,
			},
			QuicConfig: &qconf,
		}
		defer roundTrpper.Close()
		roundTri = roundTrpper
		break
	default:
		// 跳过证书验证
		roundTri = &http.Transport{
			DialContext: (&net.Dialer{
				Timeout:   30 * time.Second,
				KeepAlive: 30 * time.Second,
			}).DialContext,
			MaxIdleConns:        0,                // 最大连接数,默认0无穷大
			MaxIdleConnsPerHost: request.MaxCon,   // 对每个host的最大连接数量(MaxIdleConnsPerHost<=MaxIdleConns)
			IdleConnTimeout:     90 * time.Second, // 多长时间未使用自动关闭连接
			TLSClientConfig:     &tls.Config{InsecureSkipVerify: true},
		}
	}
	return &http.Client{
		Transport: roundTri,
	}
}
