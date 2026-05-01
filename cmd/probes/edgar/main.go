// SEC EDGAR probe.
//
// For each fund CIK we are interested in, fetches the submissions JSON,
// finds the most recent 13F-HR filing, fetches the holdings information
// table XML, saves both as fixtures, and prints a summary that answers the
// EDGAR questions in PLAN.md.
//
// Usage:
//
//	EDGAR_CONTACT_EMAIL=you@example.com go run ./cmd/probes/edgar
//	go run ./cmd/probes/edgar --email you@example.com
//
// The contact email is mandatory: SEC enforces a User-Agent that identifies
// the requester. Requests without it are blocked.
package main

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const (
	submissionsHost = "https://data.sec.gov"
	archivesHost    = "https://www.sec.gov"
	rateLimit       = 200 * time.Millisecond // ~5 req/s, well under SEC's 10/s ceiling
)

// Funds chosen for stylistic diversity, per PLAN.md.
var defaultCIKs = []string{
	"0001067983", // Berkshire Hathaway
	"0001350694", // Bridgewater Associates
	"0001167483", // Tiger Global Management
	"0001037389", // Renaissance Technologies (quant, distinct from the above)
}

type submissionsResponse struct {
	CIK     string `json:"cik"`
	Name    string `json:"name"`
	Filings struct {
		Recent struct {
			AccessionNumber []string `json:"accessionNumber"`
			FilingDate      []string `json:"filingDate"`
			ReportDate      []string `json:"reportDate"`
			Form            []string `json:"form"`
			PrimaryDocument []string `json:"primaryDocument"`
		} `json:"recent"`
		Files []struct {
			Name        string `json:"name"`
			FilingFrom  string `json:"filingFrom"`
			FilingTo    string `json:"filingTo"`
			FilingCount int    `json:"filingCount"`
		} `json:"files"`
	} `json:"filings"`
}

type filingIndex struct {
	Directory struct {
		Item []struct {
			Name         string `json:"name"`
			Type         string `json:"type"`
			Size         string `json:"size"`
			LastModified string `json:"last-modified"`
		} `json:"item"`
		Name      string `json:"name"`
		ParentDir string `json:"parent-dir"`
	} `json:"directory"`
}

// Information table is the actual holdings list. The schema namespace has
// shifted across years; we ignore the namespace and pluck the fields we need.
type informationTable struct {
	XMLName  xml.Name    `xml:"informationTable"`
	InfoRows []infoTable `xml:"infoTable"`
}

type infoTable struct {
	NameOfIssuer  string `xml:"nameOfIssuer"`
	TitleOfClass  string `xml:"titleOfClass"`
	Cusip         string `xml:"cusip"`
	Value         string `xml:"value"`
	ShrsOrPrnAmt  struct {
		SshPrnamt     string `xml:"sshPrnamt"`
		SshPrnamtType string `xml:"sshPrnamtType"`
	} `xml:"shrsOrPrnAmt"`
	PutCall              string `xml:"putCall"`
	InvestmentDiscretion string `xml:"investmentDiscretion"`
}

type filingSummary struct {
	CIK             string
	Name            string
	TotalFilings    int
	ThirteenFCount  int
	AmendmentCount  int
	LatestForm      string
	LatestAccession string
	LatestReport    string
	LatestFiled     string
	LagDays         int
	HoldingsCount   int
	TotalValueRaw   int64
	ShareTypeMix    map[string]int
	PutCallCount    int
	SubmissionsPath string
	IndexPath       string
	HoldingsPath    string
	Notes           []string
}

type client struct {
	http      *http.Client
	userAgent string
	lastReq   time.Time
}

func newClient(email string) *client {
	return &client{
		http:      &http.Client{Timeout: 30 * time.Second},
		userAgent: fmt.Sprintf("Marketview Probe %s", email),
	}
}

func (c *client) get(ctx context.Context, rawURL string) ([]byte, int, error) {
	// Polite throttle.
	if !c.lastReq.IsZero() {
		elapsed := time.Since(c.lastReq)
		if elapsed < rateLimit {
			time.Sleep(rateLimit - elapsed)
		}
	}
	c.lastReq = time.Now()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set("User-Agent", c.userAgent)
	req.Header.Set("Accept", "application/json,application/xml,text/xml,*/*")
	req.Header.Set("Host", urlHost(rawURL))

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return body, resp.StatusCode, err
	}
	if resp.StatusCode >= 400 {
		return body, resp.StatusCode, fmt.Errorf("status %d for %s: %s", resp.StatusCode, rawURL, truncate(string(body), 200))
	}
	return body, resp.StatusCode, nil
}

func urlHost(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err != nil {
		return ""
	}
	return u.Host
}

func unpaddedCIK(cik string) string {
	return strings.TrimLeft(cik, "0")
}

