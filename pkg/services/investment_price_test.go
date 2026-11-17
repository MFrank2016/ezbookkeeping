package services

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseSgeGoldPriceNormal(t *testing.T) {
	body := []byte(`{"data":[["2026-11-14 10:00:00", "930.50"], ["2026-11-14 10:01:00", "935.42"]]}`)

	price, err := parseSgeGoldPrice(body)
	assert.Nil(t, err)
	assert.Equal(t, int64(93542), price)
}

func TestParseSgeGoldPriceSkipsZeroValues(t *testing.T) {
	body := []byte(`{"data":["920.10", 0, "0", ""]}`)

	price, err := parseSgeGoldPrice(body)
	assert.Nil(t, err)
	assert.Equal(t, int64(92010), price)
}

func TestParseSgeGoldPriceEmptyData(t *testing.T) {
	_, err := parseSgeGoldPrice([]byte(`{"data":[]}`))
	assert.NotNil(t, err)

	_, err = parseSgeGoldPrice([]byte(`{}`))
	assert.NotNil(t, err)
}

func TestParseSgeGoldPriceAllZeroOrInvalid(t *testing.T) {
	_, err := parseSgeGoldPrice([]byte(`{"data":[0, "0", ""]}`))
	assert.NotNil(t, err)

	_, err = parseSgeGoldPrice([]byte(`{"data":["abc"]}`))
	assert.NotNil(t, err)

	_, err = parseSgeGoldPrice([]byte(`not a json`))
	assert.NotNil(t, err)
}
