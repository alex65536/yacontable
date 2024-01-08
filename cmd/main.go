package main

import (
	"context"
	"net/http"

	"github.com/alex65536/yacontable/internal"
	"go.uber.org/zap"
	"golang.org/x/crypto/acme/autocert"
	"github.com/klauspost/compress/gzhttp"
)

func setupServers(conf *internal.Config) {
	go func() {
		err := http.ListenAndServe(conf.ListenAddr, http.DefaultServeMux)
		if err != nil {
			panic(err)
		}
	}()
	if conf.SecureListenAddr != "" {
		m := &autocert.Manager{
			Prompt:     autocert.AcceptTOS,
			HostPolicy: autocert.HostWhitelist(conf.AllowedSecureDomains...),
			Cache:      autocert.DirCache("secrets/certs"),
		}
		server := &http.Server{
			Addr:      conf.SecureListenAddr,
			TLSConfig: m.TLSConfig(),
		}
		go func() {
			err := server.ListenAndServeTLS("", "")
			if err != nil {
				panic(err)
			}
		}()
	}
}

func main() {
	logger, err := zap.NewDevelopment()
	if err != nil {
		panic(err)
	}
	conf, err := internal.LoadConfig()
	if err != nil {
		panic(err)
	}
	sec, err := internal.LoadStaticSecrets()
	if err != nil {
		panic(err)
	}

	setupServers(conf)

	api, err := internal.NewApi(logger, context.Background(), conf, sec)
	if err != nil {
		panic(err)
	}

	keep := internal.NewKeeper(conf, api)

	pres, err := internal.NewPresenter(context.Background(), logger, keep, conf)
	if err != nil {
		panic(err)
	}

	http.Handle("/", gzhttp.GzipHandler(pres))
	http.Handle("/style.css", gzhttp.GzipHandler(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		http.ServeFile(w, req, "./data/style.css")
	})))

	<-make(chan struct{})
}
