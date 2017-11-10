package sessions

type Provider interface {
	SessionInit(id string) (Session, error)
	SessionRead(id string) (Session, error)
	SessionDestroy(id string) error
	SessionGC(maxLifetime int64)
	SessionUpdate(id string) error
}
