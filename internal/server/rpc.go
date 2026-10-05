package server

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"

	context2 "github.com/wensboy/quick_arch/internal/context"
	errs "github.com/wensboy/quick_arch/internal/error"
)

// RpcServer 是 gRPC 服务端, 与 RestServer 对齐: 地址 / 中间件 / 挂载 / 启停.
// 一元与流两种调用形态的拦截器分别声明, 各自独立成链.
type RpcServer struct {
	address     string
	unaryNames  []string
	streamNames []string
	services    []Service
	reflection  bool
	tlsEnabled  bool
	tlsCertFile string
	tlsKeyFile  string
	setupErr    error
	server      *grpc.Server
}

// NewRpcServer 默认开启服务反射: grpcurl 等工具可据此免 proto 文件直接调试.
func NewRpcServer() *RpcServer {
	return &RpcServer{reflection: true}
}

func (rs *RpcServer) WithAddress(addr string) *RpcServer {
	if addr != "" {
		rs.address = addr
	}
	return rs
}

// WithReflection 控制是否注册 gRPC 反射服务.
func (rs *RpcServer) WithReflection(enabled bool) *RpcServer {
	rs.reflection = enabled
	return rs
}

// WithTLS 开启传输层 TLS; 证书加载失败时服务不会启动 (不会静默降级为明文).
func (rs *RpcServer) WithTLS(enabled bool, certFile, keyFile string) *RpcServer {
	rs.tlsEnabled, rs.tlsCertFile, rs.tlsKeyFile = enabled, certFile, keyFile
	return rs
}

// UseUnary 声明按序生效的一元拦截器名, 名称需注册到 MiddlewareContext 的一元注册表.
func (rs *RpcServer) UseUnary(names ...string) *RpcServer {
	rs.unaryNames = append(rs.unaryNames, names...)
	return rs
}

// UseStream 声明按序生效的流拦截器名, 名称需注册到 MiddlewareContext 的流注册表.
func (rs *RpcServer) UseStream(names ...string) *RpcServer {
	rs.streamNames = append(rs.streamNames, names...)
	return rs
}

func (rs *RpcServer) Mount(services ...Service) *RpcServer {
	rs.services = append(rs.services, services...)
	return rs
}

// Handler 暴露底层 grpc.Server, 便于测试与服务注册.
func (rs *RpcServer) Handler() *grpc.Server { return rs.server }

// Setup 装配一元/流拦截器链与传输层 TLS, 注册内置健康检查/反射与全部服务.
func (rs *RpcServer) Setup(ctx context2.ServerContext) {
	var opts []grpc.ServerOption
	if ctx.MiddlewareContext != nil {
		if chain := ctx.ResolveUnary(rs.unaryNames...); len(chain) > 0 {
			opts = append(opts, grpc.ChainUnaryInterceptor(chain...))
		}
		if chain := ctx.ResolveStream(rs.streamNames...); len(chain) > 0 {
			opts = append(opts, grpc.ChainStreamInterceptor(chain...))
		}
	}
	if rs.tlsEnabled {
		creds, err := credentials.NewServerTLSFromFile(rs.tlsCertFile, rs.tlsKeyFile)
		if err != nil {
			rs.setupErr = errs.Wrap(ErrTLS, err).With("cert", rs.tlsCertFile, "key", rs.tlsKeyFile)
		} else {
			opts = append(opts, grpc.Creds(creds))
		}
	}

	rs.server = grpc.NewServer(opts...)

	healthpb.RegisterHealthServer(rs.server, health.NewServer())
	if rs.reflection {
		reflection.Register(rs.server)
	}
	for _, service := range rs.services {
		service.Register(rs.server)
	}
}

func (rs *RpcServer) Start(cancel context.CancelFunc) {
	if rs.setupErr != nil {
		fmt.Fprintf(os.Stderr, "%s\n", rs.setupErr)
		cancel()
		return
	}

	listener, err := net.Listen("tcp", rs.address)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", errs.Wrap(ErrListen, err).With("address", rs.address))
		cancel()
		return
	}
	if err := rs.server.Serve(listener); err != nil && !errors.Is(err, grpc.ErrServerStopped) {
		fmt.Fprintf(os.Stderr, "%s\n", err)
	}
	cancel()
}

// Stop 先优雅停机, 超出 ctx 期限则强制关闭.
func (rs *RpcServer) Stop(ctx context.Context) {
	stopped := make(chan struct{})
	go func() {
		rs.server.GracefulStop()
		close(stopped)
	}()
	select {
	case <-stopped:
	case <-ctx.Done():
		rs.server.Stop()
	}
}
