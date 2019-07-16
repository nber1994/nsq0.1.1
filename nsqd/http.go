package main

import (
	"bytes"
	"io"
	"log"
	"net"
	"net/http"
	"nsq"
	"strings"
	"util"
)

import _ "net/http/pprof"

func HttpServer(listener net.Listener) {
	log.Printf("HTTP: listening on %s", listener.Addr().String())
	handler := http.NewServeMux()
	//测试连通性
	handler.HandleFunc("/ping", pingHandler)
	//投递消息
	handler.HandleFunc("/put", putHandler)
	//批量投递消息
	handler.HandleFunc("/mput", mputHandler)
	//获取状态
	handler.HandleFunc("/stats", statsHandler)
	//情况topic
	handler.HandleFunc("/empty", emptyHandler)
	server := &http.Server{Handler: handler}
	err := server.Serve(listener)
	// theres no direct way to detect this error because it is not exposed
	if err != nil && !strings.Contains(err.Error(), "use of closed network connection") {
		log.Printf("ERROR: http.Serve() - %s", err.Error())
	}
}

//直接返回ok
func pingHandler(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Content-Length", "2")
	io.WriteString(w, "OK")
}

//向topic投递消息
func putHandler(w http.ResponseWriter, req *http.Request) {
	reqParams, err := util.NewReqParams(req)
	if err != nil {
		log.Printf("ERROR: failed to parse request params - %s", err.Error())
		w.Write(util.ApiResponse(500, "INVALID_REQUEST", nil))
		return
	}

	topicName, err := reqParams.Query("topic")
	if err != nil {
		w.Write(util.ApiResponse(500, "MISSING_ARG_TOPIC", nil))
		return
	}

	if len(topicName) > nsq.MaxNameLength {
		w.Write(util.ApiResponse(500, "INVALID_ARG_TOPIC", nil))
		return
	}

	topic := nsqd.GetTopic(topicName)
	//该处会向id生成器获取一个id来为message作为唯一标识
	msg := nsq.NewMessage(<-nsqd.idChan, reqParams.Body)
	topic.PutMessage(msg)

	w.Header().Set("Content-Length", "2")
	io.WriteString(w, "OK")
}

func mputHandler(w http.ResponseWriter, req *http.Request) {
	reqParams, err := util.NewReqParams(req)
	if err != nil {
		log.Printf("ERROR: failed to parse request params - %s", err.Error())
		w.Write(util.ApiResponse(500, "INVALID_REQUEST", nil))
		return
	}

	topicName, err := reqParams.Query("topic")
	if err != nil {
		w.Write(util.ApiResponse(500, "MISSING_ARG_TOPIC", nil))
		return
	}

	if len(topicName) > nsq.MaxNameLength {
		w.Write(util.ApiResponse(500, "INVALID_ARG_TOPIC", nil))
		return
	}

	topic := nsqd.GetTopic(topicName)
	//只能批量向一个topic批量投递
	for _, block := range bytes.Split(reqParams.Body, []byte("\n")) {
		if len(block) != 0 {
			msg := nsq.NewMessage(<-nsqd.idChan, block)
			topic.PutMessage(msg)
		}
	}

	w.Header().Set("Content-Length", "2")
	io.WriteString(w, "OK")
}

//清空channel
func emptyHandler(w http.ResponseWriter, req *http.Request) {
	reqParams, err := util.NewReqParams(req)
	if err != nil {
		log.Printf("ERROR: failed to parse request params - %s", err.Error())
		w.Write(util.ApiResponse(500, "INVALID_REQUEST", nil))
		return
	}

	topicName, err := reqParams.Query("topic")
	if err != nil {
		w.Write(util.ApiResponse(500, "MISSING_ARG_TOPIC", nil))
		return
	}

	if len(topicName) > nsq.MaxNameLength {
		w.Write(util.ApiResponse(500, "INVALID_ARG_TOPIC", nil))
		return
	}

	channelName, err := reqParams.Query("channel")
	if err != nil {
		w.Write(util.ApiResponse(500, "MISSING_ARG_CHANNEL", nil))
		return
	}

	if len(topicName) > nsq.MaxNameLength {
		w.Write(util.ApiResponse(500, "INVALID_ARG_CHANNEL", nil))
		return
	}

	topic := nsqd.GetTopic(topicName)
	channel := topic.GetChannel(channelName)
	//清空channel
	err = EmptyQueue(channel)
	if err != nil {
		w.Write(util.ApiResponse(500, "INTERNAL_ERROR", nil))
		return
	}

	w.Header().Set("Content-Length", "2")
	io.WriteString(w, "OK")
}
