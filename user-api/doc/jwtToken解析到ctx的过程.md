### jwt 验证中 toekn 解析过程

* 设置需要 jwt 验证，在注册路由时，需要传递对应的 RouteOption 函数
```Go
server.AddRoutes(
    []rest.Route{
        {
            Method:  http.MethodGet,
            Path:    "/user/info",
            Handler: account.GetUserInfoHandler(serverCtx),
        },
    },
    rest.WithJwt(serverCtx.Config.Auth.AccessSecret), // jwt 设置函数
    rest.WithPrefix("/v1"),  
)

func WithJwt(secret string) RouteOption {
	return func(r *featuredRoutes) {
		validateSecret(secret)
		r.jwt.enabled = true  // 设置为有效
		r.jwt.secret = secret // 设置密钥 key
	}
}
```

* 在路由管理中谈到，在绑定路由的过程中，会添加中间件作为前置执行函数，链式地执行中间件函数，最后才执行请求处理函数
```Go
// 绑定路由: chain 的链式调用, 包裹中间件函数
func (ng *engine) bindRoute(fr featuredRoutes, router httpx.Router, metrics *stat.Metrics,
	route Route, verifier func(chain.Chain) chain.Chain) error {
	chn := ng.chain
	if chn == nil {
		chn = ng.buildChainWithNativeMiddlewares(fr, route, metrics) // 初始化原生中间件
	}

    // 在这里添加验证中间件
	chn = ng.appendAuthHandler(fr, chn, verifier) 

	for _, middleware := range ng.middlewares {
		chn = chn.Append(convertMiddleware(middleware))  // 包裹中间件函数
	}
	handle := chn.ThenFunc(route.Handler) // 最后包裹路由函数

	return router.Handle(route.Method, route.Path, handle) // 添加到路由路径对应的搜索树中
}
```

* 在添加验证 jwt 中间件时添加验证函数到 chain 中
```Go
func (ng *engine) appendAuthHandler(fr featuredRoutes, chn chain.Chain,
	verifier func(chain.Chain) chain.Chain) chain.Chain {
	if fr.jwt.enabled {
		if len(fr.jwt.prevSecret) == 0 {
            // 添加验证函数到 chain 中
			chn = chn.Append(handler.Authorize(fr.jwt.secret,
				handler.WithUnauthorizedCallback(ng.unauthorizedCallback)))
		} else {
			chn = chn.Append(handler.Authorize(fr.jwt.secret,
				handler.WithPrevSecret(fr.jwt.prevSecret),
				handler.WithUnauthorizedCallback(ng.unauthorizedCallback)))
		}
	}

	return verifier(chn)
}
```

