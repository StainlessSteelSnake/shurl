package grpc_server

import (
	"context"
	"errors"
	"log"

	pb "github.com/StainlessSteelSnake/shurl/internal/grpc_server/proto"
	"github.com/StainlessSteelSnake/shurl/internal/storage"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (s *grpcServer) PostLongUrl(ctx context.Context, req *pb.PostLongUrlRequest) (*pb.PostLongUrlResponse, error) {
	log.Println("Пришедший в запросе исходный URL:", req.OriginalUrl)

	if req.OriginalUrl == "" {
		log.Println("Неверный формат URL")
		return nil, status.Error(codes.InvalidArgument, "неверный формат URL")
	}

	longURL := req.OriginalUrl

	var response = pb.PostLongUrlResponse{}

	shortURL, err := s.storage.AddURL(longURL, s.auth.GetUserID())
	if err != nil && errors.Is(err, storage.DBError{LongURL: longURL, Duplicate: false, Err: nil}) {
		log.Println("Ошибка '", err, "' при добавлении в БД URL:", longURL)
		return nil, status.Errorf(codes.Internal, "ошибка при добавлении в БД: "+err.Error())
	}

	var resultError error
	if err != nil && errors.Is(err, storage.DBError{LongURL: longURL, Duplicate: true, Err: nil}) {
		log.Println("Найденный короткий идентификатор URL:", shortURL)
		resultError = status.Error(codes.AlreadyExists, "Найденный короткий идентификатор URL:"+shortURL)
	} else if err != nil {
		log.Println("Ошибка '", err, "' при добавлении в БД URL:", longURL)
		resultError = status.Error(codes.Internal, "Ошибка '"+err.Error()+"' при добавлении в БД URL:"+longURL)
	} else {
		log.Println("Созданный короткий идентификатор URL:", shortURL)
		resultError = status.Error(codes.OK, "Созданный короткий идентификатор URL:"+shortURL)
	}

	response.ShortUrl = s.baseURL + shortURL

	return &response, resultError
}

func (s *grpcServer) GetLongUrl(ctx context.Context, req *pb.GetLongUrlRequest) (*pb.GetLongUrlResponse, error) {
	return nil, nil
}

func (s *grpcServer) PostLongUrls(ctx context.Context, req *pb.PostLongUrlsRequest) (*pb.PostLongUrlsResponse, error) {
	return nil, nil
}

func (s *grpcServer) GetLongUrlsByUser(ctx context.Context, req *pb.GetLongUrlsByUserRequest) (*pb.GetLongUrlsByUserResponse, error) {
	return nil, nil
}

func (s *grpcServer) Delete(ctx context.Context, req *pb.DeleteRequest) (*pb.DeleteResponse, error) {
	return nil, nil
}

func (s *grpcServer) Ping(ctx context.Context, req *pb.PingRequest) (*pb.PingResponse, error) {
	err := s.storage.Ping()
	if err != nil {
		log.Println(err)
		errResponse := status.Error(codes.Internal, err.Error())
		return nil, errResponse
	}

	return &pb.PingResponse{Token: s.auth.GetTokenID()}, nil
}

func (s *grpcServer) Stats(ctx context.Context, req *pb.StatsRequest) (*pb.StatsResponse, error) {
	var response pb.StatsResponse

	urls, users := s.storage.GetStatistics()
	response.Urls, response.Users = int32(urls), int32(users)
	response.Token = s.auth.GetTokenID()

	return &response, nil
}
