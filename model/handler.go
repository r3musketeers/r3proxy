package model

import "io"

type RequestHandlerFunc func([]byte) (io.Reader, error)

type JoinHandlerFunc func(string, string) error
