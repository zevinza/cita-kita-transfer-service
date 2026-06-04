package controller

import (
	"github.com/gofiber/fiber/v2"
	"github.com/zevinza/cita-kita-transfer-service/internal/logging"
	"github.com/zevinza/cita-kita-transfer-service/service"
)

type TransferController interface {
	Transfer(c *fiber.Ctx) error
	BatchTransfer(c *fiber.Ctx) error
}

type transferController struct {
	logger          logging.Logger
	transferService service.TransferService
}

func NewTransferController(logger logging.Logger, transferService service.TransferService) TransferController {
	return &transferController{logger: logger, transferService: transferService}
}

func (c *transferController) Transfer(ctx *fiber.Ctx) error {
	return nil
}

func (c *transferController) BatchTransfer(ctx *fiber.Ctx) error {
	return nil
}
