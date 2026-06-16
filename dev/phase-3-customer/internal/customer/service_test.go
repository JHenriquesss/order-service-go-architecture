package customer

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"

	apperrors "order-service-customer/internal/errors"
)

func newService() *Service {
	return NewService(NewInMemoryRepository())
}

func mustCreate(t *testing.T, s *Service, name, document string) *CustomerOutput {
	t.Helper()
	out, err := s.Create(context.Background(), CreateCustomerInput{Name: name, Document: document})
	if err != nil {
		t.Fatalf("Create(%q,%q) unexpected error: %v", name, document, err)
	}
	return out
}

func assertCode(t *testing.T, err error, want apperrors.Code) {
	t.Helper()
	var appErr *apperrors.AppError
	if !errors.As(err, &appErr) {
		t.Fatalf("expected *AppError, got %T: %v", err, err)
	}
	if appErr.Code != want {
		t.Fatalf("expected code %s, got %s", want, appErr.Code)
	}
}

// BR-CUS-001: name required.
func TestCreateRequiresName(t *testing.T) {
	s := newService()
	_, err := s.Create(context.Background(), CreateCustomerInput{Name: "  ", Document: "DOC-1"})
	if err == nil {
		t.Fatal("expected validation error for missing name")
	}
	assertCode(t, err, apperrors.CodeValidation)
}

func TestCreateValidReturnsCustomer(t *testing.T) {
	s := newService()
	out := mustCreate(t, s, "ACME Ltd", "12345678000199")
	if out.ID == uuid.Nil {
		t.Fatal("expected a generated id")
	}
	if !out.Active {
		t.Fatal("new customer must be active")
	}
	if out.Name != "ACME Ltd" || out.Document != "12345678000199" {
		t.Fatalf("unexpected output: %+v", out)
	}
}

// BR-CUS-002: document must be unique.
func TestCreateDuplicateDocument(t *testing.T) {
	s := newService()
	mustCreate(t, s, "ACME Ltd", "DOC-DUP")
	_, err := s.Create(context.Background(), CreateCustomerInput{Name: "Other", Document: "DOC-DUP"})
	if err == nil {
		t.Fatal("expected duplicate error")
	}
	assertCode(t, err, apperrors.CodeDuplicate)
}

func TestUpdateChangesMutableFields(t *testing.T) {
	s := newService()
	created := mustCreate(t, s, "Old Name", "DOC-UPD")
	out, err := s.Update(context.Background(), created.ID, UpdateCustomerInput{
		Name:     "New Name",
		Document: "DOC-UPD",
		Email:    "new@example.com",
		Phone:    "+55 11 90000-0000",
	})
	if err != nil {
		t.Fatalf("Update unexpected error: %v", err)
	}
	if out.Name != "New Name" || out.Email != "new@example.com" || out.Phone != "+55 11 90000-0000" {
		t.Fatalf("mutable fields not updated: %+v", out)
	}
}

func TestUpdateUnknownReturnsNotFound(t *testing.T) {
	s := newService()
	_, err := s.Update(context.Background(), uuid.New(), UpdateCustomerInput{Name: "X", Document: "Y"})
	assertCode(t, err, apperrors.CodeNotFound)
}

func TestUpdateDuplicateDocument(t *testing.T) {
	s := newService()
	mustCreate(t, s, "A", "DOC-A")
	b := mustCreate(t, s, "B", "DOC-B")
	_, err := s.Update(context.Background(), b.ID, UpdateCustomerInput{Name: "B", Document: "DOC-A"})
	assertCode(t, err, apperrors.CodeDuplicate)
}

// BR-CUS-004: deactivate is soft; the record remains retrievable.
func TestDeactivateIsSoftAndRetrievable(t *testing.T) {
	s := newService()
	created := mustCreate(t, s, "ACME", "DOC-DEACT")
	if err := s.Deactivate(context.Background(), created.ID); err != nil {
		t.Fatalf("Deactivate unexpected error: %v", err)
	}
	got, err := s.FindByID(context.Background(), created.ID)
	if err != nil {
		t.Fatalf("customer must still be retrievable after deactivate: %v", err)
	}
	if got.Active {
		t.Fatal("deactivated customer must have active=false")
	}
}

func TestDeactivateUnknownReturnsNotFound(t *testing.T) {
	s := newService()
	err := s.Deactivate(context.Background(), uuid.New())
	assertCode(t, err, apperrors.CodeNotFound)
}

func TestFindByIDUnknownReturnsNotFound(t *testing.T) {
	s := newService()
	_, err := s.FindByID(context.Background(), uuid.New())
	assertCode(t, err, apperrors.CodeNotFound)
}

