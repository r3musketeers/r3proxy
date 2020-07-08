package proxy

type ClientHandlerFunc func([]byte) ([]byte, error)

type JoinHandlerFunc func(string, string) error

type Transporter interface {
	ListenClient(ClientHandlerFunc) error
	ListenJoin(JoinHandlerFunc) error
	Deliver([]byte) ([]byte, error)
	SendJoinRequest(string, []byte) error
}
