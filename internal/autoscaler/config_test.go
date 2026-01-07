package autoscaler

import (
	"testing"
)

func TestScalingRule_Validate(t *testing.T) {
	tests := []struct {
		name    string
		rule    ScalingRule
		wantErr bool
	}{
		{
			name: "valid rule",
			rule: ScalingRule{
				TargetURN:       "urn:pulumi:stack::project::type::name",
				ConfigKey:       "count",
				Min:             1,
				Max:             10,
				CooldownSeconds: 60,
			},
			wantErr: false,
		},
		{
			name: "missing target urn",
			rule: ScalingRule{
				ConfigKey: "count",
				Min:       1,
				Max:       10,
			},
			wantErr: true,
		},
		{
			name: "missing config key",
			rule: ScalingRule{
				TargetURN: "urn:pulumi:stack::project::type::name",
				Min:       1,
				Max:       10,
			},
			wantErr: true,
		},
		{
			name: "min less than zero",
			rule: ScalingRule{
				TargetURN: "urn:pulumi:stack::project::type::name",
				ConfigKey: "count",
				Min:       -1,
				Max:       10,
			},
			wantErr: true,
		},
		{
			name: "max less than min",
			rule: ScalingRule{
				TargetURN: "urn:pulumi:stack::project::type::name",
				ConfigKey: "count",
				Min:       10,
				Max:       5,
			},
			wantErr: true,
		},
		{
			name: "negative cooldown",
			rule: ScalingRule{
				TargetURN:       "urn:pulumi:stack::project::type::name",
				ConfigKey:       "count",
				Min:             1,
				Max:             10,
				CooldownSeconds: -10,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.rule.Validate(); (err != nil) != tt.wantErr {
				t.Errorf("ScalingRule.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
