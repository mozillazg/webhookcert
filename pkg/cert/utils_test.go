package cert

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_removeDup(t *testing.T) {
	type args struct {
		items []string
	}
	tests := []struct {
		name string
		args args
		want []string
	}{
		{
			name: "nil -> nil",
			args: args{},
			want: nil,
		},
		{
			name: "empty -> nil",
			args: args{
				items: []string{},
			},
			want: nil,
		},
		{
			name: "no dup",
			args: args{
				items: []string{"foo", "bar", "foobar"},
			},
			want: []string{"foo", "bar", "foobar"},
		},
		{
			name: "dup",
			args: args{
				items: []string{"foo", "bar", "foo", "foobar", "bar", "foobar"},
			},
			want: []string{"foo", "bar", "foobar"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equalf(t, tt.want, removeDup(tt.args.items), "removeDup(%v)", tt.args.items)
		})
	}
}
