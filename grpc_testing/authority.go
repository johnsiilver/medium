package main

import (
	"flag"
	"log"
	"net"
	"net/http"

	"github.com/johnsiilver/medium/grpc_testing/service"

	"google.golang.org/grpc"

	pb "github.com/johnsiilver/medium/grpc_testing/proto"

	_ "net/http/pprof"
)

var (
	addr = flag.String("addr", "localhost:38457", "The host:port to run on")
)

func main() {
	flag.Parse()

	// This is a pprof endpoint.
	go func() {
		log.Println(http.ListenAndServe("localhost:6060", nil))
	}()

	auth := service.New("data/prod/data.json")

	log.Println("Running on: ", *addr)
	lis, err := net.Listen("tcp", *addr)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	grpcServer := grpc.NewServer()
	pb.RegisterAuthorityServer(grpcServer, auth)

	log.Printf("grpc server stopped: %v", grpcServer.Serve(lis))
}
