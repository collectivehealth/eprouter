<!-- CH Repo Tracking Data version 1.0.2 -->
|     Category     |         Value         |
| ---------------: | --------------------- |
| **Repo Status**  | deprecated |
| **Language**     | Go          |
| **Repo Owner**   |   |
<!-- If you use any of the following fields, please delete "Repo Owner", above,
     as it then becomes ambiguous.  If you add fields, please update the script
     at walker-github-ranger. -->
<!--
| **Repo PoC**     | <One or more humans>  |
| **Repo Oncall**  | <Link to PagerDuty>   |
| **Repo Team Owner** | <Name of owning team> |
-->


eprouter
========

Designed for quickly making REST APIs, w/ easy config and just enough magic to keep you productive, written in Go
Spiritual Successor to [amattn/grunway](https://github.com/amattn/grunway)


Installation
------------

	go get github.com/collectivehealth/eprouter


Usage
-----

eprouter is built around 2 central concepts:

1. Defining high-level routes
2. Quickly implementing handlers

Under the hood, routes map to endpoints which call handlers.
refelection is used during startup to build an internal routing map, but not during routing of requests.

### Defining High-level Routes

Looks like this:

	routerPtr := eprouter.NewRouter()
	routerPtr.BasePath = "/api/"
	routerPtr.RegisterEntity("book", &BookController{})
	routerPtr.RegisterEntity("author", &AuthorController{})
	return routerPtr

You then start the server like so:

	host := ":8090"
	err := eprouter.Start(routerPtr, host)
	if err != nil {
		log.Fatalln(err)
	}

### Quickly implementing handlers

Here is where we use a bit of reflection.  Instead of defining routes and hooking up controllers, we _just_ immplement handlers.

Like this:

	func (bc *BookController) GetHandlerV1(ctx *eprouter.Context) eprouter.RouteHandlerResult {
	   //...
	}
	func (bc *BookController) GetHandlerV2(ctx *eprouter.Context) eprouter.RouteHandlerResult {
		//...
	}
	func (bc *BookController) GetHandlerV1Popular(ctx *eprouter.Context) eprouter.RouteHandlerResult {
	    //...
	}

The basic function signature is a `RouteHandler`:

	type RouteHandler func(*Context) RouteHandlerResult

99% of the time you either return a PayloadsMap or a RouteError.  If you need special control of the response, a CustomRouteResponse is a special handler with more access to the output stream.


The format of the Handlers works like this:

	<Method>HandlerV<version><Action>

Which corresponts to http endpoints like this:

    <Method> http://host/<prefix>/v<version>/<Entity>/<optionalID>/<Action>

### Auth

Auth is a bit special that it has its own dedicated handler prefix:

	func (wc *WidgetController) AuthGetHandlerV1(ctx *eprouter.Context) eprouter.RouteHandlerResult {
	   //...
	}

There is a example package for handling auth at http://github.com/amattn/grwacct

