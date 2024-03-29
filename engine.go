package gonight

import (
	"fmt"
	//"fmt"
	"github.com/xupingao/go-easy-adapt/http"
	"github.com/xupingao/go-easy-adapt/adapters/standard"
	"github.com/xupingao/gonight/internal/bytesconv"
	"github.com/xupingao/gonight/render"
	"html/template"
	"net"
	"os"
	"path"

	//"net"
	//"os"
	//"path"

	//"path"
	"sync"
)

const defaultMultipartMemory = 32 << 20 // 32 MB

type HandlerFunc func(*Context)
type HandlersChain []HandlerFunc

func (c HandlersChain) Last() HandlerFunc {
	if length := len(c); length > 0 {
		return c[length-1]
	}
	return nil
}

type RouteInfo struct {
	Method      string
	Path        string
	Handler     string
	HandlerFunc HandlerFunc
}

type RoutesInfo []RouteInfo

type Engine struct {
	Server http.Server
	RouterGroup

	RedirectTrailingSlash  bool
	RedirectFixedPath      bool
	HandleMethodNotAllowed bool
	ForwardedByClientIP    bool
	AppEngine              bool
	UseRawPath             bool
	UnescapePathValues     bool
	MaxMultipartMemory     int64
	RemoveExtraSlash       bool

	delims           render.Delims
	secureJsonPrefix string
	HTMLRender       render.HTMLRender
	FuncMap          template.FuncMap
	allNoRoute       HandlersChain
	allNoMethod      HandlersChain
	noRoute          HandlersChain
	noMethod         HandlersChain
	pool             sync.Pool
	trees            methodTrees

	maxParams        uint16
	maxSections      uint16
}

var _ IRouter = &Engine{}

func New(server http.Server) *Engine {
	engine := &Engine{
		Server: server,
		RouterGroup: RouterGroup{
			Handlers: nil,
			basePath: "/",
			root:     true,
		},
		FuncMap:                template.FuncMap{},
		//RedirectTrailingSlash:  true,
		RedirectFixedPath:      false,
		HandleMethodNotAllowed: false,
		ForwardedByClientIP:    true,
		AppEngine:              false,
		UseRawPath:             false,
		RemoveExtraSlash:       false,
		UnescapePathValues:     true,
		MaxMultipartMemory:     defaultMultipartMemory,
		trees:                  make(methodTrees, 0, 9),
		delims:                 render.Delims{Left: "{{", Right: "}}"},
		secureJsonPrefix:       "while(1);",
	}
	engine.RouterGroup.engine = engine
	engine.pool.New = func() interface{} {
		return engine.allocateContext()
	}
	return engine
}

func NewWithDefaultServer() *Engine{
	return New(standard.NewDefaultServer())
}

// Default returns an Engine instance with the Logger and Recovery middleware already attached.
func Default() *Engine {
	engine := New(standard.NewDefaultServer())
	engine.Use(Logger(), Recovery())
	return engine
}

func (engine *Engine) allocateContext() *Context {
	v := make(Params, 0, engine.maxParams)
	skippedNodes := make([]skippedNode, 0, engine.maxSections)

	return &Context{engine: engine,params: &v, skippedNodes: &skippedNodes}
}

// Delims sets template left and right delims and returns a Engine instance.
func (engine *Engine) Delims(left, right string) *Engine {
	engine.delims = render.Delims{Left: left, Right: right}
	return engine
}

// SecureJsonPrefix sets the secureJsonPrefix used in Context.SecureJSON.
func (engine *Engine) SecureJsonPrefix(prefix string) *Engine {
	engine.secureJsonPrefix = prefix
	return engine
}

