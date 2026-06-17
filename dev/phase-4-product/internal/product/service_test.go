package product

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"

	apperrors "order-service-product/internal/errors"
	"order-service-product/internal/money"
)

func newService() *Service {
	return NewService(NewInMemoryRepository())
}

func mustCreate(t *testing.T, s *Service, name, sku string, price money.Money) *ProductOutput {
	t.Helper()
	out, err := s.Create(context.Background(), CreateProductInput{Name: name, SKU: sku, Price: price})
	if err != nil {
		t.Fatalf("Create(%q,%q) unexpected error: %v", name, sku, err)
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

func price8990(t *testing.T) money.Money {
	t.Helper()
	p, err := money.ParseString("89.90")
	if err != nil {
		t.Fatalf("parse price: %v", err)
	}
	return p
}

// BR-PRD-001: name required.
func TestCreateRequiresName(t *testing.T) {
	s := newService()
	_, err := s.Create(context.Background(), CreateProductInput{Name: "  ", SKU: "SKU-1", Price: price8990(t)})
	if err == nil {
		t.Fatal("expected validation error for missing name")
	}
	assertCode(t, err, apperrors.CodeValidation)
}

// BR-PRD-003: price must be greater than zero.
func TestCreateRejectsNonPositivePrice(t *testing.T) {
	s := newService()
	_, err := s.Create(context.Background(), CreateProductInput{
		Name:  "Mouse",
		SKU:   "SKU-ZERO",
		Price: money.FromCents(0),
	})
	if err == nil {
		t.Fatal("expected validation error for zero price")
	}
	assertCode(t, err, apperrors.CodeValidation)
}

func TestCreateValidReturnsProduct(t *testing.T) {
	s := newService()
	out := mustCreate(t, s, "Wireless Mouse", "MOUSE-001", price8990(t))
	if out.ID == uuid.Nil {
		t.Fatal("expected a generated id")
	}
	if !out.Active {
		t.Fatal("new product must be active")
	}
	if out.Name != "Wireless Mouse" || out.SKU != "MOUSE-001" || out.Price.Cents() != 8990 {
		t.Fatalf("unexpected output: %+v", out)
	}
}

// BR-PRD-002: sku must be unique.
func TestCreateDuplicateSKU(t *testing.T) {
	s := newService()
	mustCreate(t, s, "Mouse A", "SKU-DUP", price8990(t))
	_, err := s.Create(context.Background(), CreateProductInput{Name: "Mouse B", SKU: "SKU-DUP", Price: price8990(t)})
	if err == nil {
		t.Fatal("expected duplicate error")
	}
	assertCode(t, err, apperrors.CodeDuplicate)
}

func TestUpdateChangesMutableFields(t *testing.T) {
	s := newService()
	created := mustCreate(t, s, "Old Name", "SKU-UPD", price8990(t))
	newPrice, _ := money.ParseString("99.99")
	out, err := s.Update(context.Background(), created.ID, UpdateProductInput{
		Name:  "New Name",
		SKU:   "SKU-UPD",
		Price: newPrice,
	})
	if err != nil {
		t.Fatalf("Update unexpected error: %v", err)
	}
	if out.Name != "New Name" || out.Price.Cents() != 9999 {
		t.Fatalf("mutable fields not updated: %+v", out)
	}
}

func TestUpdateRejectsNonPositivePrice(t *testing.T) {
	s := newService()
	created := mustCreate(t, s, "Mouse", "SKU-NEG", price8990(t))
	_, err := s.Update(context.Background(), created.ID, UpdateProductInput{
		Name:  "Mouse",
		SKU:   "SKU-NEG",
		Price: money.FromCents(-100),
	})
	assertCode(t, err, apperrors.CodeValidation)
}

func TestUpdateUnknownReturnsNotFound(t *testing.T) {
	s := newService()
	_, err := s.Update(context.Background(), uuid.New(), UpdateProductInput{
		Name: "X", SKU: "Y", Price: price8990(t),
	})
	assertCode(t, err, apperrors.CodeNotFound)
}

func TestUpdateDuplicateSKU(t *testing.T) {
	s := newService()
	mustCreate(t, s, "A", "SKU-A", price8990(t))
	b := mustCreate(t, s, "B", "SKU-B", price8990(t))
	_, err := s.Update(context.Background(), b.ID, UpdateProductInput{
		Name: "B", SKU: "SKU-A", Price: price8990(t),
	})
	assertCode(t, err, apperrors.CodeDuplicate)
}

func TestDeactivateIsSoftAndRetrievable(t *testing.T) {
	s := newService()
	created := mustCreate(t, s, "Mouse", "SKU-DEACT", price8990(t))
	if err := s.Deactivate(context.Background(), created.ID); err != nil {
		t.Fatalf("Deactivate unexpected error: %v", err)
	}
	got, err := s.FindByID(context.Background(), created.ID)
	if err != nil {
		t.Fatalf("product must still be retrievable after deactivate: %v", err)
	}
	if got.Active {
		t.Fatal("deactivated product must have active=false")
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

func TestFindByIDReturnsRightProduct(t *testing.T) {
	s := newService()
	mustCreate(t, s, "First", "SKU-1", price8990(t))
	want := mustCreate(t, s, "Second", "SKU-2", price8990(t))
	got, err := s.FindByID(context.Background(), want.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.ID != want.ID || got.Name != "Second" {
		t.Fatalf("wrong product returned: %+v", got)
	}
}

func TestFindManyByIDReturnsMatchingSubset(t *testing.T) {
	s := newService()
	a := mustCreate(t, s, "A", "SKU-A", price8990(t))
	b := mustCreate(t, s, "B", "SKU-B", price8990(t))
	mustCreate(t, s, "C", "SKU-C", price8990(t))

	got, err := s.FindManyByID(context.Background(), []uuid.UUID{a.ID, b.ID, uuid.New()})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 products, got %d", len(got))
	}
	ids := map[uuid.UUID]bool{got[0].ID: true, got[1].ID: true}
	if !ids[a.ID] || !ids[b.ID] {
		t.Fatalf("wrong ids returned: %+v", got)
	}
}

// BR-PRD-005 enabling state: price is available on outputs for order copy at creation.
func TestFindManyByIDExposesPriceForOrderCopy(t *testing.T) {
	s := newService()
	created := mustCreate(t, s, "Mouse", "SKU-PRICE", price8990(t))
	got, err := s.FindManyByID(context.Background(), []uuid.UUID{created.ID})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 1 || got[0].Price.Cents() != 8990 {
		t.Fatalf("price must be readable for order copy: %+v", got)
	}
}

func TestListFiltersByName(t *testing.T) {
	s := newService()
	mustCreate(t, s, "Wireless Mouse", "SKU-1", price8990(t))
	mustCreate(t, s, "Keyboard", "SKU-2", price8990(t))
	page, err := s.List(context.Background(), ProductFilter{Name: "mouse"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if page.Total != 1 || len(page.Items) != 1 || page.Items[0].Name != "Wireless Mouse" {
		t.Fatalf("name filter wrong: %+v", page)
	}
}

func TestListFiltersBySKU(t *testing.T) {
	s := newService()
	mustCreate(t, s, "Mouse", "MOUSE-AAA", price8990(t))
	mustCreate(t, s, "Keyboard", "KEY-BBB", price8990(t))
	page, err := s.List(context.Background(), ProductFilter{SKU: "bbb"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if page.Total != 1 || page.Items[0].SKU != "KEY-BBB" {
		t.Fatalf("sku filter wrong: %+v", page)
	}
}

func TestListFiltersByActive(t *testing.T) {
	s := newService()
	a := mustCreate(t, s, "Active One", "SKU-1", price8990(t))
	mustCreate(t, s, "Active Two", "SKU-2", price8990(t))
	if err := s.Deactivate(context.Background(), a.ID); err != nil {
		t.Fatalf("deactivate: %v", err)
	}
	inactive := false
	page, err := s.List(context.Background(), ProductFilter{Active: &inactive})
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
		mustCreate(t, s, "Product", uuid.NewString(), price8990(t))
	}
	page, err := s.List(context.Background(), ProductFilter{Page: 2, PageSize: 2})
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
	mustCreate(t, s, "Only", "SKU-1", price8990(t))
	page, err := s.List(context.Background(), ProductFilter{})
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
	cases := []ProductFilter{
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

type faultyRepo struct {
	createErr      error
	updateErr      error
	findByIDErr    error
	findBySKUErr   error
	listErr        error
	findBySKUFound *Product
	findByIDValue  *Product
}

var errBoom = errors.New("boom")

func (r faultyRepo) Create(context.Context, *Product) error      { return r.createErr }
func (r faultyRepo) Update(context.Context, *Product) error      { return r.updateErr }
func (r faultyRepo) Deactivate(context.Context, uuid.UUID) error { return r.updateErr }
func (r faultyRepo) FindByID(context.Context, uuid.UUID) (*Product, error) {
	return r.findByIDValue, r.findByIDErr
}
func (r faultyRepo) FindBySKU(context.Context, string) (*Product, error) {
	return r.findBySKUFound, r.findBySKUErr
}
func (r faultyRepo) FindManyByID(context.Context, []uuid.UUID) ([]Product, error) {
	return nil, r.findByIDErr
}
func (r faultyRepo) List(context.Context, ProductFilter) (*Page[Product], error) {
	return nil, r.listErr
}

func TestCreateRepositoryErrorIsInternal(t *testing.T) {
	s := NewService(faultyRepo{findBySKUErr: ErrNotFound, createErr: errBoom})
	_, err := s.Create(context.Background(), CreateProductInput{Name: "A", SKU: "D", Price: price8990(t)})
	assertCode(t, err, apperrors.CodeInternal)
}

func TestCreateUniquenessCheckErrorIsInternal(t *testing.T) {
	s := NewService(faultyRepo{findBySKUErr: errBoom})
	_, err := s.Create(context.Background(), CreateProductInput{Name: "A", SKU: "D", Price: price8990(t)})
	assertCode(t, err, apperrors.CodeInternal)
}

func TestUpdateRepositoryErrorIsInternal(t *testing.T) {
	s := NewService(faultyRepo{findByIDErr: errBoom})
	_, err := s.Update(context.Background(), uuid.New(), UpdateProductInput{Name: "A", SKU: "D", Price: price8990(t)})
	assertCode(t, err, apperrors.CodeInternal)
}

func TestListRepositoryErrorIsInternal(t *testing.T) {
	s := NewService(faultyRepo{listErr: errBoom})
	_, err := s.List(context.Background(), ProductFilter{})
	assertCode(t, err, apperrors.CodeInternal)
}

func TestRepositoryNotFoundSentinel(t *testing.T) {
	repo := NewInMemoryRepository()
	_, err := repo.FindByID(context.Background(), uuid.New())
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound sentinel, got %v", err)
	}
}

func TestRepositoryFindManyByID(t *testing.T) {
	repo := NewInMemoryRepository()
	ctx := context.Background()
	p := &Product{ID: uuid.New(), Name: "X", SKU: "S", Price: price8990(t), Active: true}
	if err := repo.Create(ctx, p); err != nil {
		t.Fatalf("create: %v", err)
	}
	got, err := repo.FindManyByID(ctx, []uuid.UUID{p.ID, uuid.New()})
	if err != nil {
		t.Fatalf("FindManyByID: %v", err)
	}
	if len(got) != 1 || got[0].ID != p.ID {
		t.Fatalf("unexpected result: %+v", got)
	}
}
