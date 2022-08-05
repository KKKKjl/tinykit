package registry

var (
	DefaultRegistry Registry
)

type Registry interface {
	Register(*Service) error

	GetService(string) ([]*Service, error)

	ListServices() ([]*Service, error)
}

type Builder interface {
	GetService() ([]*Service, error)

	PutServer(*Service)

	ListServer() []*Service

	DelServer(*Service)

	Scheme() string
}

type Service struct {
	Name          string            `json:"name"`
	Version       string            `json:"version"`
	Addr          string            `json:"addr"`
	Weight        int               `json:"weight"`
	CurrentWeight int               `json:"current_weight"`
	Metadata      map[string]string `json:"metadata"`
}
