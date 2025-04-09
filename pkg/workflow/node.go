package workflow

const DefaultOutputSchema = "default"

type Node interface {
	ID() string
	Params() Params
	Execute() (interface{}, error)
}
