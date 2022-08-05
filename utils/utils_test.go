package utils

import "testing"

func TestValidateConfig(t *testing.T) {

	type Args struct {
		Name string              `required:"true"`
		Node map[string]struct{} `required:"true"`
	}

	expects := []struct {
		Title    string
		Expected bool
		args     Args
	}{
		{
			Title:    "Check normal",
			Expected: true,
			args: Args{
				Name: "test",
				Node: map[string]struct{}{
					"test": {},
				},
			},
		},
		{
			Title:    "Check required parma",
			Expected: false,
			args: Args{
				Name: "test2",
			},
		},
	}
	for _, tt := range expects {

		t.Run(tt.Title, func(t *testing.T) {
			if got := ValidateConfig(&tt.args); got != tt.Expected {
				t.Errorf("ValidateConfig() = %v, want %v", got, tt.Expected)
			}
		})
	}
}
