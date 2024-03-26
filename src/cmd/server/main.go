package main

import (
	"context"
	"errors"
	"fmt"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/reflection"
	"google.golang.org/grpc/status"
	"io"
	"log"
	"net"
	"os"
	"os/signal"
	"time"

	hallopb "mygrpc/pkg/grpc"
)

func main() {
	port := 8080
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		panic(err)
	}

	s := grpc.NewServer(grpc.UnaryInterceptor(myUnaryServerInterceptor1), grpc.StreamInterceptor(myStreamServerInterceptor1))

	hallopb.RegisterGreetingServiceServer(s, NewMyServer())

	reflection.Register(s)

	go func() {
		log.Printf("start gRPC server port: %v", port)
		s.Serve(listener)
	}()

	// if enter Ctrl+C, graceful shutdown server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt)
	<-quit
	log.Printf("stopping gRPC server...")
	s.GracefulStop()
}

type myServer struct {
	hallopb.UnimplementedGreetingServiceServer
}

func NewMyServer() *myServer {
	return &myServer{}
}

func (s *myServer) Hello(ctx context.Context, req *hallopb.HelloRequest) (*hallopb.HelloResponse, error) {
	stat := status.New(codes.Unknown, "unknown error occurred")
	stat, _ = stat.WithDetails(&errdetails.DebugInfo{
		Detail: "detail reason of err",
	})
	err := stat.Err()

	return nil, err
	//return &hallopb.HelloResponse{
	//	Message: fmt.Sprintf("Hello, %s!", req.GetName()),
	//}, nil
}

func (s *myServer) HelloServerStream(req *hallopb.HelloRequest, stream hallopb.GreetingService_HelloServerStreamServer) error {
	resCount := 5
	for i := 0; i < resCount; i++ {
		if err := stream.Send(&hallopb.HelloResponse{Message: fmt.Sprintf("[%d] Hello, %s!", i, req.GetName())}); err != nil {
			return err
		}

		time.Sleep(time.Second * 1)
	}

	// returnでメソッドを終了 = ストリームの終わりになる
	return nil
}

func (s *myServer) HelloClientStream(stream hallopb.GreetingService_HelloClientStreamServer) error {
	nameList := make([]string, 0)
	for {
		req, err := stream.Recv()
		if errors.Is(err, io.EOF) {
			message := fmt.Sprintf("Hello, %s!", nameList)
			return stream.SendAndClose(&hallopb.HelloResponse{Message: message})
		}
		if err != nil {
			return err
		}

		nameList = append(nameList, req.GetName())
	}
}

func (s *myServer) HelloBidirectionalStream(stream hallopb.GreetingService_HelloBidirectionalStreamServer) error {
	for {
		req, err := stream.Recv()
		if errors.Is(err, io.EOF) {
			return nil
		}
		if err != nil {
			return err
		}

		if err := stream.Send(&hallopb.HelloResponse{Message: fmt.Sprintf("Hello, %s!", req.GetName())}); err != nil {
			return err
		}
	}
}
