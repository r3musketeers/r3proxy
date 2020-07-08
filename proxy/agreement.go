package proxy

type HistoryGeneratorFunc func() map[string][]byte

type HistoryPopulatorFunc func(map[string][]byte)

type AgreementAdapter interface {
	NodeID() string
	Address() string
	SetHistoryGenerator(HistoryGeneratorFunc)
	SetHistoryPopulator(HistoryPopulatorFunc)
	Join(string, string) error
	Process(R3Message) error
	Ordered() <-chan R3Message
}