func dehyphenAccession(acc string) string {
	return strings.ReplaceAll(acc, "-", "")
}

func writeFixture(path string, body []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, body, 0o644)
}

func parseDate(s string) (time.Time, error) {
	return time.Parse("2006-01-02", s)
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}

func probeOne(ctx context.Context, c *client, cik, outDir string) filingSummary {
	sum := filingSummary{CIK: cik, ShareTypeMix: map[string]int{}}

	// 1. submissions JSON.
	subURL := fmt.Sprintf("%s/submissions/CIK%s.json", submissionsHost, cik)
	subBody, status, err := c.get(ctx, subURL)
	if err != nil {
		sum.Notes = append(sum.Notes, fmt.Sprintf("submissions GET %d: %v", status, err))
		return sum
	}
	subPath := filepath.Join(outDir, fmt.Sprintf("submissions_%s.json", cik))
	_ = writeFixture(subPath, subBody)
	sum.SubmissionsPath = subPath

	var sub submissionsResponse
	if err := json.Unmarshal(subBody, &sub); err != nil {
		sum.Notes = append(sum.Notes, fmt.Sprintf("submissions parse: %v", err))
		return sum
	}
	sum.Name = sub.Name
	sum.TotalFilings = len(sub.Filings.Recent.AccessionNumber)
	if len(sub.Filings.Files) > 0 {
		sum.Notes = append(sum.Notes, fmt.Sprintf("older filings spread across %d additional files (not paginated by probe)", len(sub.Filings.Files)))
	}

	// 2. find the most recent 13F-HR (and count amendments).
	var latestIdx = -1
	for i, form := range sub.Filings.Recent.Form {
		if form == "13F-HR" {
			sum.ThirteenFCount++
			if latestIdx < 0 || sub.Filings.Recent.FilingDate[i] > sub.Filings.Recent.FilingDate[latestIdx] {
				latestIdx = i
			}
		} else if form == "13F-HR/A" {
			sum.AmendmentCount++
		}
	}
	if latestIdx < 0 {
		sum.Notes = append(sum.Notes, "no 13F-HR found in recent filings")
		return sum
	}
	accDashed := sub.Filings.Recent.AccessionNumber[latestIdx]
	sum.LatestForm = sub.Filings.Recent.Form[latestIdx]
	sum.LatestAccession = accDashed
	sum.LatestReport = sub.Filings.Recent.ReportDate[latestIdx]
	sum.LatestFiled = sub.Filings.Recent.FilingDate[latestIdx]
	if rd, err1 := parseDate(sum.LatestReport); err1 == nil {
		if fd, err2 := parseDate(sum.LatestFiled); err2 == nil {
			sum.LagDays = int(fd.Sub(rd) / (24 * time.Hour))
		}
	}

	// 3. fetch the filing's index.json to discover the info table filename.
	accNoDash := dehyphenAccession(accDashed)
	indexURL := fmt.Sprintf("%s/Archives/edgar/data/%s/%s/index.json",
		archivesHost, unpaddedCIK(cik), accNoDash)
	indexBody, _, err := c.get(ctx, indexURL)
	if err != nil {
		sum.Notes = append(sum.Notes, fmt.Sprintf("index GET: %v", err))
		return sum
	}
	indexPath := filepath.Join(outDir, fmt.Sprintf("index_%s_%s.json", cik, accNoDash))
	_ = writeFixture(indexPath, indexBody)
	sum.IndexPath = indexPath

	var idx filingIndex
	if err := json.Unmarshal(indexBody, &idx); err != nil {
		sum.Notes = append(sum.Notes, fmt.Sprintf("index parse: %v", err))
		return sum
	}

	// The info table is conventionally the second XML in the filing, named
	// like "infotable.xml" or "<accession>-information_table.xml". Heuristic:
	// prefer files whose name contains "info" / "informationtable", otherwise
	// pick the largest .xml that is not the primary doc.
	var infoTableName string
	for _, item := range idx.Directory.Item {
		lname := strings.ToLower(item.Name)
		if strings.HasSuffix(lname, ".xml") && (strings.Contains(lname, "infotable") || strings.Contains(lname, "information_table") || strings.Contains(lname, "informationtable")) {
			infoTableName = item.Name
			break
		}
	}
	if infoTableName == "" {
		// Fallback: largest .xml that is not the primary doc.
		var biggest int64 = -1
		primary := strings.ToLower(sub.Filings.Recent.PrimaryDocument[latestIdx])
		for _, item := range idx.Directory.Item {
			if !strings.HasSuffix(strings.ToLower(item.Name), ".xml") {
				continue
			}
			if strings.EqualFold(item.Name, primary) {
				continue
			}
			sz, _ := strconv.ParseInt(item.Size, 10, 64)
			if sz > biggest {
				biggest = sz
				infoTableName = item.Name
			}
		}
	}
	if infoTableName == "" {
		sum.Notes = append(sum.Notes, "no info table xml identified")
		return sum
	}

	// 4. fetch info table.
	infoURL := fmt.Sprintf("%s/Archives/edgar/data/%s/%s/%s",
		archivesHost, unpaddedCIK(cik), accNoDash, infoTableName)
	infoBody, _, err := c.get(ctx, infoURL)
	if err != nil {
		sum.Notes = append(sum.Notes, fmt.Sprintf("infotable GET: %v", err))
		return sum
	}
	infoPath := filepath.Join(outDir, fmt.Sprintf("infotable_%s_%s.xml", cik, accNoDash))
	_ = writeFixture(infoPath, infoBody)
	sum.HoldingsPath = infoPath

	var table informationTable
	dec := xml.NewDecoder(strings.NewReader(string(infoBody)))
	// Be permissive about namespaces.
	dec.Strict = false
	if err := dec.Decode(&table); err != nil {
		sum.Notes = append(sum.Notes, fmt.Sprintf("infotable parse: %v", err))
		return sum
	}
	sum.HoldingsCount = len(table.InfoRows)
	for _, h := range table.InfoRows {
		v, _ := strconv.ParseInt(strings.TrimSpace(h.Value), 10, 64)
		sum.TotalValueRaw += v
		sum.ShareTypeMix[h.ShrsOrPrnAmt.SshPrnamtType]++
		if strings.TrimSpace(h.PutCall) != "" {
			sum.PutCallCount++
		}
	}

	return sum
}

