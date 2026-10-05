package server

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	reflectpb "google.golang.org/grpc/reflection/grpc_reflection_v1"
	"google.golang.org/grpc/test/bufconn"

	context2 "github.com/wensboy/quick_arch/internal/context"
)

func TestRpcServer_Setup_RegistersHealth(t *testing.T) {
	srv := NewRpcServer()
	srv.Setup(*testServerContext(t, NewMiddlewareStore()))

	if resp := rpcRequest(t, serveBufconn(t, srv)); resp.GetStatus() != healthpb.HealthCheckResponse_SERVING {
		t.Fatalf("status = %v, want SERVING", resp.GetStatus())
	}
}

func TestRpcServer_Setup_AppliesUnaryMiddleware(t *testing.T) {
	var marked bool

	store := NewMiddlewareStore()
	store.RegisterUnary("mark", func(context2.MiddlewareContext) []grpc.UnaryServerInterceptor {
		return []grpc.UnaryServerInterceptor{func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
			marked = true
			return handler(ctx, req)
		}}
	})

	srv := NewRpcServer().UseUnary("mark", "missing")
	srv.Setup(*testServerContext(t, store))

	rpcRequest(t, serveBufconn(t, srv))

	if !marked {
		t.Fatal("unary middleware from MiddlewareContext was not applied")
	}
}

func TestRpcServer_Setup_AppliesStreamMiddleware(t *testing.T) {
	var marked bool

	store := NewMiddlewareStore()
	store.RegisterStream("mark_stream", func(context2.MiddlewareContext) []grpc.StreamServerInterceptor {
		return []grpc.StreamServerInterceptor{func(srv any, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
			marked = true
			return handler(srv, ss)
		}}
	})

	srv := NewRpcServer().UseStream("mark_stream", "missing")
	srv.Setup(*testServerContext(t, store))

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// 健康检查的 Watch 是服务端流, 借它验证流拦截器链已生效.
	stream, err := healthpb.NewHealthClient(serveBufconn(t, srv)).Watch(ctx, &healthpb.HealthCheckRequest{})
	if err != nil {
		t.Fatalf("watch: %v", err)
	}
	if _, err := stream.Recv(); err != nil {
		t.Fatalf("recv: %v", err)
	}

	if !marked {
		t.Fatal("stream middleware from MiddlewareContext was not applied")
	}
}

func TestRpcServer_Reflection(t *testing.T) {
	cases := []struct {
		name       string
		reflection bool
		want       string
	}{
		{"enabled", true, "grpc.health.v1.Health"},
		{"disabled", false, ""},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			srv := NewRpcServer().WithReflection(tc.reflection)
			srv.Setup(*testServerContext(t, NewMiddlewareStore()))

			names := listServices(t, serveBufconn(t, srv))
			if tc.want == "" {
				if len(names) != 0 {
					t.Fatalf("reflection should be off, got %v", names)
				}
				return
			}
			if !contains(names, tc.want) {
				t.Fatalf("services = %v, want %s", names, tc.want)
			}
		})
	}
}

func TestRpcServer_Setup_TLS(t *testing.T) {
	certFile, keyFile, pool := selfSignedCert(t)

	srv := NewRpcServer().WithTLS(true, certFile, keyFile)
	srv.Setup(*testServerContext(t, NewMiddlewareStore()))

	conn := serveBufconn(t, srv, grpc.WithTransportCredentials(
		credentials.NewClientTLSFromCert(pool, "localhost"),
	))

	if resp := rpcRequest(t, conn); resp.GetStatus() != healthpb.HealthCheckResponse_SERVING {
		t.Fatalf("status = %v, want SERVING", resp.GetStatus())
	}
}

func TestRpcServer_Setup_InvalidTLSAbortsStart(t *testing.T) {
	dir := t.TempDir()
	srv := NewRpcServer().WithTLS(true, filepath.Join(dir, "none.crt"), filepath.Join(dir, "none.key"))
	srv.Setup(context2.ServerContext{})

	if srv.setupErr == nil {
		t.Fatal("setupErr = nil, want TLS config error")
	}

	var canceled bool
	srv.Start(func() { canceled = true })

	if !canceled {
		t.Fatal("Start should abort when TLS config is invalid")
	}
}

