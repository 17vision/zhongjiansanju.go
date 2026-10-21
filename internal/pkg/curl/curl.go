package curl

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"strings"

	"github.com/gogf/gf/v2/encoding/gjson"
	"github.com/gogf/gf/v2/errors/gerror"
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/net/gclient"
	"github.com/gogf/gf/v2/net/ghttp"
)

type HttpClient struct {
	client *gclient.Client
}

func NewHttpClient() *HttpClient {
	return &HttpClient{
		client: g.Client(),
	}
}

type Message struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// 统一解析返回公共方法
func parseResponse(res *gclient.Response) (string, error) {
	body := res.ReadAllString()
	if res.StatusCode == http.StatusOK || res.StatusCode == http.StatusNoContent {
		return body, nil
	}
	var msgObj Message
	if err := gjson.Unmarshal([]byte(body), &msgObj); err == nil && msgObj.Message != "" {
		return "", gerror.Newf("[%s] %s", msgObj.Code, msgObj.Message)
	}
	return "", gerror.Newf("third api http code:%d, response:%s", res.StatusCode, body)
}

func (h *HttpClient) Get(ctx context.Context, url string, data interface{}, headers map[string]string) (string, error) {
	for k, v := range headers {
		h.client.SetHeader(k, v)
	}

	res, err := h.client.Get(ctx, url, data)
	if err != nil {
		return "", err
	}
	defer res.Close()

	return parseResponse(res)
}

func (h *HttpClient) Post(ctx context.Context, url string, data interface{}, headers map[string]string) (string, error) {
	for k, v := range headers {
		h.client.SetHeader(k, v)
	}

	res, err := h.client.Post(ctx, url, data)
	if err != nil {
		return "", err
	}
	defer res.Close()

	return parseResponse(res)
}

func (h *HttpClient) Put(ctx context.Context, url string, data interface{}, headers map[string]string) (string, error) {
	for k, v := range headers {
		h.client.SetHeader(k, v)
	}

	res, err := h.client.Put(ctx, url, data)
	if err != nil {
		return "", err
	}
	defer res.Close()

	return parseResponse(res)
}

func (h *HttpClient) Patch(ctx context.Context, url string, data interface{}, headers map[string]string) (string, error) {
	for k, v := range headers {
		h.client.SetHeader(k, v)
	}

	res, err := h.client.Patch(ctx, url, data)
	if err != nil {
		return "", err
	}
	defer res.Close()

	return parseResponse(res)
}

func (h *HttpClient) Delete(ctx context.Context, url string, data interface{}, headers map[string]string) (string, error) {
	for k, v := range headers {
		h.client.SetHeader(k, v)
	}

	res, err := h.client.Delete(ctx, url, data)
	if err != nil {
		return "", err
	}
	defer res.Close()

	return parseResponse(res)
}

func (h *HttpClient) PostStream(ctx context.Context, url string, data interface{}, headers map[string]string) (io.ReadCloser, error) {
	// 将 data 转换为 io.Reader
	var body io.Reader
	switch v := data.(type) {
	case string:
		body = bytes.NewReader([]byte(v))
	case []byte:
		body = bytes.NewReader(v)
	default:
		// return nil, fmt.Errorf("unsupported data type: %T", v)
		// 结构体/map 自动序列化为JSON
		jsonData, err := gjson.Marshal(data)
		if err != nil {
			return nil, fmt.Errorf("json marshal failed: %w", err)
		}
		body = bytes.NewReader(jsonData)

		// 自动补充Content-Type（如果外部没有传入）
		if _, exist := headers["Content-Type"]; !exist {
			if headers == nil {
				headers = make(map[string]string)
			}
			headers["Content-Type"] = "application/json; charset=utf-8"
		}
	}

	// 创建 HTTP 请求
	req, err := http.NewRequestWithContext(ctx, "POST", url, body)
	if err != nil {
		return nil, err
	}

	// 设置请求头
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	// 发起请求
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}

	// 检查响应状态码
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("request failed with status code: %d", resp.StatusCode)
	}

	return resp.Body, nil
}

// PostMultipart 实现文件上传功能，支持同时上传文件和其他表单参数
func (h *HttpClient) PostMultipart(ctx context.Context, url string, fileField string, file *ghttp.UploadFile, data map[string]string, headers map[string]string) (string, error) {
	// 创建 multipart 表单
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	// 获取文件内容
	fileContent, err := file.Open()
	if err != nil {
		return "", err
	}
	defer fileContent.Close()

	// 添加文件字段
	filePart, err := writer.CreateFormFile(fileField, file.Filename)
	if err != nil {
		return "", err
	}

	// 从上传的文件复制内容到multipart表单
	_, err = io.Copy(filePart, fileContent)
	if err != nil {
		return "", err
	}

	// 添加其他表单参数
	for key, value := range data {
		err = writer.WriteField(key, value)
		if err != nil {
			return "", err
		}
	}

	// 关闭 writer 以确保 multipart 数据正确结束
	err = writer.Close()
	if err != nil {
		return "", err
	}

	// 创建 HTTP 请求
	req, err := http.NewRequestWithContext(ctx, "POST", url, body)
	if err != nil {
		return "", err
	}

	// 设置 Content-Type 头部
	req.Header.Set("Content-Type", writer.FormDataContentType())

	// 设置其他头部
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	// 发起请求
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	// 读取响应体
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	// 检查响应状态码
	if resp.StatusCode != http.StatusOK {
		return "", gerror.Newf("upload request failed with status code: %d, response: %s", resp.StatusCode, string(responseBody))
	}

	// 解析响应
	if strings.Contains(resp.Header.Get("Content-Type"), "application/json") {
		// 如果是 JSON 响应，尝试解析错误信息
		var msgObj Message
		if err := gjson.Unmarshal(responseBody, &msgObj); err == nil && msgObj.Message != "" {
			if msgObj.Code != "" {
				return "", gerror.Newf("[%s] %s", msgObj.Code, msgObj.Message)
			}
			return string(responseBody), nil
		}
	}

	return string(responseBody), nil
}
