package startrpc

import (
	"OpenIM/internal/common/network"
	"OpenIM/pkg/common/config"
	"OpenIM/pkg/common/constant"
	"OpenIM/pkg/common/log"
	"OpenIM/pkg/common/middleware"
	"OpenIM/pkg/common/prome"
	"flag"
	"fmt"
	"github.com/OpenIMSDK/openKeeper"
	grpcPrometheus "github.com/grpc-ecosystem/go-grpc-prometheus"
	"google.golang.org/grpc"
	"net"
)

func start(rpcPorts []int, rpcRegisterName string, prometheusPorts []int, rpcFn func(client *openKeeper.ZkClient, server *grpc.Server) error, options []grpc.ServerOption) error {
	flagRpcPort := flag.Int("port", rpcPorts[0], "get RpcGroupPort from cmd,default 16000 as port")
	flagPrometheusPort := flag.Int("prometheus_port", prometheusPorts[0], "groupPrometheusPort default listen port")
	configPath := flag.String("config_path", "../config/", "config folder")
	flag.Parse()
	if err := config.InitConfig(*configPath); err != nil {
		return err
	}
	fmt.Println("start group rpc server, port: ", *flagRpcPort, ", OpenIM version: ", constant.CurrentVersion)
	log.NewPrivateLog(constant.LogFileName)
	listener, err := net.Listen("tcp", fmt.Sprintf("%s:%d", config.Config.ListenIP, *flagRpcPort))
	if err != nil {
		return err
	}
	defer listener.Close()
	zkClient, err := openKeeper.NewClient(config.Config.Zookeeper.ZkAddr, config.Config.Zookeeper.Schema, 10, "", "")
	if err != nil {
		return err
	}
	defer zkClient.Close()
	registerIP, err := network.GetRpcRegisterIP(config.Config.RpcRegisterIP)
	if err != nil {
		return err
	}
	options = append(options, grpc.UnaryInterceptor(middleware.RpcServerInterceptor)) // ctx 中间件
	if config.Config.Prometheus.Enable {
		prome.NewGrpcRequestCounter()
		prome.NewGrpcRequestFailedCounter()
		prome.NewGrpcRequestSuccessCounter()
		options = append(options, []grpc.ServerOption{
			//grpc.UnaryInterceptor(prome.UnaryServerInterceptorPrometheus),
			grpc.StreamInterceptor(grpcPrometheus.StreamServerInterceptor),
			grpc.UnaryInterceptor(grpcPrometheus.UnaryServerInterceptor),
		}...)
	}
	srv := grpc.NewServer(options...)
	defer srv.GracefulStop()
	err = zkClient.Register(rpcRegisterName, registerIP, *flagRpcPort)
	if err != nil {
		return err
	}
	if config.Config.Prometheus.Enable {
		err := prome.StartPromeSrv(*flagPrometheusPort)
		if err != nil {
			return err
		}
	}
	return rpcFn(zkClient, srv)
}

func Start(rpcPorts []int, rpcRegisterName string, prometheusPorts []int, rpcFn func(client *openKeeper.ZkClient, server *grpc.Server) error, options ...grpc.ServerOption) {
	err := start(rpcPorts, rpcRegisterName, prometheusPorts, rpcFn, options)
	fmt.Println("end", err)
}
