package services

import (
	"encoding/json/v2"
	"fmt"
	"io"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/mayswind/ezbookkeeping/pkg/core"
	"github.com/mayswind/ezbookkeeping/pkg/errs"
)

// sgeGoldPriceUrl is the url of Shanghai Gold Exchange Au99.99 quotations
const sgeGoldPriceUrl = "https://www.sge.com.cn/graph/quotations?instid=Au99.99"

// sgeGoldPriceCacheDuration is the duration of the in-memory gold price cache
const sgeGoldPriceCacheDuration = 60

// browserUserAgent is the user agent used to request the gold price
const browserUserAgent = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"

// goldPrice represents the gold price fetched from Shanghai Gold Exchange
type goldPrice struct {
	Price     int64 // cents per gram
	FetchedAt int64
}

// sgeQuotationsResponse represents the response of Shanghai Gold Exchange quotations
type sgeQuotationsResponse struct {
	Data []any `json:"data"`
}

// parseSgeGoldPrice parses the gold price (in cents per gram) from the Shanghai Gold Exchange response body.
// It returns the last value greater than zero in the data array.
func parseSgeGoldPrice(body []byte) (int64, error) {
	var response sgeQuotationsResponse

	if err := json.Unmarshal(body, &response); err != nil {
		return 0, err
	}

	for i := len(response.Data) - 1; i >= 0; i-- {
		value, ok := parseNumericValue(response.Data[i])

		if !ok || value <= 0 {
			continue
		}

		price := int64(math.Round(value * 100))

		if price > 0 {
			return price, nil
		}
	}

	return 0, errs.ErrInvestmentPriceUnavailable
}

func parseNumericValue(value any) (float64, bool) {
	switch typedValue := value.(type) {
	case float64:
		return typedValue, true
	case string:
		parsed, err := strconv.ParseFloat(strings.TrimSpace(typedValue), 64)
		if err != nil {
			return 0, false
		}

		return parsed, true
	case []any:
		// quotation rows are arrays such as ["2026-11-14 10:01:00", "935.42"]
		// shortcut: only the last positive number of a row is taken, which assumes the price is the
		// trailing numeric field; upgrade to explicit column indexing if SGE starts returning OHLC rows
		for i := len(typedValue) - 1; i >= 0; i-- {
			if parsed, ok := parseNumericValue(typedValue[i]); ok {
				return parsed, true
			}
		}

		return 0, false
	default:
		return 0, false
	}
}

// getGoldPrice returns the current gold price, using the in-memory cache unless force is true
func (s *InvestmentService) getGoldPrice(c core.Context, force bool) (*goldPrice, error) {
	s.priceMutex.Lock()
	defer s.priceMutex.Unlock()

	now := time.Now().Unix()

	if !force && s.cachedPrice != nil && now-s.cachedPrice.FetchedAt < sgeGoldPriceCacheDuration {
		return s.cachedPrice, nil
	}

	price, err := fetchSgeGoldPrice()
	if err != nil {
		return nil, err
	}

	s.cachedPrice = price
	return price, nil
}

func fetchSgeGoldPrice() (*goldPrice, error) {
	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	request, err := http.NewRequest("GET", sgeGoldPriceUrl, nil)
	if err != nil {
		return nil, err
	}

	request.Header.Set("User-Agent", browserUserAgent)

	response, err := client.Do(request)
	if err != nil {
		return nil, err
	}

	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code %d", response.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(response.Body, 1<<20))
	if err != nil {
		return nil, err
	}

	price, err := parseSgeGoldPrice(body)
	if err != nil {
		return nil, err
	}

	return &goldPrice{
		Price:     price,
		FetchedAt: time.Now().Unix(),
	}, nil
}