// LoadHTMLGlob loads HTML files identified by glob pattern
// and associates the result with HTML renderer.
func (engine *Engine) LoadHTMLGlob(pattern string) {
	left := engine.delims.Left
	right := engine.delims.Right
	templ := template.Must(template.New("").Delims(left, right).Funcs(engine.FuncMap).ParseGlob(pattern))

	if IsDebugging() {
		debugPrintLoadTemplate(templ)
		engine.HTMLRender = render.HTMLDebug{Glob: pattern, FuncMap: engine.FuncMap, Delims: engine.delims}
		return
	}

	engine.SetHTMLTemplate(templ)
}

// LoadHTMLFiles loads a slice of HTML files
// and associates the result with HTML renderer.
func (engine *Engine) LoadHTMLFiles(files ...string) {
	if IsDebugging() {
		engine.HTMLRender = render.HTMLDebug{Files: files, FuncMap: engine.FuncMap, Delims: engine.delims}
		return
	}

	templ := template.Must(template.New("").Delims(engine.delims.Left, engine.delims.Right).Funcs(engine.FuncMap).ParseFiles(files...))
	engine.SetHTMLTemplate(templ)
}

// SetHTMLTemplate associate a template with HTML renderer.
func (engine *Engine) SetHTMLTemplate(templ *template.Template) {
	if len(engine.trees) > 0 {
		debugPrintWARNINGSetHTMLTemplate()
	}

	engine.HTMLRender = render.HTMLProduction{Template: templ.Funcs(engine.FuncMap)}
}

// SetFuncMap sets the FuncMap used for template.FuncMap.
func (engine *Engine) SetFuncMap(funcMap template.FuncMap) {
	engine.FuncMap = funcMap
}

// NoRoute adds handlers for NoRoute. It return a 404 code by default.
func (engine *Engine) NoRoute(handlers ...HandlerFunc) {
	engine.noRoute = handlers
	engine.rebuild404Handlers()
}

// NoMethod sets the handlers called when... TODO.
func (engine *Engine) NoMethod(handlers ...HandlerFunc) {
	engine.noMethod = handlers
	engine.rebuild405Handlers()
}

// Use attaches a global middleware to the router. ie. the middleware attached though Use() will be
// included in the handlers chain for every single Request. Even 404, 405, static files...
// For example, this is the right place for a logger or error management middleware.
func (engine *Engine) Use(middleware ...HandlerFunc) IRoutes {
	engine.RouterGroup.Use(middleware...)
	engine.rebuild404Handlers()
	engine.rebuild405Handlers()
	return engine
}

func (engine *Engine) rebuild404Handlers() {
	engine.allNoRoute = engine.combineHandlers(engine.noRoute)
}

func (engine *Engine) rebuild405Handlers() {
	engine.allNoMethod = engine.combineHandlers(engine.noMethod)
}

func (engine *Engine) addRoute(method, path string, handlers HandlersChain) {
	assert1(path[0] == '/', "path must begin with '/'")
	assert1(method != "", "HTTP method can not be empty")
	assert1(len(handlers) > 0, "there must be at least one handler")

	debugPrintRoute(method, path, handlers)
	root := engine.trees.get(method)
	if root == nil {
		root = new(node)
		root.fullPath = "/"
		engine.trees = append(engine.trees, methodTree{method: method, root: root})
	}
	root.addRoute(path, handlers)

	// Update maxParams
	if paramsCount := countParams(path); paramsCount > engine.maxParams {
		engine.maxParams = paramsCount
	}

	if sectionsCount := countSections(path); sectionsCount > engine.maxSections {
		engine.maxSections = sectionsCount
	}
}

// Routes returns a slice of registered routes, including some useful information, such as:
// the http method, path and the handler name.
func (engine *Engine) Routes() (routes RoutesInfo) {
	for _, tree := range engine.trees {
		routes = iterate("", tree.method, routes, tree.root)
	}
	return routes
}

func iterate(path, method string, routes RoutesInfo, root *node) RoutesInfo {
	path += root.path
	if len(root.handlers) > 0 {
		handlerFunc := root.handlers.Last()
		routes = append(routes, RouteInfo{
			Method:      method,
			Path:        path,
			Handler:     nameOfFunction(handlerFunc),
			HandlerFunc: handlerFunc,
		})
	}
	for _, child := range root.children {
		routes = iterate(path, method, routes, child)
	}
	return routes
}

