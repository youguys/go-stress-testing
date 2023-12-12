// Package verify 校验
package verify

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/link1st/go-stress-testing/model"
)

// getZipData 处理gzip压缩
func getZipData(response *http.Response) (body []byte, err error) {
	var reader io.ReadCloser
	switch response.Header.Get("Content-Encoding") {
	case "gzip":
		reader, err = gzip.NewReader(response.Body)
		defer func() {
			_ = reader.Close()
		}()
	default:
		reader = response.Body
	}
	body, err = io.ReadAll(reader)
	response.Body = io.NopCloser(bytes.NewReader(body))
	return
}

// HTTPStatusCode 通过 HTTP 状态码判断是否请求成功
func HTTPStatusCode(request *model.Request, response *http.Response, body []byte) (code int, isSucceed bool) {
	code = response.StatusCode
	if code == request.Code {
		isSucceed = true
	}
	// 开启调试模式
	if request.GetDebug() {
		fmt.Printf("请求结果 httpCode:%d body:%s \n", response.StatusCode, string(body))
	}
	return
}

/***************************  返回值为json  ********************************/

// ResponseJSON 返回数据结构体
type ResponseJSON struct {
	Code string      `json:"code"`
	Msg  string      `json:"msg"`
	Data interface{} `json:"data"`
}

// HTTPJson  通过返回的Body 判断
// 返回示例: {"code":200,"msg":"Success","data":{}}
// code 默认将http code作为返回码，http code 为200时 取body中的返回code
func HTTPJson(request *model.Request, response *http.Response, body []byte) (code int, isSucceed bool) {
	code = response.StatusCode
	if code == http.StatusOK {
		responseJSON := &ResponseJSON{}
		if err := json.Unmarshal(body, responseJSON); err != nil {
			code = model.ParseError
			fmt.Printf("请求结果 json.Unmarshal err:%v", err)
		} else {
			// body 中code返回200为返回数据成功
			if responseJSON.Code == "0" {
				isSucceed = true
			}
		}
		// 开启调试模式
		if request.GetDebug() {
			fmt.Printf("请求结果 httpCode:%d body:%s  \n", response.StatusCode, string(body))
		}
	}
	return
}
