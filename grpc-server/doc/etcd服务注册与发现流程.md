### etcd 服务端服务注册  
go-zero 中的服务注册较简单。

* (1) 在 NewServer 时根据是否有 etcd, 创建不同server实例。
```Go
func NewServer(c RpcServerConf, register internal.RegisterFn) (*RpcServer, error) {
    // ...

	if c.HasEtcd() {
		server, err = internal.NewRpcPubServer(c.Etcd, c.ListenOn, serverOptions...)
		if err != nil {
			return nil, err
		}
	} else {
		server = internal.NewRpcServer(c.ListenOn, serverOptions...)
	}

    //...
}

```

* (2) 在 start 时注册服务。
```Go
server := keepAliveServer{
    registerEtcd: registerEtcd,
    Server:       NewRpcServer(listenOn, opts...),
}

func (s keepAliveServer) Start(fn RegisterFn) error {
	if err := s.registerEtcd(); err != nil {
		return err
	}

	return s.Server.Start(fn)
}

registerEtcd := func() error {
    pubListenOn := figureOutListenOn(listenOn)
    var pubOpts []discov.PubOption
    if etcd.HasAccount() {
        pubOpts = append(pubOpts, discov.WithPubEtcdAccount(etcd.User, etcd.Pass))
    }
    if etcd.HasTLS() {
        pubOpts = append(pubOpts, discov.WithPubEtcdTLS(etcd.CertFile, etcd.CertKeyFile,
            etcd.CACertFile, etcd.InsecureSkipVerify))
    }
    if etcd.HasID() {
        pubOpts = append(pubOpts, discov.WithId(etcd.ID))
    }
    // 创建发布对象
    pubClient := discov.NewPublisher(etcd.Hosts, etcd.Key, pubListenOn, pubOpts...)
    // put key 并 keep alive
    return pubClient.KeepAlive()
}

```


### etcd 客户端服务发现  
go-zero 中的服务发现封装了 grpc 中的服务发现机制，封装较为复杂，涉及多个文件之间的调用.
* (1) 创建 Client 对象, go-zero/zrpc/client.go
```Go
// NewClient returns a Client.
func NewClient(c RpcClientConf, options ...ClientOption) (Client, error) {
	var opts []ClientOption
    // ... 

    // 初始化 target
	target, err := c.BuildTarget()
	client, err := internal.NewClient(target, c.Middlewares, opts...)

	return &RpcClient{
		client: client,
	}, nil
}

// 初始化 target 会使用配置中的 etcd 作为target
// BuildTarget builds the rpc target from the given config.
func (cc RpcClientConf) BuildTarget() (string, error) {
    // 如果没有 etcd 配置则直接使用直连方式
    // ...

    // 形成 etcd://<ip:port>/<rpc-key>
	return resolver.BuildDiscovTarget(cc.Etcd.Hosts, cc.Etcd.Key), nil
}
```

* (2) 创建 internal.Client 对象, go-zero/zrpc/internal/client.go
```Go
// client, err := internal.NewClient(target, c.Middlewares, opts...)
func NewClient(target string, middlewares ClientMiddlewaresConf, opts ...ClientOption) (Client, error) {
	cli := client{
		middlewares: middlewares,
	}

	svcCfg := fmt.Sprintf(`{"loadBalancingPolicy":"%s"}`, p2c.Name)
	balancerOpt := WithDialOption(grpc.WithDefaultServiceConfig(svcCfg))
	opts = append([]ClientOption{balancerOpt}, opts...)
    // *** 连接 target
	if err := cli.dial(target, opts...); err != nil {
		return nil, err
	}

	return &cli, nil
}

// 连接 target，最终调用 grpc.DialContext
func (c *client) dial(server string, opts ...ClientOption) error {
	// ...
	conn, err := grpc.DialContext(timeCtx, server, options...)
    // ...
	c.conn = conn
	return nil
}
```

