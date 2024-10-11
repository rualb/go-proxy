package toolconfig

import (
	"os"
	"testing"
)

func Test_expandEnv(t *testing.T) {

	os.Setenv("QWE1", "123")

	type args struct {
		data string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{"Test-1", args{`$${QWE1} $QWE ${1}\n${QWE1}`}, `$123 $QWE ${1}\n123`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := expandEnv(tt.args.data); got != tt.want {
				t.Errorf("expandEnv() = %v, want %v", got, tt.want)
			}
		})
	}
}
