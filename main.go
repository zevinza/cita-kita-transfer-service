package main

import (
	"log"
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/swagger"
	"github.com/zevinza/cita-kita-transfer-service/controller"
	"github.com/zevinza/cita-kita-transfer-service/internal/config"
	"github.com/zevinza/cita-kita-transfer-service/internal/logging"
	"github.com/zevinza/cita-kita-transfer-service/repository"
	"github.com/zevinza/cita-kita-transfer-service/service"
)

func init() {
	config.Load()
}

func main() {
	app := fiber.New()

	logger := logging.NewLogger()

	transactionRepository := repository.NewTransactionRepository(logger)
	transferService := service.NewTransferService(logger, transactionRepository)
	transferController := controller.NewTransferController(logger, transferService)

	app.Get("/health", func(c *fiber.Ctx) error {
		return c.SendString("OK")
	})

	app.Get("/swagger/*", swagger.HandlerDefault)

	app.Post("/transfer", transferController.Transfer)
	app.Post("/batch-transfer", transferController.BatchTransfer)

	log.Fatal(app.Listen(":" + strconv.Itoa(config.Get().Port)))
}
