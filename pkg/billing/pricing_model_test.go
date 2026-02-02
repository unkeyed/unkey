package billing

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCalculateCharges_FlatPricing(t *testing.T) {
	ctx := context.Background()
	service := &pricingModelService{}

	tests := []struct {
		name          string
		model         *PricingModel
		verifications int64
		want          int64
		wantErr       bool
	}{
		{
			name: "flat pricing with usage",
			model: &PricingModel{
				VerificationUnitPrice: 100, // $0.01
			},
			verifications: 1000,
			want:          100000, // 1000 * 100 = $1.00
		},
		{
			name: "zero usage",
			model: &PricingModel{
				VerificationUnitPrice: 100,
			},
			verifications: 0,
			want:          0,
		},
		{
			name: "only verifications",
			model: &PricingModel{
				VerificationUnitPrice: 100,
			},
			verifications: 500,
			want:          50000, // 500 * 100 = $0.50
		},
		{
			name:          "nil model",
			model:         nil,
			verifications: 100,
			wantErr:       true,
		},
		{
			name: "negative verifications",
			model: &PricingModel{
				VerificationUnitPrice: 100,
			},
			verifications: -1,
			wantErr:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := service.CalculateCharges(ctx, tt.model, tt.verifications, 0)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.want, got)
		})
	}
}

func TestCalculateCharges_TieredPricing(t *testing.T) {
	ctx := context.Background()
	service := &pricingModelService{}

	tests := []struct {
		name          string
		model         *PricingModel
		verifications int64
		want          int64
	}{
		{
			name: "tiered pricing - first tier only",
			model: &PricingModel{
				TieredPricing: &TieredPricing{
					Tiers: []PricingTier{
						{UpTo: 1000, UnitPrice: 100},
						{UpTo: 0, UnitPrice: 50}, // unlimited
					},
				},
			},
			verifications: 500,
			want:          50000, // 500 * 100 = $0.50
		},
		{
			name: "tiered pricing - multiple tiers",
			model: &PricingModel{
				TieredPricing: &TieredPricing{
					Tiers: []PricingTier{
						{UpTo: 1000, UnitPrice: 100},
						{UpTo: 0, UnitPrice: 50}, // unlimited
					},
				},
			},
			verifications: 1500,
			want:          125000, // (1000 * 100) + (500 * 50) = $1.25
		},
		{
			name: "tiered pricing - three tiers",
			model: &PricingModel{
				TieredPricing: &TieredPricing{
					Tiers: []PricingTier{
						{UpTo: 1000, UnitPrice: 100},
						{UpTo: 5000, UnitPrice: 50},
						{UpTo: 0, UnitPrice: 25}, // unlimited
					},
				},
			},
			verifications: 6000,
			want:          325000, // (1000 * 100) + (4000 * 50) + (1000 * 25) = $3.25
		},
		{
			name: "tiered pricing - zero usage",
			model: &PricingModel{
				TieredPricing: &TieredPricing{
					Tiers: []PricingTier{
						{UpTo: 1000, UnitPrice: 100},
						{UpTo: 0, UnitPrice: 50},
					},
				},
			},
			verifications: 0,
			want:          0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := service.CalculateCharges(ctx, tt.model, tt.verifications, 0)
			require.NoError(t, err)
			require.Equal(t, tt.want, got)
		})
	}
}

func TestValidateTieredPricing(t *testing.T) {
	tests := []struct {
		name    string
		pricing *TieredPricing
		wantErr bool
	}{
		{
			name: "valid tiered pricing",
			pricing: &TieredPricing{
				Tiers: []PricingTier{
					{UpTo: 1000, UnitPrice: 100},
					{UpTo: 0, UnitPrice: 50},
				},
			},
			wantErr: false,
		},
		{
			name: "empty tiers",
			pricing: &TieredPricing{
				Tiers: []PricingTier{},
			},
			wantErr: true,
		},
		{
			name: "tiers not in ascending order",
			pricing: &TieredPricing{
				Tiers: []PricingTier{
					{UpTo: 5000, UnitPrice: 100},
					{UpTo: 1000, UnitPrice: 50},
				},
			},
			wantErr: true,
		},
		{
			name: "unlimited tier not last",
			pricing: &TieredPricing{
				Tiers: []PricingTier{
					{UpTo: 0, UnitPrice: 100},
					{UpTo: 5000, UnitPrice: 50},
				},
			},
			wantErr: true,
		},
		{
			name: "negative unit price",
			pricing: &TieredPricing{
				Tiers: []PricingTier{
					{UpTo: 1000, UnitPrice: -100},
					{UpTo: 0, UnitPrice: 50},
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateTieredPricing(tt.pricing)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestPricingModelService_Validation(t *testing.T) {
	ctx := context.Background()
	service := &pricingModelService{}

	t.Run("rejects nil model in CalculateCharges", func(t *testing.T) {
		_, err := service.CalculateCharges(ctx, nil, 100, 0)
		require.Error(t, err)
	})

	t.Run("rejects negative verifications", func(t *testing.T) {
		model := &PricingModel{
			VerificationUnitPrice: 100,
		}
		_, err := service.CalculateCharges(ctx, model, -1, 0)
		require.Error(t, err)
	})
}
