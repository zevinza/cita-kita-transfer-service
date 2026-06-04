package controller

import (
	"github.com/gofiber/fiber/v2"
	"github.com/zevinza/cita-kita-transfer-service/internal/logging"
	"github.com/zevinza/cita-kita-transfer-service/service"
)

type AccountController interface {
	ListAccounts(c *fiber.Ctx) error
}

type accountController struct {
	logger         logging.Logger
	accountService service.AccountService
}

func NewAccountController(logger logging.Logger, accountService service.AccountService) AccountController {
	return &accountController{logger: logger, accountService: accountService}
}

// ListAccounts godoc
// @Summary      List accounts
// @Description  Returns all seeded users with their current balances
// @Tags         account
// @Produce      json
// @Success      200 {object} model.AccountListResponse
// @Failure      500 {object} model.ErrorResponse
// @Router       /accounts [get]
func (c *accountController) ListAccounts(ctx *fiber.Ctx) error {
	resp, err := c.accountService.ListAccounts(ctx.Context())
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return ctx.Status(fiber.StatusOK).JSON(resp)
}
