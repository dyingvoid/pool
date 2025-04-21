package pool

type status uint8

const (
	opening status = iota + 1
	established
	failed
)