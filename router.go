package eprouter

import (
	"fmt"
	"log"
	"net/http"
	"reflect"
	"runtime"
	"strings"

	"github.com/amattn/deeperror/levels"
)

const (
	NotFoundPrefix                            = "404 Not Found"
	NotFoundErrorNumber                       = 4040000404
	BadRequestPrefix                          = "400 Bad Request"
	BadRequestErrorNumber                     = 4000000000
	BadRequestSyntaxErrorPrefix               = BadRequestPrefix + ": Syntax Error"
	BadRequestSyntaxErrorErrorNumber          = 4000000001
	BadRequestMissingPrimaryKeyErrorNumber    = 4000000002
	BadRequestExtraneousPrimaryKeyErrorNumber = 4000000003

	InternalServerErrorPrefix = "500 Internal Server Error"
)

type PayloadController interface {
}

type ModifiablePayloadController interface {
	RegisterRoutes(router *Router)
	Version() string
}

type Router struct {
	BasePath string

	PreProcessors        []PreProcessor
	MiddlewareProcessors []MiddlewareProcessor
	PostProcessors       []PostProcessor

	Controllers map[string]PayloadController // key is entity name
	RouteMap    map[string]*Route            // key is http method
	entityName  string                       // Temporary internal variable
	controller  PayloadController            // Temporary internal variable
}

func NewRouter() *Router {
	router := new(Router)

	router.Controllers = make(map[string]PayloadController)
	router.RouteMap = make(map[string]*Route)

	router.PreProcessors = []PreProcessor{}
	router.MiddlewareProcessors = []MiddlewareProcessor{}
	router.PostProcessors = []PostProcessor{
		new(CommonLogger),
	}
	return router
}

//  #####
// #     #  ####  #    # ###### #  ####
// #       #    # ##   # #      # #    #
// #       #    # # #  # #####  # #
// #       #    # #  # # #      # #  ###
// #     # #    # #   ## #      # #    #
//  #####   ####  #    # #      #  ####
//

// Configuration of Router

func (router *Router) registerRoute(method, path string, handler RouteHandler, auth AuthHandler) *Route {
	node := router.RouteMap[method]
	if node == nil {
		node = new(Route)
		router.RouteMap[method] = node
	}
	parts := strings.Split(path, "/")
	added := node.addNode(parts, handler, auth)
	added.Action = runtime.FuncForPC(reflect.ValueOf(handler).Pointer()).Name()
	added.EntityName = router.entityName
	added.ControllerName = reflect.TypeOf(router.controller).String()
	added.Path = path
	added.Method = method
	return added
}

func (router *Router) ResolveRoute(method, path string) *Route {
	node := router.RouteMap[method]
	if node == nil {
		return nil
	}
	parts := strings.Split(path, "/")
	return node.resolveRoute(parts)
}

func (router *Router) RegisterRoute(method, version, path string, handler RouteHandler, authHandler AuthHandler) *Route {
	actualPath := "/" + version + "/" + router.entityName + path
	route := router.registerRoute(method, actualPath, handler, authHandler)
	if authHandler != nil {
		route.RequiresAuth = true
	}
	return route
}

func (router *Router) RegisterModifiableEntity(name string, payloadController ModifiablePayloadController) {
	// Setup temp variables
	router.entityName = name
	router.controller = payloadController

	payloadController.RegisterRoutes(router)
	router.RegisterEntity(name, payloadController)

	// Cleanup temp variables
	router.entityName = ""
	router.controller = nil
}

func (router *Router) RegisterEntity(name string, payloadController PayloadController) {
	payloadControllerType := reflect.TypeOf(payloadController)
	payloadControllerValue := reflect.ValueOf(payloadController)

	if isValid, reason := ValidateEntityName(name); isValid == false {
		log.Fatalln("Invalid Enitity name:'", name, "'", reason)
	}
	if payloadController == nil {
		log.Fatalln("untypedHandlerWrapper currently must not be nil")
	}

	router.Controllers[name] = payloadController
	router.controller = payloadController

	authenticator, _ := payloadController.(AuthHandler)

	for i := 0; i < payloadControllerType.NumMethod(); i++ {

		potentialHandlerMethod := payloadControllerType.Method(i)
		potentialHandlerName := potentialHandlerMethod.Name
		if len(potentialHandlerName) > 0 && potentialHandlerName[0] == strings.ToUpper(potentialHandlerName)[0] {
			// skip unexported methods
			unknownhandler := payloadControllerValue.MethodByName(potentialHandlerName).Interface()
			router.AddEntityRoute(name, payloadControllerType.String(), potentialHandlerName, unknownhandler, authenticator)
		}
	}
}

