package deepresearch

import (
	"context"
	"errors"
	"testing"
)

// mockProvider is a test double for AnnualReportProvider.
type mockProvider struct {
	name    string
	reports []AnnualReport
	err     error
	calls   []string // symbols received
}

func (m *mockProvider) Name() string { return m.name }
func (m *mockProvider) FetchAnnualReports(symbol string) ([]AnnualReport, error) {
	m.calls = append(m.calls, symbol)
	return m.reports, m.err
}

// ── Service.FetchAnnualReports ────────────────────────────────────────────────

func TestService_FetchAnnualReports_FirstProviderSucceeds(t *testing.T) {
	want := []AnnualReport{
		{SeqNumber: 1, Issuer: "RELIANCE", Year: "2024", Subject: "Annual Report 2024", PDFLink: "http://example.com/2024.pdf"},
	}
	p := &mockProvider{name: "NSE", reports: want}
	svc := NewService(nil, p)

	reports, source, err := svc.FetchAnnualReports("RELIANCE")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if source != "NSE" {
		t.Errorf("expected source NSE, got %q", source)
	}
	if len(reports) != 1 || reports[0].Year != "2024" {
		t.Errorf("unexpected reports: %+v", reports)
	}
}

func TestService_FetchAnnualReports_FirstFails_SecondSucceeds(t *testing.T) {
	want := []AnnualReport{{SeqNumber: 1, Year: "2023"}}
	p1 := &mockProvider{name: "NSE", err: errors.New("NSE down")}
	p2 := &mockProvider{name: "BSE", reports: want}
	svc := NewService(nil, p1, p2)

	reports, source, err := svc.FetchAnnualReports("TCS")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if source != "BSE" {
		t.Errorf("expected source BSE, got %q", source)
	}
	if len(reports) != 1 {
		t.Errorf("expected 1 report, got %d", len(reports))
	}
}

func TestService_FetchAnnualReports_AllFail(t *testing.T) {
	p1 := &mockProvider{name: "NSE", err: errors.New("NSE down")}
	p2 := &mockProvider{name: "BSE", err: errors.New("BSE down")}
	svc := NewService(nil, p1, p2)

	_, _, err := svc.FetchAnnualReports("INFY")
	if err == nil {
		t.Error("expected error when all providers fail, got nil")
	}
}

func TestService_FetchAnnualReports_NoProviders(t *testing.T) {
	svc := NewService(nil)
	_, _, err := svc.FetchAnnualReports("WIPRO")
	if err == nil {
		t.Error("expected error with no providers, got nil")
	}
}

func TestService_FetchAnnualReports_NormalisesSymbol(t *testing.T) {
	p := &mockProvider{name: "NSE", reports: []AnnualReport{}}
	svc := NewService(nil, p)

	_, _, err := svc.FetchAnnualReports("  reliance  ")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(p.calls) != 1 || p.calls[0] != "RELIANCE" {
		t.Errorf("expected normalised symbol RELIANCE, got %v", p.calls)
	}
}

func TestService_FetchAnnualReports_EmptyReportsOK(t *testing.T) {
	p := &mockProvider{name: "NSE", reports: []AnnualReport{}}
	svc := NewService(nil, p)

	reports, source, err := svc.FetchAnnualReports("SBIN")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if source != "NSE" {
		t.Errorf("expected source NSE, got %q", source)
	}
	if len(reports) != 0 {
		t.Errorf("expected 0 reports, got %d", len(reports))
	}
}

func TestService_FetchAnnualReports_TriesProvidersInOrder(t *testing.T) {
	var order []string
	p1 := &mockProvider{name: "A", err: errors.New("fail")}
	p2 := &mockProvider{name: "B", err: errors.New("fail")}
	p3 := &mockProvider{name: "C", reports: []AnnualReport{{Year: "2024"}}}

	// wrap to track order
	tracing := NewService(nil,
		&tracingProvider{inner: p1, order: &order},
		&tracingProvider{inner: p2, order: &order},
		&tracingProvider{inner: p3, order: &order},
	)

	_, source, _ := tracing.FetchAnnualReports("X")
	if source != "C" {
		t.Errorf("expected source C, got %q", source)
	}
	if len(order) != 3 || order[0] != "A" || order[1] != "B" || order[2] != "C" {
		t.Errorf("unexpected call order: %v", order)
	}
}

// tracingProvider records which providers were called in what order.
type tracingProvider struct {
	inner AnnualReportProvider
	order *[]string
}

func (tp *tracingProvider) Name() string { return tp.inner.Name() }
func (tp *tracingProvider) FetchAnnualReports(symbol string) ([]AnnualReport, error) {
	*tp.order = append(*tp.order, tp.inner.Name())
	return tp.inner.FetchAnnualReports(symbol)
}

// ── Service.Fetch ─────────────────────────────────────────────────────────────

func TestService_Fetch_ReturnsDeepResearch(t *testing.T) {
	reports := []AnnualReport{
		{SeqNumber: 1, Issuer: "HDFC", Year: "2024"},
	}
	p := &mockProvider{name: "NSE", reports: reports}
	svc := NewService(nil, p)

	result, err := svc.Fetch(context.Background(), "hdfcbank")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Symbol != "HDFCBANK" {
		t.Errorf("expected symbol HDFCBANK, got %q", result.Symbol)
	}
	if result.AnnualReportsSource != "NSE" {
		t.Errorf("expected source NSE, got %q", result.AnnualReportsSource)
	}
	if len(result.AnnualReports) != 1 {
		t.Errorf("expected 1 annual report, got %d", len(result.AnnualReports))
	}
}

func TestService_Fetch_ErrorPropagated(t *testing.T) {
	p := &mockProvider{name: "NSE", err: errors.New("down")}
	svc := NewService(nil, p)

	_, err := svc.Fetch(context.Background(), "RELIANCE")
	if err == nil {
		t.Error("expected error when all providers fail, got nil")
	}
}

func TestService_Fetch_SymbolNormalised(t *testing.T) {
	p := &mockProvider{name: "NSE", reports: []AnnualReport{}}
	svc := NewService(nil, p)

	result, err := svc.Fetch(context.Background(), "  tcs  ")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Symbol != "TCS" {
		t.Errorf("expected normalised symbol TCS, got %q", result.Symbol)
	}
}

// ── Provider.Name ─────────────────────────────────────────────────────────────

func TestNSEProvider_Name(t *testing.T) {
	p := &NSEProvider{}
	if got := p.Name(); got != "NSE" {
		t.Errorf("NSEProvider.Name() = %q, want NSE", got)
	}
}

func TestBSEProvider_Name(t *testing.T) {
	p := NewBSEProvider()
	if got := p.Name(); got != "BSE" {
		t.Errorf("BSEProvider.Name() = %q, want BSE", got)
	}
}
