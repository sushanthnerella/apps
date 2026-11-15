package main

import (
	"net/url"
	"testing"
)

func TestVendorSearchLocation(t *testing.T) {
	tests := []struct {
		query string
		ok    bool
	}{{"", false}, {"lat=17&lng=78", true}, {"lat=0&lng=0", true}, {"lat=91&lng=78", false}, {"lat=17&lng=181", false}, {"lat=NaN&lng=78", false}, {"lat=17&lng=78&radius=-1", false}}
	for _, tt := range tests {
		q, _ := url.ParseQuery(tt.query)
		_, _, _, err := parseVendorSearch(q)
		if (err == nil) != tt.ok {
			t.Errorf("%q: error = %v", tt.query, err)
		}
	}
}
