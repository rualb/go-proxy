package config

import (
	"os"
	"testing"
)

func Test_envReader_StringArray(t *testing.T) {

	_ = os.Setenv("APP_QWE", `["1","2","3","4","5"]`)

	type args struct {
		p        *[]string
		name     string
		cmdValue *[]string
	}
	tests := []struct {
		name string
		x    envReader
		args args
	}{
		{name: "Test-1", x: NewEnvReader(),
			args: args{p: &[]string{"0"}, name: "-", cmdValue: &[]string{"1", "2", "3"}}},
		{name: "Test-2", x: NewEnvReader(),
			args: args{p: &[]string{"0"}, name: "qwe", cmdValue: &[]string{}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			tt.x.StringArray(tt.args.p, tt.args.name, tt.args.cmdValue)

			if (*tt.args.p)[0] == "0" {
				t.Errorf("expected non-default value, got %v", (*tt.args.p)[0])
			}
		})
	}
}
