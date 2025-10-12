package api

import (
	"encoding/json"
	"strconv"

	"github.com/Egham-7/adaptive-proxy/internal/services/usage"
	"github.com/gofiber/fiber/v2"
)

type CreditsHandler struct {
	creditsService *usage.CreditsService
}

func NewCreditsHandler(creditsService *usage.CreditsService) *CreditsHandler {
	return &CreditsHandler{
		creditsService: creditsService,
	}
}

// GetBalanceResponse represents the response for balance queries
type GetBalanceResponse struct {
	OrganizationID string  `json:"organization_id"`
	Balance        float64 `json:"balance"`
	TotalPurchased float64 `json:"total_purchased"`
	TotalUsed      float64 `json:"total_used"`
}

// GetBalance retrieves the current credit balance for an organization
func (h *CreditsHandler) GetBalance(c *fiber.Ctx) error {
	organizationID := c.Params("organization_id")
	if organizationID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "organization_id is required",
		})
	}

	credit, err := h.creditsService.GetOrganizationCredit(c.Context(), organizationID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to get credit balance",
		})
	}

	return c.JSON(GetBalanceResponse{
		OrganizationID: credit.OrganizationID,
		Balance:        credit.Balance,
		TotalPurchased: credit.TotalPurchased,
		TotalUsed:      credit.TotalUsed,
	})
}

// CheckCreditsRequest represents the request body for checking credits
type CheckCreditsRequest struct {
	OrganizationID string  `json:"organization_id" binding:"required"`
	Amount         float64 `json:"amount" binding:"required,min=0"`
}

// CheckCreditsResponse represents the response for credit checks
type CheckCreditsResponse struct {
	HasEnoughCredits bool    `json:"has_enough_credits"`
	CurrentBalance   float64 `json:"current_balance"`
	RequiredAmount   float64 `json:"required_amount"`
	Shortfall        float64 `json:"shortfall,omitempty"`
}

// CheckCredits checks if an organization has sufficient credits
func (h *CreditsHandler) CheckCredits(c *fiber.Ctx) error {
	var req CheckCreditsRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	credit, err := h.creditsService.GetOrganizationCredit(c.Context(), req.OrganizationID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to get credit balance",
		})
	}

	hasEnough := credit.Balance >= req.Amount
	response := CheckCreditsResponse{
		HasEnoughCredits: hasEnough,
		CurrentBalance:   credit.Balance,
		RequiredAmount:   req.Amount,
	}

	if !hasEnough {
		response.Shortfall = req.Amount - credit.Balance
	}

	return c.JSON(response)
}

// GetTransactionHistoryResponse represents a transaction in the history
type GetTransactionHistoryResponse struct {
	Transactions []TransactionItem `json:"transactions"`
	Total        int               `json:"total"`
	Limit        int               `json:"limit"`
	Offset       int               `json:"offset"`
}

// TransactionItem represents a single transaction
type TransactionItem struct {
	ID             uint           `json:"id"`
	OrganizationID string         `json:"organization_id"`
	UserID         string         `json:"user_id"`
	Type           string         `json:"type"`
	Amount         float64        `json:"amount"`
	BalanceAfter   float64        `json:"balance_after"`
	Description    string         `json:"description"`
	Metadata       map[string]any `json:"metadata,omitempty"`
	CreatedAt      string         `json:"created_at"`
}

// GetTransactionHistory retrieves transaction history for an organization
func (h *CreditsHandler) GetTransactionHistory(c *fiber.Ctx) error {
	organizationID := c.Params("organization_id")
	if organizationID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "organization_id is required",
		})
	}

	limit := 20
	if limitStr := c.Query("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 100 {
			limit = l
		}
	}

	offset := 0
	if offsetStr := c.Query("offset"); offsetStr != "" {
		if o, err := strconv.Atoi(offsetStr); err == nil && o >= 0 {
			offset = o
		}
	}

	transactions, err := h.creditsService.GetTransactionHistory(c.Context(), organizationID, limit, offset)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to get transaction history",
		})
	}

	items := make([]TransactionItem, len(transactions))
	for i, tx := range transactions {
		var metadata map[string]any
		if tx.Metadata != "" {
			_ = json.Unmarshal([]byte(tx.Metadata), &metadata)
		}

		items[i] = TransactionItem{
			ID:             tx.ID,
			OrganizationID: tx.OrganizationID,
			UserID:         tx.UserID,
			Type:           string(tx.Type),
			Amount:         tx.Amount,
			BalanceAfter:   tx.BalanceAfter,
			Description:    tx.Description,
			Metadata:       metadata,
			CreatedAt:      tx.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		}
	}

	return c.JSON(GetTransactionHistoryResponse{
		Transactions: items,
		Total:        len(items),
		Limit:        limit,
		Offset:       offset,
	})
}
