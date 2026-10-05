package error

import (
	"errors"
	"fmt"
	"strings"
	"testing"
)

func TestRegister_DuplicatePanics(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("expected panic on duplicate code")
		}
	}()
	Register(ErrNotFound.Code, CategoryBusiness, "dup")
}

func TestLookup(t *testing.T) {
	if _, ok := Lookup(ErrNotFound.Code); !ok {
		t.Error("Lookup should find registered code")
	}
	if _, ok := Lookup(999999); ok {
		t.Error("Lookup should miss unregistered code")
	}
}

func TestError_Format(t *testing.T) {
	err := New(ErrNotFound).With("id", 7)

	for _, want := range []string{"[20001]", "资源不存在", "id=7", "common_test.go"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("Error() = %q, missing %q", err.Error(), want)
		}
	}
	if err.Caller() == "" {
		t.Error("caller should be captured")
	}
}

func TestNewf_Message(t *testing.T) {
	err := Newf(ErrInvalidParam, "field %s invalid", "email")
	if err.Message() != "field email invalid" {
		t.Errorf("Message = %q", err.Message())
	}
}

func TestWrap_Chain(t *testing.T) {
	cause := errors.New("sql: no rows")
	err := Wrap(ErrNotFound, cause).With("id", 7)

	if !errors.Is(err, ErrNotFound) {
		t.Error("errors.Is should match ErrNotFound")
	}
	if errors.Is(err, ErrConflict) {
		t.Error("should not match ErrConflict")
	}
	if !errors.Is(err, cause) {
		t.Error("errors.Is should traverse to cause")
	}
	if !strings.Contains(err.Error(), "sql: no rows") {
		t.Errorf("missing cause: %s", err)
	}
}

func TestWrap_NestedClassification(t *testing.T) {
	outer := Wrap(ErrInternal, Wrap(ErrNotFound, errors.New("x")))

	if !errors.Is(outer, ErrInternal) || !errors.Is(outer, ErrNotFound) {
		t.Error("nested errors.Is should match both definitions")
	}
	if CodeOf(outer) != ErrInternal.Code {
		t.Errorf("CodeOf = %d, want %d", CodeOf(outer), ErrInternal.Code)
	}
	if !IsCode(outer, ErrNotFound.Code) {
		t.Error("IsCode should find nested code")
	}
	if CategoryOf(outer) != CategorySystem {
		t.Errorf("CategoryOf = %v, want system", CategoryOf(outer))
	}
	if !IsCategory(outer, CategoryBusiness) {
		t.Error("IsCategory should find nested category")
	}
}

func TestFromError(t *testing.T) {
	if _, ok := FromError(nil); ok {
		t.Error("nil should not convert")
	}
	if _, ok := FromError(errors.New("plain")); ok {
		t.Error("foreign error should not convert")
	}
	if CodeOf(errors.New("plain")) != codeUnknown {
		t.Error("foreign error should map to unknown code")
	}
	if CategoryOf(errors.New("plain")) != CategoryUnknown {
		t.Error("foreign error should map to unknown category")
	}

	var e *Error
	wrapped := fmt.Errorf("ctx: %w", New(ErrInternal))
	if !errors.As(wrapped, &e) || e.Code() != ErrInternal.Code {
		t.Error("errors.As should find *Error through stdlib wrap")
	}
	if CodeOf(wrapped) != ErrInternal.Code {
		t.Error("CodeOf should traverse stdlib wrap")
	}
}

func TestWith_OddArgsAndField(t *testing.T) {
	err := New(ErrInternal).With("a", 1, "b", 2).With("dangling").WithField("c", 3)
	if len(err.Fields()) != 3 {
		t.Fatalf("fields = %v, want 3 entries", err.Fields())
	}
	if !strings.Contains(err.Error(), "a=1, b=2, c=3") {
		t.Errorf("fields should be sorted: %s", err.Error())
	}
	if err.Category() != CategorySystem || err.Definition().Code != ErrInternal.Code {
		t.Error("definition accessors mismatch")
	}
}
