package pipelines

type Pipeline interface {
	Init()
	Write()
}
