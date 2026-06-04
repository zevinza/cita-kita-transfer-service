package controller

import (
	"github.com/gofiber/fiber/v2"
	"github.com/zevinza/cita-kita-transfer-service/internal/logging"
	"github.com/zevinza/cita-kita-transfer-service/model"
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

// Transfer godoc
// @Summary      Transfer funds between accounts
// @Description  Transfers amount from one account to another. Idempotent by idempotency_key.
// @Tags         transfer
// @Accept       json
// @Produce      json
// @Param        request body model.TransferRequest true "Transfer request"
// @Success      200 {object} model.TransferResponse
// @Failure      400 {object} model.ErrorResponse
// @Failure      500 {object} model.ErrorResponse
// @Router       /transfer [post]
func (c *transferController) Transfer(ctx *fiber.Ctx) error {
	request := new(model.TransferRequest)
	if err := ctx.BodyParser(request); err != nil {
		c.logger.Warn(ctx.Context(), "transfer request invalid", "error", err)
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid request",
		})
	}

	resp, err := c.transferService.Transfer(ctx.Context(), request)
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return ctx.Status(fiber.StatusOK).JSON(resp)
}

// BatchTransfer godoc
// @Summary      Batch transfer funds
// @Description  Processes multiple transfers concurrently. Each item is idempotent by idempotency_key.
// @Tags         transfer
// @Accept       json
// @Produce      json
// @Param        request body []model.TransferRequest true "Batch transfer requests"
// @Success      200 {object} model.BatchTransferResponse
// @Failure      400 {object} model.ErrorResponse
// @Failure      500 {object} model.ErrorResponse
// @Router       /batch-transfer [post]
func (c *transferController) BatchTransfer(ctx *fiber.Ctx) error {
	requests := new([]model.TransferRequest)
	if err := ctx.BodyParser(requests); err != nil {
		c.logger.Warn(ctx.Context(), "batch transfer request invalid", "error", err)
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid request",
		})
	}

	resp, err := c.transferService.BatchTransfer(ctx.Context(), *requests)
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return ctx.Status(fiber.StatusOK).JSON(resp)
}
