package main

import (
	"fmt"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"google.golang.org/grpc"
	"lnurl-grpc-proxy/api"
	"lnurl-grpc-proxy/lnurl"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"
)

func init() {
	pflag.Uint64("grpc_port", 10512, "port to listen for incoming grpc connections")

	pflag.String("base_url", "", "the base url that the lnurl services work with e.g.: http://localhost:8012")
	pflag.String("http_host", "", "the base url that the lnurl services work with e.g.: localhost:8012")

	pflag.Parse()
	if err := viper.BindPFlags(pflag.CommandLine); err != nil {
		log.Panicf("could not bind pflags: %v", err)
	}

	viper.SetEnvPrefix("LNURLPROXY") // todo: meaningful prefix
	viper.AutomaticEnv()

	if ok := viper.IsSet("base_url"); !ok {
		log.Panicf("--base_url is not set, must be provided")
	}
}

func main() {
	var (
		grpcPort uint64 = viper.GetUint64("grpc_port")
		httpHost string = viper.GetString("http_host")
		baseUrl  string = viper.GetString("base_url")
	)
	//ctx := context.Background()
	fatalChan := make(chan error)

	lnurlService := lnurl.NewService(baseUrl)

	lis, err := net.Listen("tcp", fmt.Sprintf("0.0.0.0:%d", grpcPort))
	if err != nil {
		log.Panicf("\t [GRPC] > can not listen: %v", err)
	}
	defer lis.Close()

	grpcServer := grpc.NewServer()
	lnurlGrpc := lnurl.NewGrpcServer(lnurlService)
	api.RegisterWithdrawProxyServer(grpcServer, lnurlGrpc)

	go func() {
		log.Println("\t [MAIN] > serving grpc")
		err := grpcServer.Serve(lis)
		if err != nil {
			fatalChan <- err
		}
	}()
	defer grpcServer.Stop()

	lnurlHandler := lnurl.NewRestHandler(lnurlService)

	go func() {
		log.Println("\t [MAIN] > serving Http")
		err := lnurlHandler.Listen(httpHost)
		if err != nil {
			fatalChan <- err
		}

	}()
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	log.Println("\t [MAIN] > await signal")
	<-sigs
	log.Println("\t [MAIN] > exiting")

}
