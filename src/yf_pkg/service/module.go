package service

type Module interface {
	Init(env *Env) (err error)
}
