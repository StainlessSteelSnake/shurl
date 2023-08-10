package grpc_server

import (
	"log"
	"net"

	"github.com/StainlessSteelSnake/shurl/internal/auth"
	pb "github.com/StainlessSteelSnake/shurl/internal/grpc_server/proto"
	"github.com/StainlessSteelSnake/shurl/internal/storage"
	"google.golang.org/grpc"
)

type grpcServer struct {
	pb.UnimplementedShurlServiceServer
	storage storage.Storager
	auth    auth.Authenticator
	baseURL string
}

func NewServer(host string, baseURL string, storage storage.Storager, auth auth.Authenticator) (*grpc.Server, error) {
	server := grpcServer{
		storage: storage,
		auth:    auth,
		baseURL: baseURL,
	}

	// определяем порт для сервера
	listener, err := net.Listen("tcp", host)
	if err != nil {
		log.Fatalln("Ошибка при открытии tcp-канала", host, "для gRPC-сервера:", err)
		return nil, err
	}

	// создаём gRPC-сервер без зарегистрированной службы
	s := grpc.NewServer(grpc.UnaryInterceptor(server.auth.GrpcAuthenticate))

	// регистрируем сервис

	pb.RegisterShurlServiceServer(s, &server)
	log.Println("Сервер gRPC начал работу")

	go func() {
		if err := s.Serve(listener); err != nil {
			log.Fatalln("Ошибка при обработке запросов к gRPC-серверу:", err)
		}
	}()

	return s, nil
}
