package nostr

import (
	"reflect"
	"testing"
	"time"
)

func TestFilterToRequest(t *testing.T) {
	since := time.Unix(100, 0)
	until := since.Add(time.Hour)

	tests := []struct {
		name   string
		filter Filter
		want   map[string]any
	}{
		{
			name:   "empty filter produces empty payload",
			filter: Filter{},
			want:   map[string]any{},
		},
		{
			name: "authors and kinds only",
			filter: Filter{
				Authors: []string{"author1", "author2"},
				Kinds:   []int{1, 2},
			},
			want: map[string]any{
				"authors": []string{"author1", "author2"},
				"kinds":   []int{1, 2},
			},
		},
		{
			name: "all fields populated",
			filter: Filter{
				Authors: []string{"pub"},
				Kinds:   []int{KindTextNote},
				Since:   &since,
				Until:   &until,
				Limit:   42,
			},
			want: map[string]any{
				"authors": []string{"pub"},
				"kinds":   []int{KindTextNote},
				"since":   since.Unix(),
				"until":   until.Unix(),
				"limit":   42,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.filter.toRequest()
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("toRequest() = %#v, want %#v", got, tt.want)
			}
		})
	}
}
