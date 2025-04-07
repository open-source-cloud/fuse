package workflow

type Edge interface {
	ID() string
	DataMapping() DataMapping
}