* (3) 创建 *ClientConn 对象, 并连接 etcd, grpc/clientconn.go
```Go
func DialContext(ctx context.Context, target string, opts ...DialOption) (conn *ClientConn, err error) {
	// At the end of this method, we kick the channel out of idle, rather than
	// waiting for the first rpc.
	opts = append([]DialOption{withDefaultScheme("passthrough")}, opts...)

    // *** 创建 *ClientConn 对象
	cc, err := NewClient(target, opts...)
	if err != nil {
		return nil, err
	} 

    // ...

	// This creates the name resolver, load balancer, etc.
    // *** 退出空闲状态
	if err := cc.idlenessMgr.ExitIdleMode(); err != nil {
		return nil, err
	}


	// Return now for non-blocking dials.
	if !cc.dopts.block {
		return cc, nil
	}
	logger.Infof("_test} aaaa")
	if cc.dopts.timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, cc.dopts.timeout)
		defer cancel()
	}
	defer func() {
		select {
		case <-ctx.Done():
			switch {
			case ctx.Err() == err:
				conn = nil
			case err == nil || !cc.dopts.returnLastError:
				conn, err = nil, ctx.Err()
			default:
				conn, err = nil, fmt.Errorf("%v: %v", ctx.Err(), err)
			}
		default:
		}
	}()

	// A blocking dial blocks until the clientConn is ready.
	for {
		s := cc.GetState()
		logger.Infof("_test} s: %v", s)
		if s == connectivity.Idle {
			cc.Connect()
		}
		if s == connectivity.Ready {
			return cc, nil
		} else if cc.dopts.copts.FailOnNonTempDialError && s == connectivity.TransientFailure {
			if err = cc.connectionError(); err != nil {
				terr, ok := err.(interface {
					Temporary() bool
				})
				if ok && !terr.Temporary() {
					return nil, err
				}
			}
		}
		if !cc.WaitForStateChange(ctx, s) {
			// ctx got timeout or canceled.
			if err = cc.connectionError(); err != nil && cc.dopts.returnLastError {
				return nil, err
			}
			return nil, ctx.Err()
		}
	}
}

```

* (4) 创建 *ClientConn 对象时会初始化 target 和对应的解释器. grpc/clientconn.go
```Go
// *** 创建 *ClientConn 对象
// cc, err := NewClient(target, opts...)

func NewClient(target string, opts ...DialOption) (conn *ClientConn, err error) {
	cc := &ClientConn{
		target: target,
		conns:  make(map[*addrConn]struct{}),
		dopts:  defaultDialOptions(),
	}
    // ...

	// Determine the resolver to use.
    // *** 解析 target 并获取对应的解释器
	if err := cc.initParsedTargetAndResolverBuilder(); err != nil {
		return nil, err
	}

	// Register ClientConn with channelz. Note that this is only done after
	// channel creation cannot fail.

    // 初始化通道
	cc.channelzRegistration(target)
	channelz.Infof(logger, cc.channelz, "parsed dial target is: %#v", cc.parsedTarget)
	channelz.Infof(logger, cc.channelz, "Channel authority set to %q", cc.authority)

	cc.csMgr = newConnectivityStateManager(cc.ctx, cc.channelz) // 初始化连接状态管理器
	cc.pickerWrapper = newPickerWrapper(cc.dopts.copts.StatsHandlers) // 初始化选择器的 wrapper

	cc.initIdleStateLocked() // Safe to call without the lock, since nothing else has a reference to cc.
	cc.idlenessMgr = idle.NewManager((*idler)(cc), cc.dopts.idleTimeout) // 初始化状态管理器, *idler 是其本身
	return cc, nil
}
```

* (4.1) 解析 target 时, 会获取对应的解释器. grpc/clientconn.go
```Go
// *** 解析 target 并获取对应的解释器
// if err := cc.initParsedTargetAndResolverBuilder(); err != nil {
//     return nil, err
// }

func (cc *ClientConn) initParsedTargetAndResolverBuilder() error {
	logger.Infof("original dial target is: %q", cc.target)

	var rb resolver.Builder
    // 解释 target (etcd://<ip:port>/<rpc-key>)
	parsedTarget, err := parseTarget(cc.target)
	if err == nil {
        // *** 获取解释器
		rb = cc.getResolver(parsedTarget.URL.Scheme)
		logger.Infof("2. _test} rb is: %q, type %v", rb.Scheme(), reflect.TypeOf(rb))
		if rb != nil {
            // 赋值给成员变量
			cc.parsedTarget = parsedTarget
			cc.resolverBuilder = rb
			return nil
		}
	}

	// ... 
	cc.parsedTarget = parsedTarget
	cc.resolverBuilder = rb
	return nil
}

// 解释器先从 cc.dopts.resolvers 获取, 如果没有则在 grpc/resolver/resolver.go 中获取
func (cc *ClientConn) getResolver(scheme string) resolver.Builder {
	logger.Infof("{getResolver_test} cc.dopts.resolvers: %q", cc.dopts.resolvers)
	for _, rb := range cc.dopts.resolvers {
		if scheme == rb.Scheme() {
			return rb
		}
	}
	return resolver.Get(scheme)
}
// 其中 grpc/resolver/resolver.go 可以提前注册 builder
m = make(map[string]Builder)
func Register(b Builder) {
	m[b.Scheme()] = b
}
func Get(scheme string) Builder {
	if b, ok := m[scheme]; ok {
		return b
	}
	return nil
}
// 在 go-zero/zrpc/internal/client.go 初始化 builder
func init() {
	resolver.Register()
}
// 最终调用到注册函数，有如下几个 builder
func register() {
	resolver.Register(&directResolverBuilder)
	resolver.Register(&discovResolverBuilder)
	resolver.Register(&etcdResolverBuilder)
}
```