// Run attaches the router to a http.Server and starts listening and serving HTTP requests.
// It is a shortcut for http.ListenAndServe(addr, router)
// Note: this method will block the calling goroutine indefinitely unless an error happens.
func (engine *Engine) Run(addr ...string) (err error) {
	defer func() { debugPrintError(err) }()

	address := resolveAddress(addr)
	debugPrint("Listening and serving HTTP on %s\n", address)
	err = engine.Server.ListenAndServe(address, engine)
	return
}

// RunTLS attaches the router to a http.Server and starts listening and serving HTTPS (secure) requests.
// It is a shortcut for http.ListenAndServeTLS(addr, certFile, keyFile, router)
// Note: this method will block the calling goroutine indefinitely unless an error happens.
func (engine *Engine) RunTLS(addr, certFile, keyFile string) (err error) {
	debugPrint("Listening and serving HTTPS on %s\n", addr)
	defer func() { debugPrintError(err) }()


	err = engine.Server.ListenAndServeTLS(addr, certFile, keyFile, engine)
	return
}

// RunUnix attaches the router to a http.Server and starts listening and serving HTTP requests
// through the specified unix socket (ie. a file).
// Note: this method will block the calling goroutine indefinitely unless an error happens.
func (engine *Engine) RunUnix(file string) (err error) {
	debugPrint("Listening and serving HTTP on unix:/%s", file)
	defer func() { debugPrintError(err) }()

	os.Remove(file)
	listener, err := net.Listen("unix", file)
	if err != nil {
		return
	}
	defer listener.Close()
	err = os.Chmod(file, 0777)
	if err != nil {
		return
	}
	engine.Server.SetListener(listener)
	engine.Server.SetHandler(engine)
	err = engine.Server.Run()
	return
}

// RunFd attaches the router to a http.Server and starts listening and serving HTTP requests
// through the specified file descriptor.
// Note: this method will block the calling goroutine indefinitely unless an error happens.
func (engine *Engine) RunFd(fd int) (err error) {
	debugPrint("Listening and serving HTTP on fd@%d", fd)
	defer func() { debugPrintError(err) }()

	f := os.NewFile(uintptr(fd), fmt.Sprintf("fd@%d", fd))
	listener, err := net.FileListener(f)
	if err != nil {
		return
	}
	defer listener.Close()
	err = engine.RunListener(listener)
	return
}

// RunListener attaches the router to a http.Server and starts listening and serving HTTP requests
// through the specified net.Listener
func (engine *Engine) RunListener(listener net.Listener) (err error) {
	debugPrint("Listening and serving HTTP on listener what's bind with address@%s", listener.Addr())
	defer func() { debugPrintError(err) }()
	engine.Server.SetListener(listener)
	engine.Server.SetHandler(engine)
	err = engine.Server.Run()
	return
}
//
//// ServeHTTP conforms to the http.Handler interface.
//func (engine *Engine) ServeHTTP(w http.ResponseWriter, req *http.Request) {
//	c := engine.pool.Get().(*Context)
//	c.writermem.reset(w)
//	c.Request = req
//	c.reset()
//
//	engine.handleHTTPRequest(c)
//
//	engine.pool.Put(c)
//}

func (engine *Engine) ServeHTTP(httpCtx http.Context) {
	c := engine.pool.Get().(*Context)
	c.httpCtx = httpCtx
	c.Request = httpCtx.Request()
	c.Response = httpCtx.Response()
	if v, ok := httpCtx.Response().(http.HTTP2Response); ok {
		c.http2Response = v
	}

	c.reset()
	engine.handleHTTPRequest(c)
	engine.pool.Put(c)
}

