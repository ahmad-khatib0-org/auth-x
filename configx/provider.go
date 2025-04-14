package configx

const (
	FlagConfig = "config"
	Delimiter  = "."
)

type Provider struct{}

type tuple struct {
	Key   string
	Value interface{}
}
