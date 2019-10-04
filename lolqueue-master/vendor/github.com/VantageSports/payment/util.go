package payment

import (
	"errors"
	"fmt"

	"github.com/stripe/stripe-go"
)

// Event is a custom struct for getting a stripe event
// and being able to call some methods on it for retrieving data
type Event struct {
	Data *stripe.Event `json:"data"`
}

// Plan is a custom struct for getting a stripe Plan
// and being able to call some methods on it for retrieving data
type Plan struct {
	Data map[string]interface{} `json:"data"`
}

func (c *CustomerRequest) Valid() error {
	if c.CouponId == "" || c.UserId == "" {
		return errors.New("request requires coupon_id and user_id")
	}
	return nil
}

func (p *PurchaseRequest) Valid() error {
	if p.SkuId == "" || p.UserId == "" {
		return errors.New("request requires sku_id and user_id")
	}
	return nil
}

func (p *SubscriptionRequest) Valid() error {
	if p.Plan == "" || p.UserId == "" {
		return errors.New("request requires plan and user_id")
	}
	return nil
}

func (p *SubscriptionsListRequest) Valid() error {
	if p.UserId == "" {
		return errors.New("request requires user_id")
	}
	return nil
}

func (ps *PaymentSourceRequest) Valid() error {
	if ps.UserId == "" || ps.StripeToken == "" {
		return errors.New("request requires user_id, and stripe_token")
	}
	return nil
}

func (c *PaymentSourceFilter) Valid() error {
	if c.UserId == "" {
		return errors.New("request requires user id")
	}
	return nil
}

// Plan retrieves the plan from the event
func (e Event) Plan() (*Plan, error) {
	data := e.Data.Data.Obj
	if plan, ok := data["plan"]; !ok {
		return nil, fmt.Errorf("no plan found with subscription, event id: %s", e.Data.ID)
	} else {
		pl, ok := plan.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("no plan found with subscription, event id: %s", e.Data.ID)
		}
		return &Plan{Data: pl}, nil
	}
}

// GetMetaData retrieves the metadata for the plan if it exists
// for the passed in key
func (p Plan) GetMetaData(key string) string {
	metadata, found := p.Data["metadata"]
	if found {
		md := metadata.(map[string]interface{})
		if value, ok := md[key].(string); ok {
			return value
		}
	}
	return ""
}
