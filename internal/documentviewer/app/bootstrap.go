// Package app is the document-viewer composition root (R6): it constructs
// adapters and wires them to ports for this binary only.
package app

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/ducnd58233/unified-document-viewer/configs"
	audituc "github.com/ducnd58233/unified-document-viewer/internal/documentviewer/modules/audit/app/usecases"
	auditpersist "github.com/ducnd58233/unified-document-viewer/internal/documentviewer/modules/audit/infra/persistence"
	docsapi "github.com/ducnd58233/unified-document-viewer/internal/documentviewer/modules/documents/api"
	docsapp "github.com/ducnd58233/unified-document-viewer/internal/documentviewer/modules/documents/app"
	docusecases "github.com/ducnd58233/unified-document-viewer/internal/documentviewer/modules/documents/app/usecases"
	docsupstream "github.com/ducnd58233/unified-document-viewer/internal/documentviewer/modules/documents/infra/http"
	docpersist "github.com/ducnd58233/unified-document-viewer/internal/documentviewer/modules/documents/infra/persistence"
	"github.com/ducnd58233/unified-document-viewer/internal/shared/infra/circuitbreaker"
	"github.com/ducnd58233/unified-document-viewer/internal/shared/infra/httpserver"
	"github.com/ducnd58233/unified-document-viewer/internal/shared/infra/lifecycle"
	"github.com/ducnd58233/unified-document-viewer/internal/shared/infra/postgres"
	"github.com/ducnd58233/unified-document-viewer/internal/shared/observability"
)

// Run is the composition root; cmd/ only handles signals (R6).
func Run(ctx context.Context) error {
	cfg, err := configs.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	logger, logClose, err := observability.NewLogger(observability.Options{
		Service: "documentviewer",
		Level:   cfg.Log.Level,
	})
	if err != nil {
		return fmt.Errorf("logger: %w", err)
	}
	slog.SetDefault(logger)

	var pool *postgres.Pool
	lc := lifecycle.New(logger)
	lc.Add(lifecycle.Hook{
		Name: "postgres",
		Start: func(ctx context.Context) error {
			p, err := postgres.Open(ctx, cfg.Database.URL, cfg.Database.MaxConns)
			if err != nil {
				return err
			}
			pool = p
			logger.Info("postgres ready")
			return nil
		},
		Stop: func(context.Context) error {
			logger.Info("postgres closing")
			if pool != nil {
				pool.Close()
			}
			return nil
		},
	})
	if err := lc.Start(ctx); err != nil {
		_ = logClose.Close()
		return err
	}
	defer func() {
		lc.Stop(context.WithoutCancel(ctx))
		_ = logClose.Close()
	}()

	cacheRepo := docpersist.NewCacheRepository(pool, nil)
	auditRepo := auditpersist.NewAuditRepository(pool)
	recordAccess := audituc.NewRecordAccess(auditRepo, cfg.Log.VINHashSalt, nil)

	agg := docusecases.New(docusecases.Options{
		Sources: []docsapp.DocumentSource{
			docsupstream.WithBreaker(
				docsupstream.NewSalesClient(cfg.Sources.SalesBaseURL),
				circuitbreaker.New(circuitbreaker.Settings{Name: "sales", Log: logger}),
			),
			docsupstream.WithBreaker(
				docsupstream.NewServiceClient(cfg.Sources.ServiceBaseURL),
				circuitbreaker.New(circuitbreaker.Settings{Name: "service", Log: logger}),
			),
		},
		DocumentCache:    cacheRepo,
		PerSourceTimeout: cfg.Sources.PerSourceTimeout,
		AggregateTimeout: cfg.Sources.AggregateTimeout,
		CacheTTL:         cfg.Cache.TTL,
	})

	docsHandler := docsapi.New(agg, newAccessLog(recordAccess, logger), logger)
	handler := mountHTTP(httpDeps{
		log:            logger,
		docs:           docsHandler,
		requestTimeout: cfg.HTTP.RequestTimeout,
	})

	return httpserver.Serve(ctx, cfg.HTTP.Address, handler, logger)
}
