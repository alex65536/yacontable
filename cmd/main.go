package main

import (
	"context"
	"net/http"

	"github.com/alex65536/yacontable/internal"
	"go.uber.org/zap"
)

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

	done := make(chan struct{}, 1)
	go func() {
		err := http.ListenAndServe(conf.ListenAddr, http.DefaultServeMux)
		if err != nil {
			panic(err)
		}
		done <- struct{}{}
	}()

	api, err := internal.NewApi(logger, context.Background(), conf, sec)
	if err != nil {
		panic(err)
	}

	keep := internal.NewKeeper(conf, api)

	pres, err := internal.NewPresenter(context.Background(), logger, keep, conf)
	if err != nil {
		panic(err)
	}

	http.Handle("/", pres)
	http.HandleFunc("/style.css", func(w http.ResponseWriter, req *http.Request) {
		http.ServeFile(w, req, "./data/style.css")
	})

	<-done
}
