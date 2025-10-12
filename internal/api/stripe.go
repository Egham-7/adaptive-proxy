package api

import (
	"io"

	"github.com/Egham-7/adaptive-proxy/internal/services/usage"
	"github.com/gofiber/fiber/v2"
)

type StripeHandler struct {
	stripeService *usage.StripeService
}

func NewStripeHandler(stripeService *usage.StripeService) *StripeHandler {
	return &StripeHandler{
		stripeService: stripeService,
	}
}

// CreateCheckoutSessionRequest represents the request body for creating a checkout session
type CreateCheckoutSessionRequest struct {
	OrganizationID string  `json:"organization_id" binding:"required"`
	UserID         string  `json:"user_id" binding:"required"`
	StripePriceID  string  `json:"stripe_price_id" binding:"required"`
	CreditAmount   float64 `json:"credit_amount" binding:"required,min=0"`
	SuccessURL     string  `json:"success_url" binding:"required"`
	CancelURL      string  `json:"cancel_url" binding:"required"`
	CustomerEmail  string  `json:"customer_email,omitempty"`
}

// CreateCheckoutSessionResponse represents the response for checkout session creation
type CreateCheckoutSessionResponse struct {
	SessionID   string  `json:"session_id"`
	CheckoutURL string  `json:"checkout_url"`
	Amount      float64 `json:"amount"`
}

// CreateCheckoutSession creates a Stripe checkout session for purchasing credits
func (h *StripeHandler) CreateCheckoutSession(c *fiber.Ctx) error {
	var req CreateCheckoutSessionRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	// Validate required fields
	if req.OrganizationID == "" || req.UserID == "" || req.StripePriceID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "organization_id, user_id, and stripe_price_id are required",
		})
	}

	if req.CreditAmount <= 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "credit_amount must be greater than 0",
		})
	}

	if req.SuccessURL == "" || req.CancelURL == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "success_url and cancel_url are required",
		})
	}

	// Create checkout session
	session, err := h.stripeService.CreateCheckoutSession(c.Context(), usage.CreateCheckoutParams{
		OrganizationID: req.OrganizationID,
		UserID:         req.UserID,
		StripePriceID:  req.StripePriceID,
		CreditAmount:   req.CreditAmount,
		SuccessURL:     req.SuccessURL,
		CancelURL:      req.CancelURL,
		CustomerEmail:  req.CustomerEmail,
	})
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to create checkout session",
		})
	}

	return c.JSON(CreateCheckoutSessionResponse{
		SessionID:   session.ID,
		CheckoutURL: session.URL,
		Amount:      req.CreditAmount,
	})
}

// HandleWebhook processes Stripe webhook events
func (h *StripeHandler) HandleWebhook(c *fiber.Ctx) error {
	// Get the request body
	payload, err := io.ReadAll(c.Context().RequestBodyStream())
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Failed to read request body",
		})
	}

	// Get the Stripe signature header
	signature := c.Get("Stripe-Signature")
	if signature == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Missing Stripe-Signature header",
		})
	}

	// Process the webhook
	if err := h.stripeService.HandleWebhook(c.Context(), payload, signature); err != nil {
		// Check if it's a signature verification error
		if err.Error() == "failed to verify webhook signature" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Invalid webhook signature",
			})
		}

		// For other errors, return internal server error
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to process webhook",
		})
	}

	// Return 200 OK to acknowledge receipt
	return c.JSON(fiber.Map{
		"received": true,
	})
}
