package local

const (
	env     = "local"
	address = "localhost:8080"
)

type Cfg struct {
	Env     string
	Address string
}

func New() *Cfg {
	return &Cfg{
		Env:     env,
		Address: address,
	}
}
