package server

import (
	"context"
	"errors"
	"net"
	"net/http"
	"strconv"
	"time"

	tx "github.com/KKKKjl/tinykit/internal/context"
	"github.com/KKKKjl/tinykit/internal/filter"
	"github.com/KKKKjl/tinykit/internal/proxy"
	"github.com/KKKKjl/tinykit/internal/server/ws"
	"github.com/KKKKjl/tinykit/logger"
	"github.com/google/uuid"
	"github.com/spf13/viper"
)

var (
	log     = logger.GetLogger()
	mainLog = log.WithField("prefix", "main")

	_defaultAddr string
)

var (
	RpcRequestTimeoutErr = errors.New("rpc request timeout")
)

type GatewayServer struct {
	Server     *http.Server
	Timeout    time.Duration
	ApiPath    string
	TlsEnabled bool
	proxy      *proxy.Proxy
	chains     *filter.FilterChains
	wsHandler  *ws.WsHanlder
}

func init() {
	port := viper.GetString("HTTP_PORT")
	if port != "" {
		_defaultAddr = net.JoinHostPort("0.0.0.0", port)
	} else {
		_defaultAddr = "0.0.0.0:8080"
	}
}

func New(proxy *proxy.Proxy, opts ...Option) *GatewayServer {
	gatewayServer := &GatewayServer{
		Timeout:   5 * time.Second,
		ApiPath:   "/",
		proxy:     proxy,
		wsHandler: ws.NewWsHanlder(),
	}

	for _, opt := range opts {
		opt(gatewayServer)
	}

	return gatewayServer
}

func (g *GatewayServer) Start(ctx context.Context) {
	mainLog.Infof("Start http server at addr: %s", _defaultAddr)

	mux := http.NewServeMux()
	mux.HandleFunc(g.ApiPath, g.dispatch)
	mux.Handle("/ws", g.wsHandler)
	mux.HandleFunc("/ping", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("pong"))
	})

	g.Server = &http.Server{
		Addr:    _defaultAddr,
		Handler: mux,
	}

	go func() {
		defer g.Stop()

		<-ctx.Done()
		mainLog.Infof("Shutting down the http server gracefully.")
	}()

	go g.runServer()
}

func (g *GatewayServer) runServer() {
	defer func() {
		if err := recover(); err != nil {
			mainLog.Fatalf("Recover from error: %v", err)
		}
	}()

	var err error
	if g.TlsEnabled {
		err = g.Server.ListenAndServeTLS("server.crt", "server.key")
	} else {
		err = g.Server.ListenAndServe()
	}

	if err != nil && err != http.ErrServerClosed {
		mainLog.Fatalf("ListenAndServe error: %v", err)
	}
}

func (g *GatewayServer) Stop() {
	if err := g.Server.Shutdown(context.Background()); err != nil {
		mainLog.Errorf("Failed to shutdown http server: %v", err)
	}
	mainLog.Debug("Shutdown the http server gracefully.")
}

// TODO router suppport
func (g *GatewayServer) dispatch(w http.ResponseWriter, r *http.Request) {
	// create newable context
	ctx := tx.New(w, r)

	// execute filter chain and proxy request
	g.chains.Compose()(ctx, g.proxy.ServeHTTP)
}

// default error handler
func defaultErrorHandler(w http.ResponseWriter, msg string, statusCode int) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")

	w.WriteHeader(statusCode)
	w.Write([]byte(`{"code": ` + strconv.Itoa(statusCode) + `, "message": "` + msg + `"}`))
}

func PreHandle(next http.HandlerFunc) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		uid, err := uuid.NewRandom()
		if err != nil {
			mainLog.Errorf("Failed to generate uid: %v", err)
		}

		// add trace id to request
		ctx := context.WithValue(r.Context(), "trace_id", uid.String())
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
