package app

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"shared/config"
	restController "shared/controller"
	"shared/sys"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type App[T any] struct {
	ORM        *gorm.DB
	Mux        *http.ServeMux
	Cfg        *config.Config
	Repository *restController.CRUDRepository[T]
	// Files — вторая таблица сервиса (файловые приложения сущностей,
	// см. controller/files.go). Одна на БД, одинаковая везде.
	Files *restController.FileHandler
}

func (a *App[T]) CreateConnection() {
	cfg, err := config.InitConfig()
	if err != nil {
		panic(fmt.Errorf("config error: %w", err))
	}
	a.Cfg = cfg

	dsn := fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		cfg.DB.Host, cfg.DB.Port, cfg.DB.User, cfg.DB.Password, cfg.DB.Name,
	)

	// FIX: убрано дублирование — только один customLogger
	customLogger := logger.New(
		log.New(os.Stdout, "\r\n", log.LstdFlags),
		logger.Config{
			SlowThreshold:             100 * time.Millisecond,
			LogLevel:                  logger.Warn, // FIX: Warn вместо Info — меньше шума
			IgnoreRecordNotFoundError: true,
			Colorful:                  true,
		},
	)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		PrepareStmt: true,
		Logger:      customLogger,
	})
	if err != nil {
		panic(fmt.Errorf("database connection error: %w", err))
	}

	// FIX: проверяем ошибку от db.DB()
	sqlDB, err := db.DB()
	if err != nil {
		panic(fmt.Errorf("failed to get sql.DB: %w", err))
	}
	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(100)
	sqlDB.SetConnMaxLifetime(time.Hour)

	a.ORM = db
	a.Repository = restController.NewCRUDRepository[T](a.ORM, a.Cfg.Server.TableName)
	// Таблица files — во всех БД одинаково (AutoMigrate покрывает и
	// существующие volume: initdb-скрипты для них уже не выполнятся).
	// Отдельного DDL в db.sh нет осознанно: схема одна, владеет shared.
	if err := restController.EnsureFileTable(a.ORM); err != nil {
		panic(fmt.Errorf("files table migrate error: %w", err))
	}
	a.Files = restController.NewFileHandler(a.ORM, a.Cfg.Server.TableName)
}

func (a *App[T]) CreateRoutes() {
	mux := http.NewServeMux()
	basePath := "/api/" + a.Cfg.Server.TableName

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Service is running"))
	})
	mux.HandleFunc(basePath, a.Repository.HandleRequest)
	// Файлы — на двух путях: /api/files (прямой доступ по IP контейнера)
	// и /api/<ref>/files (через traefik: PathPrefix префикс НЕ срезает,
	// сервис видит полный путь; пайплайны ходят как end_point "<ref>/files").
	mux.HandleFunc("/api/files", a.Files.HandleRequest)
	mux.HandleFunc("/api/"+a.Cfg.Server.TableName+"/files", a.Files.HandleRequest)
	// Фоновая чистка файлов-сирот (лёгкий тикер; планировщика в стенде нет).
	a.Files.StartFileGC()
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"healthy"}`))
	})

	a.Mux = mux
	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", a.Cfg.Server.Port),
		Handler:      mux,
		ReadTimeout:  30 * time.Second, // FIX: добавлены таймауты
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	fmt.Printf("Server starting on %s, endpoint: %s\n", server.Addr, basePath)
	sys.CreateSysHandlers(server)
}
