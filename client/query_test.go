package client

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParsePrice(t *testing.T) {
	price, err := parsePrice(&SnapshotResponse{RealtimeQuote: &RealtimeQuote{CurrentPrice: "12.34"}})

	assert.NoError(t, err)
	assert.Equal(t, 12.34, price)
}

func TestParsePriceInvalid(t *testing.T) {
	price, err := parsePrice(&SnapshotResponse{RealtimeQuote: &RealtimeQuote{CurrentPrice: "invalid"}})

	assert.Error(t, err)
	assert.Zero(t, price)
}
