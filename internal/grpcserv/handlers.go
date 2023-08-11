package grpcserv

import (
	"context"
	"errors"
	"log"
	"strings"

	pb "github.com/StainlessSteelSnake/shurl/internal/grpcserv/proto"
	"github.com/StainlessSteelSnake/shurl/internal/storage"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// PostLongUrl обрабатывает gRPC-запрос на сокращение URL, возвращает короткий URL.
func (s *grpcServer) PostLongUrl(ctx context.Context, req *pb.PostLongUrlRequest) (*pb.PostLongUrlResponse, error) {
	log.Println("Пришедший в запросе исходный URL:", req.OriginalUrl)

	if req.OriginalUrl == "" {
		log.Println("Неверный формат URL")
		return nil, status.Error(codes.InvalidArgument, "неверный формат URL")
	}

	longURL := req.OriginalUrl

	var response = pb.PostLongUrlResponse{Token: s.auth.GetTokenID()}

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

// GetLongUrl обрабатывает gRPC-запрос на восстановление исходного URL по переданному короткому URL.
func (s *grpcServer) GetLongUrl(ctx context.Context, req *pb.GetLongUrlRequest) (*pb.GetLongUrlResponse, error) {
	var response = pb.GetLongUrlResponse{Token: s.auth.GetTokenID()}

	shortUrl := req.ShortUrl
	log.Println("Идентификатор короткого URL, полученный из gRPC-запроса:", shortUrl)

	result, err := s.storage.FindURL(shortUrl)
	if err != nil {
		log.Println("Ошибка '", err, "'. Не найден URL с указанным коротким идентификатором:", shortUrl)
		return nil, status.Error(codes.NotFound, "URL с указанным коротким идентификатором не найден")
	}

	if result.Deleted {
		log.Println("URL", result.LongURL, "для короткого идентификатора", shortUrl, "был удалён")
		return nil, status.Error(codes.Unavailable, "URL с указанным коротким идентификатором не найден")
	}

	log.Println("Найден URL", result.LongURL, "для короткого идентификатора", shortUrl)
	response.OriginalUrl = result.LongURL

	return &response, nil
}

// PostLongUrls обрабатывает gRPC-запрос на сокращение переданных URL, возвращает список коротких URL.
func (s *grpcServer) PostLongUrls(ctx context.Context, req *pb.PostLongUrlsRequest) (*pb.PostLongUrlsResponse, error) {
	var response = pb.PostLongUrlsResponse{Token: s.auth.GetTokenID()}

	var longUrls = make(storage.BatchURLs, 0, len(req.LongUrls))
	for _, longUrl := range req.LongUrls {
		longUrls = append(longUrls, storage.RecordURL{ID: longUrl.CorrelationId, URL: longUrl.OriginalUrl})
	}

	shortUrls, err := s.storage.AddURLs(longUrls, s.auth.GetUserID())
	if err != nil {
		log.Println("Ошибка '", err, "' при добавлении в БД URLs:", longUrls)
		return nil, status.Error(codes.Internal, "ошибка при добавлении в БД URLs: "+err.Error())
	}

	response.ShortUrls = make([]*pb.PostLongUrlsResponse_PostLongUrlResponseRecord, len(shortUrls))
	for _, shortUrl := range shortUrls {
		response.ShortUrls = append(response.ShortUrls, &pb.PostLongUrlsResponse_PostLongUrlResponseRecord{
			CorrelationId: shortUrl.ID,
			ShortUrl:      s.baseURL + shortUrl.URL,
		})
	}

	return &response, nil
}

// GetLongUrlsByUser обрабатывает gRPC-запрос на получение списка всех сокращённых и исходных URL для текущего пользователя.
func (s *grpcServer) GetLongUrlsByUser(ctx context.Context, req *pb.GetLongUrlsByUserRequest) (*pb.GetLongUrlsByUserResponse, error) {
	var response = pb.GetLongUrlsByUserResponse{Token: s.auth.GetTokenID()}

	urls := s.storage.GetURLsByUser(s.auth.GetUserID())
	if len(urls) == 0 {
		log.Println("Для пользователя с идентификатором '" + s.auth.GetUserID() + "' не найдены сохранённые URL")
		return &response, nil
	}
	log.Println("Для пользователя с идентификатором '"+s.auth.GetUserID()+"' найдено ", len(urls), "сохранённых URL:")

	for i, shortURL := range urls {
		result, err := s.storage.FindURL(shortURL)
		if err != nil {
			continue
		}

		record := pb.GetLongUrlsByUserResponse_GetLongUrlsByUserResponseRecord{
			ShortUrl:    s.baseURL + shortURL,
			OriginalUrl: result.LongURL,
		}
		log.Println("Запись", i, "короткий URL", record.ShortUrl, "длинный URL", record.OriginalUrl)
		response.Urls = append(response.Urls, &record)
	}

	return &response, nil
}

// Delete обрабатывает gRPC-запрос на удаление переданных URL.
func (s *grpcServer) Delete(ctx context.Context, req *pb.DeleteRequest) (*pb.DeleteResponse, error) {
	var response = pb.DeleteResponse{Token: s.auth.GetTokenID()}

	log.Println("Тело запроса на удаление данных:\n", req.ShortUrls)
	if len(req.ShortUrls) == 0 {
		log.Println("Пустой список идентификаторов URL")
		return nil, status.Error(codes.InvalidArgument, "пустой список идентификаторов URL")
	}

	for i, record := range req.ShortUrls {
		req.ShortUrls[i] = strings.Replace(record, s.baseURL, "", -1)
	}
	log.Println("Список подлежащих удалению коротких идентификаторов URL:\n", req.ShortUrls)

	_ = s.storage.DeleteURLs(req.ShortUrls, s.auth.GetUserID())

	return &response, nil
}

// Ping обрабатывает gRPC-запрос на проверку подключения к хранилищу сокращённых URL.
func (s *grpcServer) Ping(ctx context.Context, req *pb.PingRequest) (*pb.PingResponse, error) {
	err := s.storage.Ping()
	if err != nil {
		log.Println(err)
		errResponse := status.Error(codes.Internal, err.Error())
		return nil, errResponse
	}

	return &pb.PingResponse{Token: s.auth.GetTokenID()}, nil
}

// Stats обрабатывает gRPC-запрос на получение статистики сервиса: количества URL и пользователей.
func (s *grpcServer) Stats(ctx context.Context, req *pb.StatsRequest) (*pb.StatsResponse, error) {
	var response = pb.StatsResponse{Token: s.auth.GetTokenID()}

	urls, users := s.storage.GetStatistics()
	response.Urls, response.Users = int32(urls), int32(users)

	return &response, nil
}
