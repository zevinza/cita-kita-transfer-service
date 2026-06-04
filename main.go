// @title           Cita Kita Transfer Service API
// @version         1.0
// @description     Mini financial transfer API with atomicity, idempotency, and batch processing.
// @termsOfService  http://swagger.io/terms/

// @contact.name   API Support
// @contact.email  support@example.com

// @license.name  Apache 2.0
// @license.url   http://www.apache.org/licenses/LICENSE-2.0.html

// @host      localhost:8000
// @BasePath  /
// @schemes   http

package main

import (
	"context"
	"log"
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/swagger"
	"github.com/zevinza/cita-kita-transfer-service/controller"
	_ "github.com/zevinza/cita-kita-transfer-service/docs"
	"github.com/zevinza/cita-kita-transfer-service/internal/cache"
	"github.com/zevinza/cita-kita-transfer-service/internal/config"
	"github.com/zevinza/cita-kita-transfer-service/internal/logging"
	"github.com/zevinza/cita-kita-transfer-service/repository"
	"github.com/zevinza/cita-kita-transfer-service/service"
)

func init() {
	if err := config.Load(); err != nil {
		log.Fatalf("config: %v", err)
	}
}

func main() {
	app := fiber.New()

	logger := logging.NewLogger()
	rdb := cache.NewRedisClient()

	transactionRepository := repository.NewTransactionRepository(logger, rdb)
	transferService := service.NewTransferService(logger, transactionRepository)
	transferController := controller.NewTransferController(logger, transferService)

	seedUserRepository := repository.NewSeedUserRepository(logger, rdb)
	accountService := service.NewAccountService(logger, seedUserRepository)
	accountController := controller.NewAccountController(logger, accountService)

	if err := seedUserRepository.SeedUserBalances(context.Background()); err != nil {
		log.Fatalf("seed users: %v", err)
	}

	app.Get("/health", healthCheck)

	app.Get("/swagger/*", swagger.HandlerDefault)

	app.Get("/accounts", accountController.ListAccounts)
	app.Post("/transfer", transferController.Transfer)
	app.Post("/batch-transfer", transferController.BatchTransfer)

	log.Fatal(app.Listen(":" + strconv.Itoa(config.Get().Port)))
}

// healthCheck godoc
// @Summary      Health check
// @Description  Returns OK when the service is running
// @Tags         health
// @Produce      plain
// @Success      200 {string} string "OK"
// @Router       /health [get]
func healthCheck(c *fiber.Ctx) error {
	return c.SendString("OK")
}
