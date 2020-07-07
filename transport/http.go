package transport

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"

	r3model "r3proxy/model"
)

type HTTPTransport struct {
	listenClientAddr string
	listenJoinAddr   string
	deliverAddr      string
}

func NewHTTPTransport(listenClientAddr, listenJoinAddr, deliverAddr string) *HTTPTransport {
	return &HTTPTransport{
		listenClientAddr: listenClientAddr,
		listenJoinAddr:   listenJoinAddr,
		deliverAddr:      deliverAddr,
	}
}

func (t *HTTPTransport) ListenRequest(handle r3model.RequestHandlerFunc) error {
	http.HandleFunc("/", t.handler(handle))
	return http.ListenAndServe(t.listenClientAddr, nil)
}

func (t *HTTPTransport) ListenJoin(join r3model.JoinHandlerFunc) error {
	joinMux := http.NewServeMux()
	joinMux.HandleFunc("/join", func(w http.ResponseWriter, r *http.Request) {
		joinReq := map[string]string{}
		err := json.NewDecoder(r.Body).Decode(&joinReq)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		nodeID, ok := joinReq["id"]
		if !ok {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		joinAddr, ok := joinReq["addr"]
		if !ok {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		err = join(nodeID, joinAddr)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	})
	return http.ListenAndServe(t.listenJoinAddr, joinMux)
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

func (t *HTTPTransport) handler(handle r3model.RequestHandlerFunc) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		reqBuffer := bytes.NewBuffer([]byte{})
		err := r.Write(reqBuffer)
		log.Println(string(reqBuffer.Bytes()))
		if err != nil {
			log.Println(err.Error())
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		log.Println("handling by proxy")
		respReader, err := handle(reqBuffer.Bytes())
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
