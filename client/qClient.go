package client

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/quic-go/quic-go"
	"github.com/quic-go/quic-go/http3"
)

const (
	HttpsScheme      = "https"
	HttpScheme       = "http"
	DefaultHttpsPort = "443"
	DefaultHttpPort  = "80"

	DefaultScheme = HttpsScheme
	DefaultPort   = DefaultHttpsPort

	ContentTypeHeaderKey = "content-type"
)

type requestStruct struct {
	URL     *url.URL
	method  string
	header  http.Header
	body    io.ReadCloser
	timeout time.Duration
	retry   int
	uuid    string
}

type responseStruct struct {
	statusCode int
	header     http.Header
	iobody     io.ReadCloser
	err        error
	req        *requestStruct
	contentLen int64
}

func ReqCreate(uri, method, domain string) *requestStruct {
	u, _ := url.ParseRequestURI(fmt.Sprintf("%s://%s%s", DefaultScheme, domain, uri))
	return &requestStruct{
		URL:     u,
		method:  method,
		header:  make(http.Header),
		body:    nil,
		timeout: time.Second * 60,
		retry:   0,
		uuid:    "",
	}
}

func ReqCreateFullURL(URL, method string) *requestStruct {
	u, _ := url.ParseRequestURI(URL)

	return &requestStruct{
		URL:     u,
		method:  method,
		header:  make(http.Header),
		body:    nil,
		timeout: time.Second * 60,
		retry:   0,
		uuid:    "",
	}
}

func (request *requestStruct) SchemeUrlSetHTTP() *requestStruct {
	request.URL.Scheme = HttpScheme
	return request
}

func (request *requestStruct) UUIDSet(uuid string) *requestStruct {
	request.uuid = uuid
	return request
}

func (request *requestStruct) SchemeUrlSetHTTPS() *requestStruct {
	request.URL.Scheme = HttpsScheme
	return request
}

func (request *requestStruct) Url() *url.URL {

	return request.URL
}

func (request *requestStruct) RetrySet(retry int) *requestStruct {
	request.retry = retry
	return request
}

func (request *requestStruct) TimeoutSet(time time.Duration) *requestStruct {
	request.timeout = time
	return request
}

func (request *requestStruct) ByteBodySet(body []byte) *requestStruct {

	request.body = io.NopCloser(bytes.NewReader(body))
	return request
}

func (request *requestStruct) IoBodySet(body io.ReadCloser) *requestStruct {

	request.body = body
	return request
}

func (request *requestStruct) ModelBodySet(model interface{}) (*requestStruct, error) {
	body, err := json.Marshal(model)
	request.ByteBodySet(body)
	return request, err
}

func (request *requestStruct) HeaderSet(key, val string) *requestStruct {
	request.header.Set(key, val)
	return request
}

func (request *requestStruct) AllHeaderSet(headers http.Header) *requestStruct {
	request.header = headers
	return request
}

func (request *requestStruct) URIQuerySet(key, val string) *requestStruct {
	values := request.URL.Query()
	values.Set(key, val)
	request.URL.RawQuery = values.Encode()
	return request
}

func (request *requestStruct) PostFormSet(values url.Values) *requestStruct {
	request.ByteBodySet([]byte(values.Encode()))
	request.HeaderSet("Content-Type", "application/x-www-form-urlencoded")

	return request
}

func (request *requestStruct) Send() *responseStruct {
	return request.SendWithContext(context.Background())
}

func (request *requestStruct) SendWithContext(ctx context.Context) *responseStruct {

	rt := &http3.RoundTripper{

		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true,
		},
		QuicConfig: &quic.Config{
			EnableDatagrams: true,
		},
		EnableDatagrams: true,
	}

	defer rt.Close()

	client := &http.Client{
		Transport: rt,
		Timeout:   request.timeout,
		CheckRedirect: func(req *http.Request, via []*http.Request) error { //disable redirect follow
			return http.ErrUseLastResponse
		},
	}

	req := &http.Request{
		Method: request.method,
		URL:    request.URL,
		Header: request.header,
		Body:   request.body,
	}

	var (
		res *http.Response
		err error
	)
	for i := 0; i <= request.retry; i++ {
		res, err = client.Do(req.WithContext(ctx))
		if err != nil {
			if ctx.Err() == nil {
				continue
			}
		}
		break
	}

	if err != nil {
		return &responseStruct{
			err: err,
		}
	}

	return &responseStruct{
		statusCode: res.StatusCode,
		header:     res.Header,
		iobody:     res.Body,
		err:        err,
		req:        request,
		contentLen: res.ContentLength,
	}

}

func (response *responseStruct) ErrGet() error {
	return response.err
}

func (response *responseStruct) ByteBodyGet() ([]byte, error) {
	return ReadCloser(response.iobody)
}

func (response *responseStruct) IoBodyGet() io.ReadCloser {
	return response.iobody
}

func (response *responseStruct) ModelBodyGet(model interface{}) error {
	body, err := response.ByteBodyGet()
	if err != nil {
		return err
	}
	return json.Unmarshal(body, model)
}

func (response *responseStruct) AllHeadersGet() http.Header {
	return response.header
}

func (response *responseStruct) AllHeadersGetMap() map[string]string {
	m := make(map[string]string)
	for k, v := range response.header {
		m[k] = v[0]
	}
	return m
}

func (response *responseStruct) HeaderGet(key string) (val string) {
	return response.header.Get(key)
}

func (response *responseStruct) StatusIsOk() bool {
	return response.statusCode == http.StatusOK
}

func (response *responseStruct) StatusCodeGet() int {
	return response.statusCode
}

func (response *responseStruct) ReqGet() *requestStruct {
	return response.req
}

func (response *responseStruct) ContentLenGet() int64 {
	return response.contentLen
}

func (response *responseStruct) ContentTypeGet() string {
	return response.HeaderGet(ContentTypeHeaderKey)
}

var bufferPool2 = sync.Pool{
	New: func() interface{} {
		buffer := make([]byte, 512) // Adjust the buffer size as per your needs
		return &buffer
	},
}

func ReadCloser(iobody io.ReadCloser) ([]byte, error) {
	buffer := bufferPool2.Get().(*[]byte)
	defer bufferPool2.Put(buffer)

	var bodyBuffer []byte

	for {

		tempLen, err := iobody.Read((*buffer)[:cap(*buffer)])
		if err != nil {
			if err == io.EOF {
				bodyBuffer = append(bodyBuffer, (*buffer)[:tempLen]...)
				break
			}
			return nil, err
		}
		bodyBuffer = append(bodyBuffer, (*buffer)[:tempLen]...)

	}

	return bodyBuffer, nil
}