func TestFindByIDReturnsRightCustomer(t *testing.T) {
	s := newService()
	mustCreate(t, s, "First", "DOC-1")
	want := mustCreate(t, s, "Second", "DOC-2")
	got, err := s.FindByID(context.Background(), want.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.ID != want.ID || got.Name != "Second" {
		t.Fatalf("wrong customer returned: %+v", got)
	}
}

func TestListFiltersByName(t *testing.T) {
	s := newService()
	mustCreate(t, s, "ACME Ltd", "DOC-1")
	mustCreate(t, s, "Globex", "DOC-2")
	page, err := s.List(context.Background(), CustomerFilter{Name: "acme"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if page.Total != 1 || len(page.Items) != 1 || page.Items[0].Name != "ACME Ltd" {
		t.Fatalf("name filter wrong: %+v", page)
	}
}

func TestListFiltersByDocument(t *testing.T) {
	s := newService()
	mustCreate(t, s, "ACME", "AAA-111")
	mustCreate(t, s, "Globex", "BBB-222")
	page, err := s.List(context.Background(), CustomerFilter{Document: "bbb"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if page.Total != 1 || page.Items[0].Document != "BBB-222" {
		t.Fatalf("document filter wrong: %+v", page)
	}
}

func TestListFiltersByActive(t *testing.T) {
	s := newService()
	a := mustCreate(t, s, "Active One", "DOC-1")
	mustCreate(t, s, "Active Two", "DOC-2")
	if err := s.Deactivate(context.Background(), a.ID); err != nil {
		t.Fatalf("deactivate: %v", err)
	}
	inactive := false
	page, err := s.List(context.Background(), CustomerFilter{Active: &inactive})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if page.Total != 1 || page.Items[0].ID != a.ID {
		t.Fatalf("active filter wrong: %+v", page)
	}
}

func TestListPaginates(t *testing.T) {
	s := newService()
	for i := 0; i < 5; i++ {
		mustCreate(t, s, "Customer", uuid.NewString())
	}
	page, err := s.List(context.Background(), CustomerFilter{Page: 2, PageSize: 2})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if page.Total != 5 || page.TotalPages != 3 || len(page.Items) != 2 || page.Page != 2 {
		t.Fatalf("pagination wrong: total=%d totalPages=%d items=%d page=%d",
			page.Total, page.TotalPages, len(page.Items), page.Page)
	}
}

func TestListDefaultsPageAndSize(t *testing.T) {
	s := newService()
	mustCreate(t, s, "Only", "DOC-1")
	page, err := s.List(context.Background(), CustomerFilter{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if page.Page != 1 || page.PageSize != DefaultPageSize {
		t.Fatalf("expected default page=1 size=%d, got page=%d size=%d",
			DefaultPageSize, page.Page, page.PageSize)
	}
}

func TestListRejectsInvalidPagination(t *testing.T) {
	s := newService()
	cases := []CustomerFilter{
		{Page: -1, PageSize: 10},
		{Page: 1, PageSize: -5},
		{Page: 1, PageSize: MaxPageSize + 1},
	}
	for _, f := range cases {
		_, err := s.List(context.Background(), f)
		if err == nil {
			t.Fatalf("expected validation error for filter %+v", f)
		}
		assertCode(t, err, apperrors.CodeValidation)
	}
}

// faultyRepo lets each method's behavior be injected so the service's error
// paths (repository failures distinct from not-found) can be exercised.
type faultyRepo struct {
	createErr      error
	updateErr      error
	findByIDErr    error
	findByDocErr   error
	listErr        error
	findByDocFound *Customer
	findByIDValue  *Customer
}

var errBoom = errors.New("boom")

func (r faultyRepo) Create(context.Context, *Customer) error     { return r.createErr }
func (r faultyRepo) Update(context.Context, *Customer) error     { return r.updateErr }
func (r faultyRepo) Deactivate(context.Context, uuid.UUID) error { return r.updateErr }
func (r faultyRepo) FindByID(context.Context, uuid.UUID) (*Customer, error) {
	return r.findByIDValue, r.findByIDErr
}
func (r faultyRepo) FindByDocument(context.Context, string) (*Customer, error) {
	return r.findByDocFound, r.findByDocErr
}
func (r faultyRepo) List(context.Context, CustomerFilter) (*Page[Customer], error) {
	return nil, r.listErr
}

func TestCreateRepositoryErrorIsInternal(t *testing.T) {
	s := NewService(faultyRepo{findByDocErr: ErrNotFound, createErr: errBoom})
	_, err := s.Create(context.Background(), CreateCustomerInput{Name: "A", Document: "D"})
	assertCode(t, err, apperrors.CodeInternal)
}

func TestCreateUniquenessCheckErrorIsInternal(t *testing.T) {
	s := NewService(faultyRepo{findByDocErr: errBoom})
	_, err := s.Create(context.Background(), CreateCustomerInput{Name: "A", Document: "D"})
	assertCode(t, err, apperrors.CodeInternal)
}

func TestUpdateRepositoryErrorIsInternal(t *testing.T) {
	s := NewService(faultyRepo{findByIDErr: errBoom})
	_, err := s.Update(context.Background(), uuid.New(), UpdateCustomerInput{Name: "A", Document: "D"})
	assertCode(t, err, apperrors.CodeInternal)
}

func TestListRepositoryErrorIsInternal(t *testing.T) {
	s := NewService(faultyRepo{listErr: errBoom})
	_, err := s.List(context.Background(), CustomerFilter{})
	assertCode(t, err, apperrors.CodeInternal)
}

func TestRepositoryNotFoundSentinel(t *testing.T) {
	repo := NewInMemoryRepository()
	_, err := repo.FindByID(context.Background(), uuid.New())
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound sentinel, got %v", err)
	}
}
