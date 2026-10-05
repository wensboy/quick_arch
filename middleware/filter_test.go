package middleware

import "testing"

func TestFilter_Allow(t *testing.T) {
	cases := []struct {
		name    string
		include []string
		exclude []string
		path    string
		want    bool
	}{
		{"empty allows all", nil, nil, "/api/v1/ping", true},
		{"include match", []string{"/api"}, nil, "/api/v1/ping", true},
		{"include miss", []string{"/admin"}, nil, "/api/v1/ping", false},
		{"exclude wins", []string{"/api"}, []string{"/api/v1/ping"}, "/api/v1/ping", false},
		{"exclude other path", []string{"/api"}, []string{"/api/v1/ping"}, "/api/v1/other", true},
	}

	for _, tc := range cases {
		filter := NewFilter(tc.include, tc.exclude)
		if got := filter.Allow(tc.path); got != tc.want {
			t.Errorf("%s: Allow(%q) = %v, want %v", tc.name, tc.path, got, tc.want)
		}
	}
}
