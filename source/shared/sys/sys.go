package sys

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func CreateSysHandlers(server *http.Server) {
	// Канал для прослушивания сигналов ОС (Ctrl+C или команда от Docker)
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	go func() {
		log.Printf("Сервер запущен на %s\n", server.Addr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Ошибка запуска: %v", err)
		}
	}()

	// Ждём сигнала от ОС (блокируемся здесь)
	sig := <-stop
	log.Printf("Получен сигнал: %v. Начинаем плавную остановку...", sig)

	// Даём серверу 5 секунд на завершение текущих запросов
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("Ошибка при остановке сервера: %v", err)
	}
	log.Println("Сервер успешно остановлен.")
}
