package model

import (
	"testing"
)

func TestPrices_EffectivePrice_NoDiscount(t *testing.T) {
	p := Prices{Price: 100.0}
	if got := p.EffectivePrice(); got != 100.0 {
		t.Errorf("expected 100.0, got %v", got)
	}
}

func TestPrices_EffectivePrice_WithDiscount(t *testing.T) {
	disc := 79.90
	p := Prices{Price: 100.0, DiscountPrice: &disc}
	if got := p.EffectivePrice(); got != 79.90 {
		t.Errorf("expected 79.90, got %v", got)
	}
}

func TestPrices_EffectivePrice_ZeroDiscount(t *testing.T) {
	zero := 0.0
	p := Prices{Price: 100.0, DiscountPrice: &zero}
	// zero discount should fall back to regular price
	if got := p.EffectivePrice(); got != 100.0 {
		t.Errorf("expected 100.0, got %v", got)
	}
}