func rpcRequest(t *testing.T, conn *grpc.ClientConn) *healthpb.HealthCheckResponse {
	t.Helper()

	resp, err := healthpb.NewHealthClient(conn).Check(context.Background(), &healthpb.HealthCheckRequest{})
	if err != nil {
		t.Fatalf("health check: %v", err)
	}
	return resp
}

// listServices 通过服务反射列出服务名, 反射未开启时返回空.
func listServices(t *testing.T, conn *grpc.ClientConn) []string {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	stream, err := reflectpb.NewServerReflectionClient(conn).ServerReflectionInfo(ctx)
	if err != nil {
		t.Fatalf("reflection stream: %v", err)
	}
	if err := stream.Send(&reflectpb.ServerReflectionRequest{
		MessageRequest: &reflectpb.ServerReflectionRequest_ListServices{ListServices: "*"},
	}); err != nil {
		t.Fatalf("reflection send: %v", err)
	}

	resp, err := stream.Recv()
	if err != nil {
		// 反射未注册时服务端返回 Unimplemented, 视为空列表.
		return nil
	}

	names := make([]string, 0)
	for _, svc := range resp.GetListServicesResponse().GetService() {
		names = append(names, svc.GetName())
	}
	return names
}

func testServerContext(t *testing.T, store *MiddlewareStore) *context2.ServerContext {
	t.Helper()
	return context2.NewServerContext(nil, nil, nil, store, nil)
}

// serveBufconn 在内存 listener 上启动 rpc 服务, 返回客户端连接.
func serveBufconn(t *testing.T, srv *RpcServer, opts ...grpc.DialOption) *grpc.ClientConn {
	t.Helper()

	listener := bufconn.Listen(1024 * 1024)
	go func() { _ = srv.Handler().Serve(listener) }()

	dialOpts := append([]grpc.DialOption{
		grpc.WithContextDialer(func(ctx context.Context, _ string) (net.Conn, error) {
			return listener.DialContext(ctx)
		}),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	}, opts...)

	conn, err := grpc.NewClient("passthrough:///bufnet", dialOpts...)
	if err != nil {
		t.Fatalf("new client: %v", err)
	}

	t.Cleanup(func() {
		_ = conn.Close()
		srv.Handler().Stop()
	})
	return conn
}

// selfSignedCert 生成仅用于测试的自签证书, 返回证书/私钥路径与信任池.
func selfSignedCert(t *testing.T) (certFile, keyFile string, pool *x509.CertPool) {
	t.Helper()

	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}

	tmpl := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: "localhost"},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(time.Hour),
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		IsCA:                  true,
		DNSNames:              []string{"localhost"},
		IPAddresses:           []net.IP{net.ParseIP("127.0.0.1")},
	}

	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		t.Fatalf("create certificate: %v", err)
	}
	keyDER, err := x509.MarshalECPrivateKey(key)
	if err != nil {
		t.Fatalf("marshal key: %v", err)
	}

	dir := t.TempDir()
	certFile = filepath.Join(dir, "server.crt")
	keyFile = filepath.Join(dir, "server.key")

	if err := os.WriteFile(certFile, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}), 0o600); err != nil {
		t.Fatalf("write cert: %v", err)
	}
	if err := os.WriteFile(keyFile, pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER}), 0o600); err != nil {
		t.Fatalf("write key: %v", err)
	}

	parsed, err := x509.ParseCertificate(der)
	if err != nil {
		t.Fatalf("parse certificate: %v", err)
	}
	pool = x509.NewCertPool()
	pool.AddCert(parsed)

	return certFile, keyFile, pool
}

func contains(names []string, want string) bool {
	for _, name := range names {
		if name == want {
			return true
		}
	}
	return false
}