func (router *Router) AddEntityRoute(entityName, controllerName, handlerName string, unknownhandler interface{}, authenticator AuthHandler) {
	// simple first:
	if strings.Contains(handlerName, MAGIC_HANDLER_KEYWORD) == false {
		// just skip it
		return
	}

	isValid, reason, handler := ValidateHandler(unknownhandler)
	if isValid == false {
		errNum := int64(3230075622)
		errMsg := fmt.Sprintln(errNum, "Handler Validation Failure:", "entityName:", entityName, "controllerName:", controllerName, "Invalid Handler:", handlerName, "reason:", reason)
		// derr := deeperror.New(errNum, errMsg, nil)
		log.Println(errMsg)
		// skip... invalid prefix
		return
	}

	routePtr := new(Route)
	routePtr.Path = entityName + "/"
	routePtr.EntityName = entityName
	routePtr.Handler = handler
	routePtr.HandlerName = handlerName
	routePtr.ControllerName = controllerName

	// Step 1 Check for Auth prrefix
	deauthedHandlerName := handlerName
	if strings.HasPrefix(handlerName, MAGIC_AUTH_REQUIRED_PREFIX) {
		deauthedHandlerName = handlerName[len(MAGIC_AUTH_REQUIRED_PREFIX):]
		routePtr.RequiresAuth = true
		if authenticator == nil {
			log.Fatalf("1323798307 Auth required handler defined (%s), but controller (%s) does not implement AuthHandler", handlerName, controllerName)
			return
		}
		routePtr.Authenticator = authenticator
	}

	// step 2 Find method
	var versionActionHandlerName string
	switch {
	case strings.HasPrefix(deauthedHandlerName, MAGIC_GET_HANDLER_PREFIX):
		routePtr.Method = "GET"
		versionActionHandlerName = deauthedHandlerName[len(MAGIC_GET_HANDLER_PREFIX):]
	case strings.HasPrefix(deauthedHandlerName, MAGIC_POST_HANDLER_PREFIX):
		routePtr.Method = "POST"
		versionActionHandlerName = deauthedHandlerName[len(MAGIC_POST_HANDLER_PREFIX):]
	case strings.HasPrefix(deauthedHandlerName, MAGIC_PUT_HANDLER_PREFIX):
		routePtr.Method = "PUT"
		versionActionHandlerName = deauthedHandlerName[len(MAGIC_PUT_HANDLER_PREFIX):]
	case strings.HasPrefix(deauthedHandlerName, MAGIC_DELETE_HANDLER_PREFIX):
		routePtr.Method = "DELETE"
		versionActionHandlerName = deauthedHandlerName[len(MAGIC_DELETE_HANDLER_PREFIX):]
	case strings.HasPrefix(deauthedHandlerName, MAGIC_PATCH_HANDLER_PREFIX):
		routePtr.Method = "PATCH"
		versionActionHandlerName = deauthedHandlerName[len(MAGIC_PATCH_HANDLER_PREFIX):]
	case strings.HasPrefix(deauthedHandlerName, MAGIC_HEAD_HANDLER_PREFIX):
		routePtr.Method = "HEAD"
		versionActionHandlerName = deauthedHandlerName[len(MAGIC_HEAD_HANDLER_PREFIX):]
	default:
		// skip... it's not a known prefix
		log.Println("1860816435 Skipping Route:", entityName, controllerName, handlerName)
		return
	}

	// do a bit of primite parsing:

	if isValid, reason := ValidateHandlerName(handler); isValid == false {
		log.Fatalln("1411397818 entity name:", routePtr.EntityName, "method:", routePtr.Method, "route:", routePtr.Path, "Invalid Handler:", handlerName, "reason:", reason)
	}

	// log.Println("versionActionHandlerName", versionActionHandlerName)

	versionStr, action := parseVersionFromPrefixlessHandlerName(versionActionHandlerName)
	if versionStr == "" {
		log.Println("1259486570 Skipping Route:", entityName, controllerName, handlerName)
		// skip... invalid prefix
		return
	}

	// log.Println("version, action", version, action)
	routePtr.Action = action
	routePtr.Path += action
	routePtr.VersionStr = versionStr

	path := "/v" + routePtr.VersionStr + "/" + routePtr.EntityName
	if routePtr.Action != "" {
		path = path + "/" + routePtr.Action
	}
	if routePtr.RequiresAuth {
		authenticator = routePtr.Authenticator
	}
	result := router.registerRoute(routePtr.Method, path, routePtr.Handler, authenticator)
	result.Action = handlerName

	path = "/v" + routePtr.VersionStr + "/" + routePtr.EntityName + "/:id"
	if routePtr.Action != "" {
		path = path + "/" + routePtr.Action
	}
	if routePtr.RequiresAuth {
		authenticator = routePtr.Authenticator
	}
	result = router.registerRoute(routePtr.Method, path, routePtr.Handler, authenticator)
	result.Action = handlerName
}

// Convenience method
func (router *Router) AllRoutesCount() int {
	data := router.AllRoutesDescription()
	return len(data)
}

// Basically just used for logging and debugging.
// the first addon is a prefix, all remaining addons are treated as suffixes and appended to the end
func (router *Router) AllRoutesDescription(addons ...string) []string {
	result := make([]string, 0, 100)
	for _, route := range router.RouteMap {
		result = append(result, route.AllRoutesDescription()...)
	}
	return result
}

