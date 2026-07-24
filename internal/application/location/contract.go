package location

type ILinkProvider interface {
	GetLink(location string) (string, error)
}
