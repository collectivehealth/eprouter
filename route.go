package eprouter

import (
	"log"
	"regexp"
	"strconv"
	"strings"
)

const (
	MAGIC_AUTH_REQUIRED_PREFIX  = "Auth"
	MAGIC_HANDLER_KEYWORD       = "Handler"
	MAGIC_GET_HANDLER_PREFIX    = "GetHandler"    // CRUD: read
	MAGIC_POST_HANDLER_PREFIX   = "PostHandler"   // CRUD: create
	MAGIC_PUT_HANDLER_PREFIX    = "PutHandler"    // CRUD: update (the whole thing)
	MAGIC_PATCH_HANDLER_PREFIX  = "PatchHandler"  // CRUD: update (just a field or two)
	MAGIC_DELETE_HANDLER_PREFIX = "DeleteHandler" // CRUD: delete (duh)
	MAGIC_HEAD_HANDLER_PREFIX   = "HeadHandler"   // usually when you just want to check Etags or something.
)

const VERSION_BIT_DEPTH = 16

type VersionUint uint16

type Route struct {
	RequiresAuth  bool
	Authenticator AuthHandler

	Method         string
	Path           string
	VersionStr     string
	EntityName     string
	Action         string
	Handler        RouteHandler
	HandlerName    string // not actually used except for logging and debugging
	ControllerName string // not actually used except for logging and debugging

	part     string
	Children []*Route
}

// A convenience method to get all the route description in the following format:
//
//     [AUTH] GET /path/../  <Controller>.<Action>
func (n *Route) AllRoutesDescription() []string {
	result := make([]string, 0, 5)
	if n.Handler != nil {
		msg := n.Method + " " + n.Path + " → " + n.ControllerName + "." + n.Action
		// If Auth is required add AUTH to the head of the message
		if n.RequiresAuth {
			msg = "AUTH " + msg
		}
		result = append(result, msg)
	}
	for _, c := range n.Children {
		result = append(result, c.AllRoutesDescription()...)
	}
	return result
}

func (n *Route) addNode(parts []string, handler RouteHandler, authHandler AuthHandler) *Route {
	if len(parts) == 0 {
		n.Handler = handler
		n.Authenticator = authHandler
		return n
	} else if parts[0] == "" {
		return n.addNode(parts[1:], handler, authHandler)
	} else if n.Children == nil {
		n.Children = make([]*Route, 0, 5)
	}

	part, rest := parts[0], parts[1:]

	for _, c := range n.Children {
		if c.part == part {
			return c.addNode(rest, handler, authHandler)
		}
	}

	c := new(Route)
	c.part = part
	n.Children = append(n.Children, c)
	return c.addNode(rest, handler, authHandler)
}

func (n *Route) resolveRoute(parts []string) *Route {
	if len(parts) == 0 {
		return n
	}

	part, rest := parts[0], parts[1:]
	if part == "" {
		return n.resolveRoute(rest)
	}

	for _, child := range n.Children {
		key := child.part
		if key == "*" {
			return child.resolveRoute(rest)
		} else if strings.HasPrefix(key, ":") {
			return child.resolveRoute(rest)
		} else if key == part {
			return child.resolveRoute(rest)
		}
	}
	return nil
}

func parseVersionFromPrefixlessHandlerName(versionActionHandlerName string) (vStr string, action string) {
	re := regexp.MustCompile("^V([0-9]+)(.*)")
	matches := re.FindStringSubmatch(versionActionHandlerName)

	if len(matches) < 3 {
		log.Println("2457509067 Failed to parse V<#><Action> for", versionActionHandlerName)
		return "", ""
	}

	vStr = matches[1]
	vStr = strings.TrimLeft(vStr, "0")
	_, err := strconv.ParseUint(vStr, 10, VERSION_BIT_DEPTH)
	if err != nil {
		return "", ""
	}

	action = matches[2]
	action = strings.ToLower(action)

	return
}

// Validation

func ValidateEntityName(name string) (isValid bool, reason string) {
	if len(name) < 1 {
		return false, "name must have at least one character"
	}
	// TODO: check for valid url chars: [a-Z 0-9 _ -]
	return true, ""
}
func ValidateHandlerName(handler interface{}) (isValid bool, reason string) {
	// TODO length? not much here really.
	return true, ""
}
func ValidateHandler(unknownHandler interface{}) (isValid bool, reason string, handler RouteHandler) {

	// We have to do some type gymnastics here.  first check the if the method matches the raw function type...
	validHandler, ok := unknownHandler.(func(*Context) RouteHandlerResult)

	if ok == false {
		return false, "wrong function type, expected function type of RouteHandler", nil
	}

	// ...then convert the raw funtion type to the typed RouteHandler
	handler = validHandler
	return true, "", handler
}
