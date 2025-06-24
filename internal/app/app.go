package app

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"strings"
	"syscall"
	"time"

	grpcruntime "github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/project/library/config"
	"github.com/project/library/db"
	generated "github.com/project/library/generated/api/library"
	"github.com/project/library/internal/controller"
	"github.com/project/library/internal/entity"
	"github.com/project/library/internal/usecase/library"
	"github.com/project/library/internal/usecase/outbox"
	"github.com/project/library/internal/usecase/repository"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/reflection"
)

const sleepTime = 3
const dialerTimeout = 30
const dialerKeepAlive = 180
const transportMaxIdleConns = 100
const transportMaxConnsPerHost = 100
const transportIdleConnTimeout = 90
const transportTLSHandshakeTimeout = 15
const transportExpectContinueTimeout = 2
const httpMinErrorStatus = 400

func Run(logger *zap.Logger, cfg *config.Config) {
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	dbPool, err := pgxpool.New(ctx, cfg.PG.URL)

	if err != nil {
		logger.Error("can not create pgxpool", zap.Error(err))
		return
	}

	defer dbPool.Close()

	db.SetupPostgres(dbPool, logger)

	repo := repository.NewPostgresRepository(logger, dbPool)
	outboxRepository := repository.NewOutbox(dbPool)

	transactor := repository.NewTransactor(dbPool, logger)
	go runOutbox(ctx, cfg, logger, outboxRepository, transactor)

	useCases := library.New(logger, transactor, outboxRepository, repo, repo)

	ctrl := controller.New(logger, useCases, useCases)

	go runRest(ctx, cfg, logger)
	go runGrpc(cfg, logger, ctrl)

	<-ctx.Done()

	time.Sleep(time.Second * sleepTime)
}

func runOutbox(
	ctx context.Context,
	cfg *config.Config,
	logger *zap.Logger,
	outboxRepository repository.OutboxRepository,
	transactor repository.Transactor,
) {
	dialer := &net.Dialer{
		Timeout:   dialerTimeout * time.Second,
		KeepAlive: dialerKeepAlive * time.Second,
	}

	transport := &http.Transport{
		DialContext:           dialer.DialContext,
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          transportMaxIdleConns,
		MaxConnsPerHost:       transportMaxConnsPerHost,
		IdleConnTimeout:       transportIdleConnTimeout * time.Second,
		TLSHandshakeTimeout:   transportTLSHandshakeTimeout * time.Second,
		ExpectContinueTimeout: transportExpectContinueTimeout * time.Second,
		MaxIdleConnsPerHost:   runtime.GOMAXPROCS(0) + 1,
	}

	client := new(http.Client)
	client.Transport = transport

	globalHandler := globalOutboxHandler(client, cfg.Outbox.BookSendURL, cfg.Outbox.AuthorSendURL, logger)
	outboxService := outbox.New(logger, outboxRepository, globalHandler, cfg, transactor)

	outboxService.Start(
		ctx,
		cfg.Outbox.Workers,
		cfg.Outbox.BatchSize,
		cfg.Outbox.WaitTimeMS,
		cfg.Outbox.InProgressTTLMS,
	)
}

func globalOutboxHandler(
	client *http.Client,
	bookURL string,
	authorURL string,
	logger *zap.Logger,
) outbox.GlobalHandler {
	return func(kind repository.OutboxKind) (outbox.KindHandler, error) {
		switch kind {
		case repository.OutboxKindBook:
			return bookOutboxHandler(client, bookURL, logger), nil
		case repository.OutboxKindAuthor:
			return authorOutboxHandler(client, authorURL, logger), nil
		default:
			return nil, fmt.Errorf("unsupported outbox kind: %d", kind)
		}
	}
}

func SendID(client *http.Client, url string, id string, logger *zap.Logger) error {
	resp, err := client.Post(url, "text/plain", strings.NewReader(id))

	if err != nil {
		return fmt.Errorf("error while processing post request: %w", err)
	}

	defer func() {
		err = resp.Body.Close()
		if err != nil {
			logger.Error("Error while closing response body.", zap.Error(err))
		}
	}()

	if resp.StatusCode >= httpMinErrorStatus {
		return errors.New("http error: " + resp.Status)
	}

	return nil
}

func bookOutboxHandler(client *http.Client, url string, logger *zap.Logger) outbox.KindHandler {
	return func(_ context.Context, data []byte) error {
		book := entity.Book{}
		err := json.Unmarshal(data, &book)

		if err != nil {
			logger.Error("error while deserializing data in book.")
			return fmt.Errorf("can not deserialize data in book outbox handler: %w", err)
		}

		return SendID(client, url, book.ID, logger)
	}
}

func authorOutboxHandler(client *http.Client, url string, logger *zap.Logger) outbox.KindHandler {
	return func(_ context.Context, data []byte) error {
		author := entity.Author{}
		err := json.Unmarshal(data, &author)

		if err != nil {
			logger.Error("error while deserializing data in author.")
			return fmt.Errorf("can not deserialize data in author outbox handler: %w", err)
		}

		return SendID(client, url, author.ID, logger)
	}
}

func runRest(ctx context.Context, cfg *config.Config, logger *zap.Logger) {
	mux := grpcruntime.NewServeMux()
	opts := []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())}

	address := "localhost:" + cfg.GRPC.Port
	err := generated.RegisterLibraryHandlerFromEndpoint(ctx, mux, address, opts)

	if err != nil {
		logger.Error("can not register grpc gateway", zap.Error(err))
		os.Exit(-1)
	}

	gatewayPort := ":" + cfg.GRPC.GatewayPort
	logger.Info("gateway listening at port", zap.String("port", gatewayPort))

	if err = http.ListenAndServe(gatewayPort, mux); err != nil {
		logger.Error("gateway listen error", zap.Error(err))
	}
}

func runGrpc(cfg *config.Config, logger *zap.Logger, libraryService generated.LibraryServer) {
	port := ":" + cfg.GRPC.Port
	lis, err := net.Listen("tcp", port)

	if err != nil {
		logger.Error("can not open tcp socket", zap.Error(err))
		os.Exit(-1)
	}

	s := grpc.NewServer()
	reflection.Register(s)

	generated.RegisterLibraryServer(s, libraryService)

	logger.Info("grpc server listening at port", zap.String("port", port))

	if err = s.Serve(lis); err != nil {
		logger.Error("grpc server listen error", zap.Error(err))
	}
}
