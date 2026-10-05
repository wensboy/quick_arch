package model

import (
	"encoding/json"
	"testing"
)

func TestNewPage_Fields(t *testing.T) {
	page := NewPage(20, 10, 42, []string{"a", "b", "c"})

	if page.Total != 42 || page.Base != 20 || page.Count != 10 {
		t.Fatalf("page = %+v", page)
	}
	if page.ExactCount != 3 {
		t.Fatalf("exact_count = %d, want 3", page.ExactCount)
	}
	if len(page.Items) != 3 {
		t.Fatalf("items = %v", page.Items)
	}
}

func TestNewPage_NilItemsBecomesEmptyArray(t *testing.T) {
	b, err := json.Marshal(NewPage[int](0, 10, 0, nil))
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	want := `{"total":0,"base":0,"count":10,"exact_count":0,"items":[]}`
	if string(b) != want {
		t.Fatalf("json = %s, want %s", b, want)
	}
}

func TestPage_OffsetLimit(t *testing.T) {
	page := NewPage(40, 20, 100, []int{1})

	if page.Offset() != 40 || page.Limit() != 20 {
		t.Fatalf("offset/limit = %d/%d, want 40/20", page.Offset(), page.Limit())
	}
}

func TestPage_EmptyAndHasMore(t *testing.T) {
	cases := []struct {
		name        string
		page        Page[int]
		wantEmpty   bool
		wantHasMore bool
	}{
		{"empty", NewPage[int](0, 10, 0, nil), true, false},
		{"more", NewPage(0, 2, 5, []int{1, 2}), false, true},
		{"last page", NewPage(4, 2, 5, []int{5}), false, false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.page.Empty(); got != tc.wantEmpty {
				t.Errorf("Empty = %v, want %v", got, tc.wantEmpty)
			}
			if got := tc.page.HasMore(); got != tc.wantHasMore {
				t.Errorf("HasMore = %v, want %v", got, tc.wantHasMore)
			}
		})
	}
}

func TestPage_WrappedInResponse(t *testing.T) {
	resp := Success(NewPage(0, 2, 2, []int{1, 2}))

	b, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	want := `{"code":0,"message":"","data":{"total":2,"base":0,"count":2,"exact_count":2,"items":[1,2]}}`
	if string(b) != want {
		t.Fatalf("json = %s, want %s", b, want)
	}
}