* (5) 回到第(3)步创建 *ClientConn 对象后, 退出空闲状态. grpc/internal/idle/idle.go
```Go
func (m *Manager) ExitIdleMode() error {
	// ...

    // *** 退出空闲状态, 这里的 enforcer 就是 *clientConn 本身
	if err := m.enforcer.ExitIdleMode(); err != nil {
		return fmt.Errorf("failed to exit idle mode: %w", err)
	}

    // ...
	return nil
}

```

* (5.1) 退出空闲状态会启动 etcd resolver, 连接 etcd 并检测变化. grpc/clientconn.go
```Go
func (i *idler) ExitIdleMode() error {
	return (*ClientConn)(i).exitIdleMode()
}
func (cc *ClientConn) exitIdleMode() (err error) {
    // ...

    // *** 开启域名解析器
	if err := cc.resolverWrapper.start(); err != nil {
		return err
	}

	cc.addTraceEvent("exiting idle mode")
	return nil
}

```
* （5.2）开启解析器后会执行 build 函数, 监听 etcd 配置变化. grpc/resolver_wrapper.go
```Go
func (ccr *ccResolverWrapper) start() error {
	errCh := make(chan error)
	ccr.serializer.Schedule(func(ctx context.Context) {
		if ctx.Err() != nil {
			return
		}
		opts := resolver.BuildOptions{
			DisableServiceConfig: ccr.cc.dopts.disableServiceConfig,
			DialCreds:            ccr.cc.dopts.copts.TransportCredentials,
			CredsBundle:          ccr.cc.dopts.copts.CredsBundle,
			Dialer:               ccr.cc.dopts.copts.Dialer,
			Authority:            ccr.cc.authority,
		}
		var err error
        // 执行 Build 函数, 这里的 ccr.cc 是 *clientConn 本身
        // ccr.cc.resolverBuilder 对应的实例是 
        /*
            type etcdBuilder struct {
                discovBuilder
            }
        */
		ccr.resolver, err = ccr.cc.resolverBuilder.Build(ccr.cc.parsedTarget, ccr, opts)
		errCh <- err
	})
	return <-errCh
}
```

* (5.3) etcd 服务发现的 build 函数如下, 解释如下。 go-zero/zrpc/resolver/internal/discovbuilder.go
```Go
func (b *discovBuilder) Build(target resolver.Target, cc resolver.ClientConn, _ resolver.BuildOptions) (
	resolver.Resolver, error) {
	hosts := strings.FieldsFunc(targets.GetAuthority(target), func(r rune) bool {
		return r == EndpointSepChar
	})
    // 创建 etcd 订阅者, 监听 key 的变化, 首次会 load 一次, 添加到 sub.items.Value 字段
	sub, err := discov.NewSubscriber(hosts, targets.GetEndpoints(target))
	if err != nil {
		return nil, err
	}

    // 每次发生变化都会调用该函数保存修改
	update := func() {
		vals := subset(sub.Values(), subsetSize)
		addrs := make([]resolver.Address, 0, len(vals))
		for _, val := range vals {
			addrs = append(addrs, resolver.Address{
				Addr: val,
			})
		}
        // func (ccr *ccResolverWrapper) UpdateState(s resolver.State) error
        // 把变化应用到 *ClientConn 中
		if err := cc.UpdateState(resolver.State{
			Addresses: addrs,
		}); err != nil {
			logx.Error(err)
		}
	}
	sub.AddListener(update)
	update()

	return &discovResolver{
		cc:  cc,
		sub: sub,
	}, nil
}

```