* 在验证函数中解析 token 
```Go
// Authorize returns an authorization middleware.
// (1) 验证中间件
func Authorize(secret string, opts ...AuthorizeOption) func(http.Handler) http.Handler {
	var authOpts AuthorizeOptions
	for _, opt := range opts {
		opt(&authOpts)
	}

	parser := token.NewTokenParser()
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            // (2) 解析 token
			tok, err := parser.ParseToken(r, secret, authOpts.PrevSecret)
            /* ... */	
			claims, ok := tok.Claims.(jwt.MapClaims)
            /* ... */

			ctx := r.Context()
			for k, v := range claims {
				switch k {
				case jwtAudience, jwtExpire, jwtId, jwtIssueAt, jwtIssuer, jwtNotBefore, jwtSubject:
					// ignore the standard claims
				default:
                    // 把每个 k v 都加到 ctx 中, 是能够解析出来
					ctx = context.WithValue(ctx, k, v)
				}
			}

            // 调用下一个函数
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// (2) 处理和解析 token 
// ParseToken parses token from given r, with passed in secret and prevSecret.
func (tp *TokenParser) ParseToken(r *http.Request, secret, prevSecret string) (*jwt.Token, error) {
	var token *jwt.Token
	var err error

	if len(prevSecret) > 0 {
        /*...*/
		token, err = tp.doParseToken(r, first)
        /*...*/
	} else {
        /*...*/
        // 调用 (3) ，根据请求解析 token 
		token, err = tp.doParseToken(r, secret) 
        /*...*/
	}

	return token, nil
}
// (3) 根据请求解析 token 
func ParseFromRequest(req *http.Request, extractor Extractor, keyFunc jwt.Keyfunc, options ...ParseFromRequestOption) (token *jwt.Token, err error) {
	// Create basic parser struct
	p := &fromRequestParser{req, extractor, nil, nil}
    /*...*/
	// Set defaults
	if p.claims == nil {
		p.claims = jwt.MapClaims{}
	}
	if p.parser == nil {
		p.parser = &jwt.Parser{}
	}
    /*...*/

	// perform parse
    // 调用 (4), 分段解析
	return p.parser.ParseWithClaims(tokenString, p.claims, keyFunc)
}

// (4) 分段解析
func (p *Parser) ParseWithClaims(tokenString string, claims Claims, keyFunc Keyfunc) (*Token, error) {
    // 解析 parts[0] parts[1], 如 (5)
	token, parts, err := p.ParseUnverified(tokenString, claims)
    /*...*/
	
    // Perform validation
    // 解析 parts[2]
	token.Signature = parts[2]
	if err := token.Method.Verify(strings.Join(parts[0:2], "."), token.Signature, key); err != nil {
		return token, &ValidationError{Inner: err, Errors: ValidationErrorSignatureInvalid}
	}
    /*...*/

	// No errors so far, token is valid.
	token.Valid = true

	return token, nil
}

// (5) 解析 parts[0][1]
func (p *Parser) ParseUnverified(tokenString string, claims Claims) (token *Token, parts []string, err error) {
    // 分段
    parts = strings.Split(tokenString, ".")
    /*...*/
	token = &Token{Raw: tokenString}
	// parse Header: 解析 parts[0]
	var headerBytes []byte
	if headerBytes, err = DecodeSegment(parts[0]); err != nil {
		/*...*/
	}
	if err = json.Unmarshal(headerBytes, &token.Header); err != nil {
		return token, parts, &ValidationError{Inner: err, Errors: ValidationErrorMalformed}
	}

	// parse Claims: 解析 parts[1]
	var claimBytes []byte
	token.Claims = claims

	if claimBytes, err = DecodeSegment(parts[1]); err != nil {
		return token, parts, &ValidationError{Inner: err, Errors: ValidationErrorMalformed}
	}
	dec := json.NewDecoder(bytes.NewBuffer(claimBytes))
	if p.UseJSONNumber {
		dec.UseNumber()
	}
	// JSON Decode.  Special case for map type to avoid weird pointer behavior
	if c, ok := token.Claims.(MapClaims); ok {
		err = dec.Decode(&c)
	} else {
		err = dec.Decode(&claims)
	}
	/*...*/
	return token, parts, nil
}
```

* 从 ctx 读取对应的 key 的 val 的巧妙设计
```Go
// 新的 context
type valueCtx struct {
	Context
	key, val any
}
// 存储k-v: context 包中的包裹设计, 若有多个参数则有多层
func WithValue(parent Context, key, val any) Context {
	if parent == nil {
		panic("cannot create context from nil parent")
	}
	if key == nil {
		panic("nil key")
	}
	if !reflectlite.TypeOf(key).Comparable() {
		panic("key is not comparable")
	}
	return &valueCtx{parent, key, val}  // 返回一个新的 ctx
}

// 读取k: 递归地读取
func (c *valueCtx) Value(key any) any {
	if c.key == key {
		return c.val
	}
	return value(c.Context, key)
}

// 使用 for 循环不断地查找
/* 层次结构示意图
ctx3 -> valueCtx{parent: ctx2, key: "isPremium", val: true}
   |
ctx2 -> valueCtx{parent: ctx1, key: "role", val: "admin"}
   |
ctx1 -> valueCtx{parent: ctx, key: "userID", val: 123}
   |
ctx  -> (background context)
*/ 
func value(c Context, key any) any {
	for {
		switch ctx := c.(type) {
		case *valueCtx:
			if key == ctx.key {
				return ctx.val
			}
			c = ctx.Context
		case *cancelCtx:
			if key == &cancelCtxKey {
				return c
			}
			c = ctx.Context
		case withoutCancelCtx:
			if key == &cancelCtxKey {
				// This implements Cause(ctx) == nil
				// when ctx is created using WithoutCancel.
				return nil
			}
			c = ctx.c
		case *timerCtx:
			if key == &cancelCtxKey {
				return &ctx.cancelCtx
			}
			c = ctx.Context
		case backgroundCtx, todoCtx:
			return nil
		default:
			return c.Value(key)
		}
	}
}
```