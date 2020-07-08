package model

type HistoryGeneratorFunc func() map[string][]byte

type HistoryPopulatorFunc func(map[string][]byte)
