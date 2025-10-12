package usage

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/Egham-7/adaptive-proxy/internal/models"
	"github.com/stripe/stripe-go/v81"
	"github.com/stripe/stripe-go/v81/checkout/session"
	"github.com/stripe/stripe-go/v81/webhook"
)

type StripeService struct {
	secretKey      string
	webhookSecret  string
	creditsService *CreditsService
}

type StripeConfig struct {
	SecretKey     string
	WebhookSecret string
}

func NewStripeService(cfg StripeConfig, creditsService *CreditsService) *StripeService {
	stripe.Key = cfg.SecretKey

	return &StripeService{
		secretKey:      cfg.SecretKey,
		webhookSecret:  cfg.WebhookSecret,
		creditsService: creditsService,
	}
}

// CreateCheckoutSession creates a Stripe checkout session for purchasing credits
func (s *StripeService) CreateCheckoutSession(ctx context.Context, params CreateCheckoutParams) (*stripe.CheckoutSession, error) {
	sessionParams := &stripe.CheckoutSessionParams{
		Mode: stripe.String(string(stripe.CheckoutSessionModePayment)),
		LineItems: []*stripe.CheckoutSessionLineItemParams{
			{
				Price:    stripe.String(params.StripePriceID),
				Quantity: stripe.Int64(1),
			},
		},
		SuccessURL: stripe.String(params.SuccessURL),
		CancelURL:  stripe.String(params.CancelURL),
		Metadata: map[string]string{
			"organization_id": params.OrganizationID,
			"user_id":         params.UserID,
			"credit_amount":   fmt.Sprintf("%.2f", params.CreditAmount),
		},
	}

	if params.CustomerEmail != "" {
		sessionParams.CustomerEmail = stripe.String(params.CustomerEmail)
	}

	sess, err := session.New(sessionParams)
	if err != nil {
		return nil, fmt.Errorf("failed to create checkout session: %w", err)
	}

	return sess, nil
}

// HandleWebhook processes Stripe webhook events
func (s *StripeService) HandleWebhook(ctx context.Context, payload []byte, signature string) error {
	event, err := webhook.ConstructEvent(payload, signature, s.webhookSecret)
	if err != nil {
		return fmt.Errorf("failed to verify webhook signature: %w", err)
	}

	switch event.Type {
	case "checkout.session.completed":
		return s.handleCheckoutSessionCompleted(ctx, event)
	case "payment_intent.succeeded":
		return s.handlePaymentIntentSucceeded(ctx, event)
	case "payment_intent.payment_failed":
		return s.handlePaymentIntentFailed(ctx, event)
	default:
		// Ignore other event types
		return nil
	}
}

// handleCheckoutSessionCompleted processes successful checkout sessions
func (s *StripeService) handleCheckoutSessionCompleted(ctx context.Context, event stripe.Event) error {
	var session stripe.CheckoutSession
	if err := json.Unmarshal(event.Data.Raw, &session); err != nil {
		return fmt.Errorf("failed to parse checkout session: %w", err)
	}

	// Extract metadata
	organizationID := session.Metadata["organization_id"]
	userID := session.Metadata["user_id"]
	creditAmount := 0.0
	if _, err := fmt.Sscanf(session.Metadata["credit_amount"], "%f", &creditAmount); err != nil {
		return fmt.Errorf("failed to parse credit amount: %w", err)
	}

	if organizationID == "" || userID == "" || creditAmount <= 0 {
		return fmt.Errorf("invalid checkout session metadata")
	}

	// Add credits to organization
	metadataMap := map[string]any{
		"stripe_session_id":        session.ID,
		"stripe_payment_intent_id": session.PaymentIntent,
		"amount_paid":              float64(session.AmountTotal) / 100.0, // Convert from cents
	}
	metadataJSON, err := json.Marshal(metadataMap)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	description := fmt.Sprintf("Credit purchase via Stripe (%.2f credits)", creditAmount)
	metadataStr := string(metadataJSON)
	paymentIntentID := session.PaymentIntent.ID
	sessionID := session.ID

	_, err = s.creditsService.AddCredits(ctx, models.AddCreditsParams{
		OrganizationID:        organizationID,
		UserID:                userID,
		Amount:                creditAmount,
		Type:                  models.CreditTransactionPurchase,
		Description:           &description,
		Metadata:              &metadataStr,
		StripePaymentIntentID: &paymentIntentID,
		StripeSessionID:       &sessionID,
	})
	if err != nil {
		return fmt.Errorf("failed to add credits: %w", err)
	}

	return nil
}

// handlePaymentIntentSucceeded processes successful payment intents
func (s *StripeService) handlePaymentIntentSucceeded(ctx context.Context, event stripe.Event) error {
	// This is already handled by checkout.session.completed
	// But we can add additional logging or processing here if needed
	return nil
}

// handlePaymentIntentFailed processes failed payment intents
func (s *StripeService) handlePaymentIntentFailed(ctx context.Context, event stripe.Event) error {
	// Log the failure, potentially notify the user
	// For now, just acknowledge the event
	return nil
}

// CreateCheckoutParams contains parameters for creating a checkout session
type CreateCheckoutParams struct {
	OrganizationID string
	UserID         string
	StripePriceID  string
	CreditAmount   float64
	SuccessURL     string
	CancelURL      string
	CustomerEmail  string
}