// Basically just used for logging and debugging.
// the first addon is a prefix, all remaining addons are treated as suffixes and appended to the end
func (router *Router) AllRoutesSummary(addons ...string) string {
	lines := router.AllRoutesDescription(addons...)
	lines = append(lines, "") // basically, append an newline at the end.
	return strings.Join(lines, "\n")
}

func (router *Router) LogAllRoutes(addons ...string) {
	lines := router.AllRoutesDescription(addons...)
	for _, line := range lines {
		log.Println(line)
	}
}

//  #####                              #     # ####### ####### ######
// #     # ###### #####  #    # ###### #     #    #       #    #     #
// #       #      #    # #    # #      #     #    #       #    #     #
//  #####  #####  #    # #    # #####  #######    #       #    ######
//       # #      #####  #    # #      #     #    #       #    #
// #     # #      #   #   #  #  #      #     #    #       #    #
//  #####  ###### #    #   ##   ###### #     #    #       #    #
//

// ServeHTTP does the basics:
// 1. Any pre-handler stuff
// 2. parse the route
// 3. lookup route
// 4. validate/auth route
// 5. Auth (if necessary)
// 6. Middleware
// 7. call handler method
// 8. any post processors

// Note on steps 1-8:
// - post processors are always called, even if

func (router *Router) ServeHTTP(w http.ResponseWriter, req *http.Request) {

	ctx := new(Context) // needs a leakybucket
	ctx.w = w
	ctx.Req = req
	ctx.router = router

	// we use defer so our post processors are ALWAYS called.
	defer func() {
		// 8. any post-handler stuff
		for _, postproc := range ctx.router.PostProcessors {
			terminateEarly, derr := postproc.Process(ctx)
			if derr != nil {
				log.Println(derr)
			}
			if terminateEarly {
				return
			}
		}
	}()

	// 1. Any pre-handler stuff
	for _, preproc := range ctx.router.PreProcessors {
		terminateEarly, derr := preproc.Process(ctx)
		if derr != nil {
			log.Println(derr)
		}
		if terminateEarly {
			return
		}
	}

	// 2. parse the route
	endpoint, clientDeepErr, serverDeepErr := parsePath(req.URL, router.BasePath)
	ctx.Endpoint = endpoint

	if clientDeepErr != nil {
		// log.Println("clientDeepErr", clientDeepErr)
		code := http.StatusBadRequest
		if clientDeepErr.StatusCode > 299 && clientDeepErr.StatusCode < 999 {
			code = clientDeepErr.StatusCode
		}

		if clientDeepErr.StatusCode == 400 {
			ctx.SendSimpleErrorPayload(code, clientDeepErr.Num, BadRequestPrefix)
		} else if clientDeepErr.StatusCode == 404 {
			ctx.SendSimpleErrorPayload(code, clientDeepErr.Num, NotFoundPrefix)
		}

		return
	}

	if serverDeepErr != nil {
		log.Println("serverDeepErr", serverDeepErr)
		code := http.StatusInternalServerError
		if serverDeepErr.StatusCode > 299 && serverDeepErr.StatusCode < 999 {
			code = serverDeepErr.StatusCode
		}
		ctx.SendSimpleErrorPayload(code, serverDeepErr.Num, InternalServerErrorPrefix)
		return
	}

	router.handleContext(ctx, req)
}

func (router *Router) handleContext(ctx *Context, req *http.Request) {
	// 3. lookup the handler method
	routePtr := router.ResolveRoute(req.Method, req.URL.Path[len(router.BasePath):])
	if routePtr == nil || routePtr.Handler == nil {
		ctx.SendSimpleErrorPayload(http.StatusNotFound, NotFoundErrorNumber, "404 Not Found")
		return
	}

	// 4. validate/auth route
	// TODO Nothing to do

	// 5. Auth
	if routePtr.RequiresAuth {
		// log.Println("RequiresAuth = true")
		isAuthorized, failureToAuthErrorNum, failureToAuthErrorMessage := routePtr.Authenticator.PerformAuth(routePtr, ctx)
		if isAuthorized == false {
			ctx.SendSimpleErrorPayload(http.StatusUnauthorized, int64(failureToAuthErrorNum), failureToAuthErrorMessage)
			return
		}
	}

	// 6. Middleware

	for _, middleware := range router.MiddlewareProcessors {
		middleware.Process(routePtr, ctx)
	}

	// 7. call handler method
	routeHandlerResult := routePtr.Handler(ctx)
	if routeHandlerResult.rerr != nil {
		rtErr := routeHandlerResult.rerr
		if rtErr.ErrorLevel == levels.Undefined {
			rtErr.ErrorLevel = levels.Error
		}
		log.Printf("%v %+v", rtErr.ErrorLevel, rtErr)
		ctx.SendErrorInfoPayload(rtErr.statusCode, rtErr.errorInfo)
	} else if routeHandlerResult.pmap != nil {
		ctx.WrapAndSendPayloadsMap(routeHandlerResult.pmap)
	} else if routeHandlerResult.crr != nil {
		routeHandlerResult.crr(ctx)
	} else {
		ctx.SendSimpleErrorPayload(http.StatusInternalServerError, 2302586595, "Invalid Handler response")
	}
}