func printSummary(s filingSummary) {
	if s.Name == "" {
		fmt.Printf("CIK %s: FAILED %s\n", s.CIK, strings.Join(s.Notes, "; "))
		return
	}
	// SEC switched 13F value units from $1000s to whole dollars effective for
	// filings on or after 2023-01-03. Use filing date as the discriminator,
	// not the magnitude: small-fund $1s totals look identical to large-fund
	// $1000s totals.
	unitGuess := "?"
	if fd, err := parseDate(s.LatestFiled); err == nil {
		if fd.Before(time.Date(2023, 1, 3, 0, 0, 0, 0, time.UTC)) {
			unitGuess = "$1000s (pre-2023 schema)"
		} else {
			unitGuess = "$1s (post-2023 schema)"
		}
	}
	fmt.Printf("== %s (CIK %s) ==\n", s.Name, s.CIK)
	fmt.Printf("  recent filings: %d total | 13F-HR: %d | 13F-HR/A: %d\n",
		s.TotalFilings, s.ThirteenFCount, s.AmendmentCount)
	fmt.Printf("  latest 13F-HR : %s  reportDate=%s  filed=%s  lag=%dd  acc=%s\n",
		s.LatestForm, s.LatestReport, s.LatestFiled, s.LagDays, s.LatestAccession)
	fmt.Printf("  holdings rows : %d | sum(value)=%d (units guess: %s) | put/call rows: %d\n",
		s.HoldingsCount, s.TotalValueRaw, unitGuess, s.PutCallCount)
	fmt.Printf("  share types   : %v\n", s.ShareTypeMix)
	fmt.Printf("  fixtures      : %s\n                  %s\n                  %s\n",
		s.SubmissionsPath, s.IndexPath, s.HoldingsPath)
	for _, n := range s.Notes {
		fmt.Printf("  note: %s\n", n)
	}
	fmt.Println()
}

func main() {
	emailFlag := flag.String("email", os.Getenv("EDGAR_CONTACT_EMAIL"), "contact email for SEC User-Agent header (required)")
	ciksFlag := flag.String("ciks", strings.Join(defaultCIKs, ","), "comma-separated 10-digit zero-padded CIKs")
	outDir := flag.String("out", "testdata/edgar", "fixture output directory")
	flag.Parse()

	if strings.TrimSpace(*emailFlag) == "" {
		die(errors.New("contact email is required: pass --email or set EDGAR_CONTACT_EMAIL. SEC enforces a User-Agent containing it."))
	}

	if err := os.MkdirAll(*outDir, 0o755); err != nil {
		die(err)
	}

	c := newClient(*emailFlag)
	ctx := context.Background()

	fmt.Printf("EDGAR probe | UA=%q | rate=%v\n\n", c.userAgent, rateLimit)

	for _, cik := range strings.Split(*ciksFlag, ",") {
		cik = strings.TrimSpace(cik)
		if cik == "" {
			continue
		}
		sum := probeOne(ctx, c, cik, *outDir)
		printSummary(sum)
	}
}

func die(err error) {
	fmt.Fprintln(os.Stderr, "fatal:", err)
	os.Exit(1)
}
