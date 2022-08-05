package server

import (
	"context"
	"net/http"
	"net/http/pprof"
	"os"
	"os/signal"
	"syscall"

	"github.com/KKKKjl/tinykit/config"
	"github.com/KKKKjl/tinykit/internal/proxy"
	"github.com/KKKKjl/tinykit/internal/registry/balance"
)

type Server interface {
	Start(context.Context)
	Stop()
}

func prof(stop <-chan struct{}) {
	pprofServeMux := http.NewServeMux()
	pprofServeMux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
	pprofServeMux.HandleFunc("/debug/pprof/profile", pprof.Profile)
	pprofServeMux.HandleFunc("/debug/pprof/", pprof.Index)
	pprofServeMux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)

	server := &http.Server{
		Addr:    ":9090",
		Handler: pprofServeMux,
	}

	go func() {
		<-stop
		mainLog.Debug("退出")
		server.Shutdown(context.Background())
	}()

	server.ListenAndServe()
}

func Start() {
	mainLog.Info("Starting TinyKit.")

	config.InitConfig()

	proxy := proxy.New(proxy.ProxyConfig{
		URLRewriteEnabled:    false,
		LoadBalancingEnabled: true,
		ForceTlsEnabled:      false,
		BalancingType:        balance.ROUND_ROBIN,
	})

	done := make(chan struct{})

	// channel to receive os signal
	waiter := make(chan os.Signal, 1)
	signal.Notify(waiter, syscall.SIGQUIT, syscall.SIGTERM, syscall.SIGINT)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	gateway := New(proxy, WithFilters("ratelimit"))
	gateway.Start(ctx)

	// for debug
	go prof(done)

	select {
	case sig := <-waiter:
		mainLog.Infof("Received signal %v, exiting.", sig)
		close(done)
		return
	}
}
