package yahoo

import (
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"
)

var crumbCache struct {
	sync.Mutex
	value     string
	cookies   []*http.Cookie
	fetchedAt time.Time
}

// GetCrumb returns a valid Yahoo Finance crumb and cookies, refreshing every 30 minutes.
func GetCrumb() (string, []*http.Cookie, error) {
	crumbCache.Lock()
	defer crumbCache.Unlock()

	if crumbCache.value != "" && time.Since(crumbCache.fetchedAt) < 30*time.Minute {
		return crumbCache.value, crumbCache.cookies, nil
	}

	consentReq, _ := http.NewRequest(http.MethodGet, "https://fc.yahoo.com", nil)
	consentReq.Header.Set("User-Agent", "Mozilla/5.0")
	consentResp, err := http.DefaultClient.Do(consentReq)
	if err != nil {
		return "", nil, fmt.Errorf("yahoo crumb: consent request failed: %w", err)
	}
	consentResp.Body.Close()
	cookies := consentResp.Cookies()

	crumbReq, _ := http.NewRequest(http.MethodGet, "https://query2.finance.yahoo.com/v1/test/getcrumb", nil)
	crumbReq.Header.Set("User-Agent", "Mozilla/5.0")
	for _, c := range cookies {
		crumbReq.AddCookie(c)
	}
	crumbResp, err := http.DefaultClient.Do(crumbReq)
	if err != nil {
		return "", nil, fmt.Errorf("yahoo crumb: getcrumb request failed: %w", err)
	}
	defer crumbResp.Body.Close()
	b, _ := io.ReadAll(crumbResp.Body)
	crumb := string(b)
	if crumb == "" {
		return "", nil, fmt.Errorf("yahoo crumb: empty crumb returned")
	}

	crumbCache.value = crumb
	crumbCache.cookies = cookies
	crumbCache.fetchedAt = time.Now()
	return crumb, cookies, nil
}
