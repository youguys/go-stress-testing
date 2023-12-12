package golink

import (
	"context"
	"net"
	"sync"
	"time"

	"github.com/link1st/go-stress-testing/helper"
	"github.com/link1st/go-stress-testing/model"
)

// UdpRaw 接口请求
func UdpRaw(ctx context.Context, chanID uint64, ch chan<- *model.RequestResults, totalNumber uint64, wg *sync.WaitGroup,
	request *model.Request) {
	defer func() {
		wg.Done()
	}()
	for i := uint64(0); i < totalNumber; i++ {
		spaRequest(chanID, ch, i, request)
	}
	return
}

// grpcRequest 请求
func spaRequest(chanID uint64, ch chan<- *model.RequestResults, i uint64, request *model.Request) {
	var (
		startTime = time.Now()
		isSucceed = false
		errCode   = model.RequestErr
	)
	// 需要发送的数据
	// 解析远程地址
	remoteAddr, err := net.ResolveUDPAddr("udp", request.URL)
	if err == nil {
		conn, err := net.DialUDP("udp", nil, remoteAddr)
		if err == nil {
			data := []byte(request.Body)
			_, err = conn.Write(data)
			if err == nil {
				// 接收服务器的响应
				timeout := 6 * time.Second
				conn.SetDeadline(time.Now().Add(timeout))
				buffer := make([]byte, 1024)
				_, _, err := conn.ReadFromUDP(buffer)
				if err == nil {
					isSucceed = true
					errCode = model.HTTPOk
				} else {
					errCode = model.ReadTimeout
				}
			}
			conn.Close()
		}
	}

	requestTime := uint64(helper.DiffNano(startTime))
	requestResults := &model.RequestResults{
		Time:      requestTime,
		IsSucceed: isSucceed,
		ErrCode:   errCode,
	}
	requestResults.SetID(chanID, i)
	ch <- requestResults
}
