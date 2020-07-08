package model

type ClientHandlerFunc func([]byte) ([]byte, error)

type JoinHandlerFunc func(string, string) error
