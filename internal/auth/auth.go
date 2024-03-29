// Пакет auth обеспечивает авторизацию пользователя и получение его идентификатора.
package auth

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"log"
	"net/http"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

const (
	secretKey            = "TheKey"         // Секретный ключ для подписи cookie
	userIDLength         = 5                // Длина идентификатора пользователя для генерации случайной последовательности символов
	cookieAuthentication = "authentication" // Заголовок HTTP-запроса для передачи данных авторизации
)

// Типы данных для хранения данных авторизации пользователя.
type (
	// authentication хранит данные авторизованного пользователя: его идентификатор, cookie и подпись для cookie.
	authentication struct {
		userID     string // Идентификатор пользователя
		cookieSign []byte // Подпись для подтверждения подлинности cookie
		cookieFull string // Переданные в HTTP-запросе или сгенерированные при авторизации cookie пользователя
	}

	// Authenticator позволяет выполнять авторизацию пользователя и получать идентификатор авторизованного пользователя.
	Authenticator interface {
		// Обработка HTTP-запроса и авторизация пользователя
		Authenticate(http.Handler) http.Handler
		// Обработка gRPC-запроса и авторизация пользователя
		GrpcAuthenticate(context.Context, interface{}, *grpc.UnaryServerInfo, grpc.UnaryHandler) (interface{}, error)
		// Получение идентификатора авторизованного пользователя
		GetUserID() string
		// Получение токена авторизованного пользователя
		GetTokenID() string
	}
)

// NewAuth создаёт экземпляр аутентификатора.
func NewAuth() Authenticator {
	a := authentication{"", make([]byte, 0), ""}
	return &a
}

// authNew создаёт идентификатор для нового пользователя и соответствующие cookie.
func (a *authentication) authNew() error {
	log.Println("Создание ID для нового пользователя")

	b := make([]byte, userIDLength)
	_, err := rand.Read(b)
	if err != nil {
		return err
	}
	log.Println("Сгенерированы случайные байты для ID нового пользователя:", b)

	a.userID = hex.EncodeToString(b)
	log.Println("Создан ID для нового пользователя:", a.userID)

	a.cookieSign, err = getSign(a.userID)
	if err != nil {
		return err
	}
	log.Println("Сгенерирована подпись в байтах:", a.cookieSign)

	a.cookieFull = a.userID + hex.EncodeToString(a.cookieSign)
	log.Println("Сгенерированы cookie из ID нового пользователя подписи:", a.cookieFull)

	return nil
}

// authExisting проверяет переданные в HTTP-запросе cookie и авторизовывает пользователя на их основании.
func (a *authentication) authExisting(cookie string) error {
	if cookie == "" {
		return errors.New("не переданы cookie для идентификации пользователя")
	}
	log.Println("Получены cookie '"+cookieAuthentication+"':", cookie)

	data, err := hex.DecodeString(cookie)
	if err != nil {
		return err
	}
	log.Println("Cookie расшифрованы в следующие байты:", data)

	if len(cookie) < userIDLength*2 {
		return errors.New("неправильная длина cookie")
	}
	id := cookie[:userIDLength*2]
	log.Println("Из cookie извлечён ID пользователя:", id)
	if id == "" {
		return errors.New("неправильная длина ID пользователя")
	}

	signReceived := data[userIDLength:]
	log.Println("Из cookie извлечена подпись:", signReceived)

	signCalculated, err := getSign(id)
	if err != nil {
		return err
	}
	log.Println("Рассчитана подпись для полученного ID пользователя:", signCalculated)

	if !hmac.Equal(signReceived, signCalculated) {
		return errors.New("в cookie передана неправильная подпись для ID пользователя")
	}

	log.Println("Рассчитанная и полученная подписи для переданного в cookie ID пользователя совпадают")
	a.userID = id
	a.cookieSign = signReceived
	a.cookieFull = cookie
	return nil
}

// Authenticate обрабатывает http-запрос на авторизацию пользователя.
// Затем передаёт запрос следующему обработчику в цепочке.
func (a *authentication) Authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		a.userID = ""
		a.cookieFull = ""

		cookie, err := r.Cookie(cookieAuthentication)
		if err != nil {
			log.Println("Cookie '" + cookieAuthentication + "' не переданы")
		}

		err = nil
		if cookie != nil {
			err = a.authExisting(cookie.Value)
		}
		if err != nil {
			log.Println("Ошибка при аутентификации пользователя через cookie 'authentication':", err)
		}

		err = nil
		if a.cookieFull == "" {
			err = a.authNew()
		}
		if err != nil {
			log.Println("Ошибка при создании ID пользователя:", err)
		}

		if a.cookieFull != "" {
			http.SetCookie(w, &http.Cookie{Name: cookieAuthentication, Value: a.cookieFull})
		}

		next.ServeHTTP(w, r)
	})
}

// Authenticate обрабатывает gRPC-запрос на авторизацию пользователя.
// Затем передаёт запрос следующему обработчику в цепочке.
func (a *authentication) GrpcAuthenticate(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	a.userID = ""
	a.cookieFull = ""

	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		log.Println("Метаданные '" + cookieAuthentication + "' не переданы")
	}

	tokens := md.Get(cookieAuthentication)
	if len(tokens) == 0 {
		log.Println("Метаданные '" + cookieAuthentication + "' не переданы")
	} else {
		err := a.authExisting(tokens[0])
		if err != nil {
			log.Println("Ошибка при аутентификации пользователя через метаданные 'authentication':", err)
		}
	}

	if a.cookieFull == "" {
		err := a.authNew()
		if err != nil {
			log.Println("Ошибка при создании ID пользователя:", err)
		}
	}

	if a.cookieFull == "" {
		err := status.Error(codes.Unauthenticated, "Ошибка при аутентификации пользователя")
		return nil, err
	}

	md.Set(cookieAuthentication, a.cookieFull)

	return handler(ctx, req)
}

// GetUserID возвращает идентификатор авторизованного пользователя.
func (a *authentication) GetUserID() string {
	return a.userID
}

// GetUserToken возвращает токен авторизованного пользователя.
func (a *authentication) GetTokenID() string {
	return a.cookieFull
}

// getSign создаёт подпись для переданного идентификатора пользователя
// по алгоритму SHA-256 с использованием секретного ключа.
func getSign(id string) ([]byte, error) {
	if id == "" {
		return nil, errors.New("не задан user ID пользователя")
	}

	h := hmac.New(sha256.New, []byte(secretKey))
	_, err := h.Write([]byte(id))
	if err != nil {
		return nil, err
	}

	return h.Sum(nil), nil
}
