package proxy

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strconv"
	"strings"

	"github.com/KKKKjl/tinykit/internal/registry"
	"github.com/KKKKjl/tinykit/internal/registry/balance"
	"github.com/KKKKjl/tinykit/internal/registry/etcd"
	"github.com/KKKKjl/tinykit/internal/request"
	"github.com/KKKKjl/tinykit/internal/response"
	"github.com/KKKKjl/tinykit/internal/rewrite"
	"github.com/KKKKjl/tinykit/internal/server/ws"
	"github.com/KKKKjl/tinykit/internal/transform"
	"github.com/KKKKjl/tinykit/logger"
	"github.com/KKKKjl/tinykit/utils"
	"github.com/gorilla/websocket"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	tx "github.com/KKKKjl/tinykit/internal/context"
)

var (
	defaultUserAgent = "TinyKit/" + os.Getenv("TINYKIT_VERSION")

	log     = logger.GetLogger()
	mainLog = log.WithField("prefix", "proxy")
)

type ProxyConfig struct {
	URLRewriteEnabled    bool
	LoadBalancingEnabled bool
	ForceTlsEnabled      bool
	BalancingType        balance.BalanceType
}

type Proxy struct {
	reverseProxy *httputil.ReverseProxy
	proxyConfig  ProxyConfig
	ReWrite      *rewrite.ReWrite
	balancer     balance.Picker
	parser       *transform.ApiDefinitionParser
	builder      registry.Builder
	ws           *ws.WsHanlder
}

func New(proxyConfig ProxyConfig, opts ...ProxyOption) *Proxy {
	proxy := &Proxy{
		reverseProxy: nil,
		proxyConfig:  proxyConfig,
		balancer:     balance.NewBalancer(proxyConfig.BalancingType),
		ReWrite:      rewrite.NewReWrite(),
		parser:       new(transform.ApiDefinitionParser),
		builder:      etcd.Builder(),
		ws:           ws.NewWsHanlder(),
	}

	for _, opt := range opts {
		opt(proxy)
	}

	proxy.reverseProxy = &httputil.ReverseProxy{
		Director:       proxy.createDirector(),
		ModifyResponse: proxy.createModifyResponse(),
	}

	return proxy
}

// ServeHttp is an HTTP Handler that takes an incoming request and sends it to another server, proxying the response back to the client.
func (p *Proxy) ServeHTTP(ctx tx.HttpContext) {
	if matched := p.parser.IsMatchTransformRule(ctx); !matched {
		p.reverseProxy.ServeHTTP(ctx.ResponseWriter, ctx.Request)
		return
	}

	message, err := p.parser.TransformToRPC(ctx)
	if err != nil {
		mainLog.Errorf("Transform to RPC error: %v", err)
		ctx.AbortWithMsg(err.Error())
		return
	}

	target, err := p.nextTarget(ctx.Request)
	if err != nil {
		ctx.AbortWithMsg(err.Error())
		return
	}

	newCtx, cancel := context.WithCancel(ctx.Request.Context())
	defer cancel()

	conn, err := grpc.Dial(target.Host, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		mainLog.Errorf("Cannot connect to %s err: %v", target.Host, err)
		ctx.AbortWithMsg(err.Error())
		return
	}
	defer conn.Close()

	// call grpc request
	client := request.NewRPCClient(newCtx, conn)
	resp, err := client.Call(newCtx, message)
	if err != nil {
		ctx.AbortWithMsg(err.Error())
		return
	}

	var (
		wsConn   *websocket.Conn
		errs     error
		upgrader websocket.Upgrader
	)

	if resp.IsStream {
		upgrader = websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			CheckOrigin: func(r *http.Request) bool {
				return true
			},
		}

		// Upgrade initial request to a WebSocket connection.
		wsConn, errs = upgrader.Upgrade(ctx.ResponseWriter, ctx.Request, nil)
		if errs != nil {
			ctx.Error(errs)
			mainLog.Errorf("[PROXY] Upgrade error: %v", err)
			return
		}
		defer wsConn.Close()
	}

	for {
		select {
		case err := <-resp.Done:
			{
				if err != nil {
					log.Println("[PROXY] RPC server error: ", err)
					ctx.Error(err)
					break
				}
			}
		case data, ok := <-resp.DataChan:
			{
				if !ok {
					if !resp.IsStream {
						return
					}

					// channel closed, close the WebSocket connection if the response is a stream.
					if error := wsConn.WriteMessage(websocket.CloseMessage, []byte{}); error != nil {
						mainLog.Errorf("[PROXY] Write close message error: %v", error)
					}
					return
				}

				mainLog.Infof("received data: %s", string(data))

				if !resp.IsStream {
					headers := p.parser.ToHeaders(resp.RespHeader)

					ctx.SetResponseHeaders(headers)
					ctx.ToJSON(data)
					return
				}

				// messageType, _, err := wsConn.ReadMessage()
				// if err != nil {
				// 	mainLog.Errorf("[PROXY] ReadMessage error: %v", err)
				// 	return
				// }

				if err := wsConn.WriteMessage(websocket.TextMessage, data); err != nil {
					mainLog.Errorf("[PROXY] WriteMessage error: %v", err)
					return
				}
			}
		case <-newCtx.Done():
			{
				mainLog.Debug("[PROXY] rpc request Timeout")
				ctx.AbortWithMsg("Rpc request timeout.")
				return
			}
		default:
		}
	}
}