// HandleContext re-enter a context that has been rewritten.
// This can be done by setting c.Request.URL.Path to your new target.
// Disclaimer: You can loop yourself to death with this, use wisely.
func (engine *Engine) HandleContext(c *Context) {
	oldIndexValue := c.index
	c.reset()
	engine.handleHTTPRequest(c)
	c.index = oldIndexValue
}

func (engine *Engine) handleHTTPRequest(c *Context) {
	httpMethod := c.Method()
	rPath := c.RequestURL().Path()
	unescape := false
	if engine.UseRawPath && len(c.RequestURL().RawPath()) > 0 {
		rPath = c.RequestURL().RawPath()
		unescape = engine.UnescapePathValues
	}

	if engine.RemoveExtraSlash {
		rPath = cleanPath(rPath)
	}

	// Find root of the tree for the given HTTP method
	t := engine.trees
	for i, tl := 0, len(t); i < tl; i++ {
		if t[i].method != httpMethod {
			continue
		}
		root := t[i].root
		// Find route in tree
		value := root.getValue(rPath, c.params, c.skippedNodes,unescape)
		if value.params != nil {
			c.Params = *value.params
		}
		if value.handlers != nil {
			c.handlers = value.handlers
			c.fullPath = value.fullPath
			c.Next()
			c.Response.WriteHeaderNow()
			return
		}
		if httpMethod != "CONNECT" && rPath != "/" {
			if value.tsr && engine.RedirectTrailingSlash {
				redirectTrailingSlash(c)
				return
			}
			if engine.RedirectFixedPath && redirectFixedPath(c, root, engine.RedirectFixedPath) {
				return
			}
		}
		break
	}

	if engine.HandleMethodNotAllowed {
		for _, tree := range engine.trees {
			if tree.method == httpMethod {
				continue
			}
			if value := tree.root.getValue(rPath, nil, c.skippedNodes, unescape); value.handlers != nil {
				c.handlers = engine.allNoMethod
				serveError(c, http.StatusMethodNotAllowed, []byte(http.StatusText(405)))
				return
			}
		}
	}
	c.handlers = engine.allNoRoute
	serveError(c, http.StatusNotFound, []byte(http.StatusText(404)))
}


func serveError(c *Context, code int, defaultMessage []byte) {
	c.Response.WriteHeader(code)
	c.Next()
	if c.Response.HeaderWritten() {
		return
	}
	if c.Response.Status() == code {
		c.Response.Header().Set(http.HeaderContentType,http.MIMETextPlain)
		_, err := c.Response.Write(defaultMessage)
		if err != nil {
			debugPrint("cannot write message to writer during serve error: %v", err)
		}
		return
	}
	c.Response.WriteHeaderNow()
}

func redirectTrailingSlash(c *Context) {
	req := c.Request
	p := req.URL().Path()
	if prefix := path.Clean(c.Request.Header().Get("X-Forwarded-Prefix")); prefix != "." {
		p = prefix + "/" + req.URL().Path()
	}
	req.URL().SetPath(p + "/")
	if length := len(p); length > 1 && p[length-1] == '/' {
		req.URL().SetPath(p[:length-1])
	}
	redirectRequest(c)
}

func redirectFixedPath(c *Context, root *node, trailingSlash bool) bool {
	req := c.Request
	rPath := req.URL().Path()

	if fixedPath, ok := root.findCaseInsensitivePath(cleanPath(rPath), trailingSlash); ok {
		req.URL().SetPath(bytesconv.BytesToString(fixedPath))
		redirectRequest(c)
		return true
	}
	return false
}

func redirectRequest(c *Context) {
	req := c.Request
	rPath := req.URL().Path()
	rURL := req.URL().String()
	code := http.StatusMovedPermanently // Permanent redirect, Request with GET method
	if c.Method() != http.MethodGet {
		code = http.StatusTemporaryRedirect
	}
	debugPrint("redirecting Request %d: %s --> %s", code, rPath, rURL)

	c.httpCtx.Redirect(code, rURL)
	c.Response.WriteHeaderNow()
}
