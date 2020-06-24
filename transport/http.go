package transport

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"

	"r3-proxy/proxy"
)

type HTTPTransport struct {
	listenAddr  string
	deliverAddr string
}

func NewHTTPTransport(listenAddr, deliverAddr string) *HTTPTransport {
	return &HTTPTransport{
		listenAddr:  listenAddr,
		deliverAddr: deliverAddr,
	}
}

func (t *HTTPTransport) Listen(handle proxy.HandlerFunc) error {
	http.HandleFunc("/", t.handler(handle))
	return http.ListenAndServe(t.listenAddr, nil)
}

func (t *HTTPTransport) Deliver(reqReader io.Reader) (io.Reader, error) {
	req, err := http.ReadRequest(bufio.NewReader(reqReader))
	if err != nil {
		return nil, err
	}
	url, err := url.Parse(fmt.Sprintf("http://localhost%s%s", t.deliverAddr, req.RequestURI))
	if err != nil {
		return nil, err
	}
	req.RequestURI = ""
	req.URL = url
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	log.Println("request sent")
	respBuffer := bytes.NewBuffer([]byte{})
	err = resp.Write(respBuffer)
	log.Println("response received")
	log.Println(string(respBuffer.Bytes()))
	if err != nil {
		return nil, err
	}
	return respBuffer, nil
}

func (t *HTTPTransport) handler(handle proxy.HandlerFunc) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		reqBuffer := bytes.NewBuffer([]byte{})
		err := r.Write(reqBuffer)
		if err != nil {
			log.Println(err.Error())
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		log.Println("handling by proxy")
		respReader, err := handle(reqBuffer)
		log.Println("response received")
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		resp, err := http.ReadResponse(bufio.NewReader(respReader), r)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		err = resp.Write(w)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	}
}
