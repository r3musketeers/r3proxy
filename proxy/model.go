package proxy

import "io"

type R3Message struct {
	ID string
	Reader io.Reader
}