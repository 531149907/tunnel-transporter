package constants

type AuthenticationType string

const (
	None        AuthenticationType = "none"
	StaticToken AuthenticationType = "static-token"
	Certificate AuthenticationType = "certificate"
)
