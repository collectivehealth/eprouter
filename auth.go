package eprouter

type AuthHandler interface {
	PerformAuth(routePtr *Route, ctx *Context) (authenticationWasSucessful bool, failureToAuthErrorNum int)
}
