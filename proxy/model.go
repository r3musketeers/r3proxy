package proxy

import "io"

type HandlerFunc func(io.Reader) (io.Reader, error)

type R3Message struct {
	ID     string
	Reader io.Reader
}