func (p *Proxy) createDirector() func(req *http.Request) {
	return func(req *http.Request) {
		var targetToUse *url.URL
		var err error

		config := p.proxyConfig
		target := req.URL

		switch {
		case config.URLRewriteEnabled:
			{
				if rule := p.ReWrite.Match(req.URL.Path); rule != nil {
					targetToUse, err = rule.ReWrite(*req)
					if err != nil {
						mainLog.Errorf("[PROXY] Url rewrite error: %v", err)
					}
				}
			}

		case config.LoadBalancingEnabled:
			{
				targetToUse, err = p.nextTarget(req)
				if err != nil {
					mainLog.Errorf("[PROXY] Get backend target error: %s", err)
				}
			}

		default:
			{
				targetToUse = req.URL
			}
		}

		// TODO err handle
		if err != nil {
			return
		}

		if targetToUse != target {
			req.URL.Scheme = targetToUse.Scheme
			req.URL.Host = targetToUse.Host

			if config.URLRewriteEnabled {
				req.URL.Path = targetToUse.Path
			} else {
				req.URL.Path = singleJoiningSlash(targetToUse.Path, req.URL.Path)
			}

			if req.URL.RawPath != "" {
				req.URL.RawPath = singleJoiningSlash(targetToUse.Path, req.URL.RawPath)
			}
		}

		targetQuery := targetToUse.RawQuery
		if targetQuery == "" || req.URL.RawQuery == "" {
			req.URL.RawQuery = targetQuery + req.URL.RawQuery
		} else {
			req.URL.RawQuery = targetQuery + "&" + req.URL.RawQuery
		}

		mainLog.Debugf("Upstream Path: %s", req.URL.String())

		// Set default User-Agent
		if _, ok := req.Header["User-Agent"]; !ok {
			req.Header.Set("User-Agent", defaultUserAgent)
		}

		// Set origin ip address
		req.Header.Set("X-Real-IP", req.RemoteAddr)

		switch req.URL.Scheme {
		// case "ws":
		// 	req.URL.Scheme = "http"
		// case "wss":
		// 	req.URL.Scheme = "https"
		case "http":
			// Enforce https on proxied http requests.
			if config.ForceTlsEnabled {
				req.URL.Scheme = "https"
			}
		}
	}
}

// TODO Response modify
func (p *Proxy) createModifyResponse() func(*http.Response) error {
	return func(resp *http.Response) error {
		if isOkResponse(resp) {
			return nil
		}

		if resp.ContentLength > 0 && resp.Body != nil && strings.ToLower(resp.Header.Get("Content-Type")) == "application/json" {
			buf, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				return err
			}
			defer resp.Body.Close()

			body := json.NewDecoder(ioutil.NopCloser(bytes.NewBuffer(buf)))
			body.DisallowUnknownFields()

			var model response.ResponseModel
			if err := body.Decode(&model); err != nil {
				mainLog.Errorf("[PROXY] Decode response body error:%v", resp.Status)
				return nil
			}

			mainLog.Warnf("[PROXY] Unnormal response default code=%d, msg=%s", model.Code, model.Message)

			resp.Body = ioutil.NopCloser(bytes.NewBuffer(buf))
			resp.ContentLength = int64(len(buf))
			resp.Header.Set("Content-Length", fmt.Sprint(len(buf)))
		}
		return nil
	}
}

// nextTarget returns the next available target from the load balance.
func (p *Proxy) nextTarget(req *http.Request) (*url.URL, error) {
	endPoint := req.Header.Get("X-TinyKit-EndPoint")
	if endPoint != "" {
		mainLog.Infof("Get remote end point from header %s", endPoint)
		return url.Parse(endPoint)
	}

	services, err := p.builder.GetService()
	if err != nil {
		mainLog.Errorf("Fail to get service from discovery(%s): %v", p.builder.Scheme(), err)
		return nil, err
	}

	ip, err := utils.GetIPAddr(req)
	if err != nil {
		mainLog.Errorf("Fail to get ip addr from req: %v", err)
		return nil, err
	}

	service, err := p.balancer.Pick(ip, services)
	if err != nil {
		mainLog.Errorf("Fail to get service from balance(%s): %v", p.balancer.Scheme(), err)
		return nil, err
	}

	mainLog.Infof("Got Service: Name(%s) Addr(%s) Weight(%d)", service.Name, service.Addr, service.Weight)
	return url.Parse(service.Addr)
}

// isOkResponse check either the response status code is ok or not.
func isOkResponse(resp *http.Response) bool {
	return resp.StatusCode >= 200 && resp.StatusCode < 300
}

func singleJoiningSlash(a, b string) string {
	if len(b) == 0 {
		return a
	}

	a = strings.TrimRight(a, "/")
	b = strings.TrimLeft(b, "/")
	if len(b) > 0 {
		return a + "/" + b
	}
	return a
}

func defaultErrorHandler(w http.ResponseWriter, msg string, statusCode int) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")

	w.WriteHeader(statusCode)
	w.Write([]byte(`{"code": ` + strconv.Itoa(statusCode) + `, "message": "` + msg + `"}`))
}
