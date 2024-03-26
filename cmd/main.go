package main

import (
	"context"
	"io"
	"net/http"

	"github.com/alex65536/yacontable/internal"
	"github.com/klauspost/compress/gzhttp"
	"go.uber.org/zap"
	"golang.org/x/crypto/acme/autocert"
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
	logger, err := zap.NewProduction()
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

	keep, err := internal.NewKeeper(conf, api)
	if err != nil {
		panic(err)
	}

	pres, err := internal.NewPresenter(context.Background(), logger, keep, conf)
	if err != nil {
		panic(err)
	}

	http.Handle("/", gzhttp.GzipHandler(pres))
	http.Handle("/style.css", gzhttp.GzipHandler(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		http.ServeFile(w, req, "./data/style.css")
	})))
	http.HandleFunc("/robots.txt", func(w http.ResponseWriter, req *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = io.WriteString(w, "not found")
	})
	http.Handle("/favicon.ico", gzhttp.GzipHandler(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		http.ServeFile(w, req, "./data/favicon.ico")
	})))

	<-make(chan struct{})
}
