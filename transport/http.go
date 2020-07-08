package transport

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
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

////////////////////////////////////////////////////////////////////////////////
//
// r3proxy.Transporter interface implementation
//
////////////////////////////////////////////////////////////////////////////////

func (t *HTTPTransport) ListenClient(handle r3model.ClientHandlerFunc) error {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		reqBuffer := bytes.NewBuffer([]byte{})
		err := r.Write(reqBuffer)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		respData, err := handle(reqBuffer.Bytes())
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		respReader := bytes.NewReader(respData)
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
	})
	return http.ListenAndServe(t.listenClientAddr, nil)
}

func (t *HTTPTransport) ListenJoin(join r3model.JoinHandlerFunc) error {
	joinMux := http.NewServeMux()
	joinMux.HandleFunc("/join", func(w http.ResponseWriter, r *http.Request) {
		log.Println("received join request")
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
		log.Println("NODE ID", nodeID)
		joinAddr, ok := joinReq["addr"]
		if !ok {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		err = join(nodeID, joinAddr)
		if err != nil {
			log.Println(err.Error())
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	})
	return http.ListenAndServe(t.listenJoinAddr, joinMux)
}

func (t *HTTPTransport) Deliver(reqData []byte) ([]byte, error) {
	reqReader := bytes.NewReader(reqData)
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
	respBuffer := bytes.NewBuffer([]byte{})
	err = resp.Write(respBuffer)
	if err != nil {
		return nil, err
	}
	return respBuffer.Bytes(), nil
}

func (t *HTTPTransport) SendJoinRequest(joinAddr string, joinBody []byte) error {
	resp, err := http.Post(
		fmt.Sprintf("http://%s/join", joinAddr),
		"application-type/json",
		bytes.NewReader(joinBody),
	)
	if err != nil {
		return err
	}
	resp.Body.Close()
	return nil
}
