// Package client http 客户端
package client

import (
	"crypto/tls"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/link1st/go-stress-testing/model"
	httplongclinet "github.com/link1st/go-stress-testing/server/client/http_longclinet"
	"golang.org/x/net/http2"

	"bytes"
	"crypto/x509"
	"fmt"

	"github.com/link1st/go-stress-testing/helper"
	"github.com/quic-go/quic-go"
	"github.com/quic-go/quic-go/http3"
	"io"
)

// logErr err
var logErr = log.New(os.Stderr, "", 0)

// HTTPRequest HTTP 请求
// method 方法 GET POST
// url 请求的url
// body 请求的body
// headers 请求头信息
// timeout 请求超时时间
func HTTPRequest(chanID uint64, request *model.Request) (resp *http.Response, requestTime uint64, err error, body1 []byte) {
	method := request.Method
	url := request.URL
	body := request.GetBody()
	timeout := request.Timeout
	headers := request.Headers

	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return
	}

	// 在req中设置Host，解决在header中设置Host不生效问题
	if _, ok := headers["Host"]; ok {
		req.Host = headers["Host"]
	}
	// 设置默认为utf-8编码
	if _, ok := headers["Content-Type"]; !ok {
		if headers == nil {
			headers = make(map[string]string)
		}
		headers["Content-Type"] = "application/x-www-form-urlencoded; charset=utf-8"
	}
	for key, value := range headers {
		req.Header.Set(key, value)
	}
	var client *http.Client
	if request.Keepalive {
		client = httplongclinet.NewClient(chanID, request)
		startTime := time.Now()
		resp, err = client.Do(req)
		requestTime = uint64(helper.DiffNano(startTime))
		if err != nil {
			logErr.Println("请求失败:", err)

			return
		}
		return
	} else {
		req.Close = true
		var roundTri http.RoundTripper

		switch request.HttpVersion {
		case "2":
			// 使用真实证书 验证证书 模拟真实请求
			tr := &http.Transport{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: false},
			}
			if err = http2.ConfigureTransport(tr); err != nil {
				return
			}
			roundTri = tr
			break
		case "3":
			pool, err1 := x509.SystemCertPool()
			if err1 != nil {
				logErr.Println("pool初始化失败:", err)
			}
			var keyLog io.Writer
			var qconf quic.Config
			//用短连接发送请求，避免长链接带来的不稳定
			qconf.KeepAlivePeriod = 0
			tr := &http3.RoundTripper{
				TLSClientConfig: &tls.Config{
					RootCAs:            pool,
					InsecureSkipVerify: true,
					KeyLogWriter:       keyLog,
				},
				QuicConfig: &qconf,
			}
			defer tr.Close()
			hclient := &http.Client{
				Transport: tr,
			}
			startTime := time.Now()
			resp, err = hclient.Post(url, "application/json", body)
			if err != nil {
				logErr.Println("请求失败:", err)
				return
			}
			requestTime = uint64(helper.DiffNano(startTime))
			defer resp.Body.Close()
			respBody := &bytes.Buffer{}
			_, err1 = io.Copy(respBody, resp.Body)
			if err1 != nil {
				fmt.Println("body解析失败:", err1)
			}
			body1 = respBody.Bytes()
			return
		default:
			// 跳过证书验证
			tr := &http.Transport{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			}
			roundTri = tr
		}

		client = &http.Client{
			Transport: roundTri,
			Timeout:   timeout,
		}
	}

	startTime := time.Now()
	resp, err = client.Do(req)
	requestTime = uint64(helper.DiffNano(startTime))
	if err != nil {
		logErr.Println("请求失败:", err)
		return
	}
	return
}
