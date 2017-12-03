package trend

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"reflect"
	"strings"
	"time"

	"github.com/influxdata/influxdb/models"
)

// Ticker represents a response from the ticker endpoint of
// coinmarketcap.com api.
type Ticker struct {
	ID              string
	Name            string
	Symbol          string
	Rank            int `json:",string"`
	AvailableSupply float64
	TotalSupply     float64
	MaxSupply       float64
	LastUpdated     time.Time
	Price
	Volume
	MarketCap

	requestedCurrency string
}

// GetNewTick retreives a tick of just crypto currency.
func GetNewTick(crypto, exchange string) (Ticker, error) {
	if crypto == "" {
		return Ticker{}, fmt.Errorf("crypto currency not specified")
	}
	var url string
	if exchange == "" {
		url = fmt.Sprintf("https://api.coinmarketcap.com/v1/ticker/%s/", crypto)
	} else {
		url = fmt.Sprintf("https://api.coinmarketcap.com/v1/ticker/%s/?convert=%s", crypto, exchange)
	}
	resp, err := http.Get(url)
	if err != nil {
		return Ticker{}, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return Ticker{}, err
	}

	ticks := []Ticker{}

	err = json.Unmarshal(body, &ticks)
	if err != nil {
		return Ticker{}, err
	}

	return ticks[0], nil
}

// GetNewTicks retreives a tick for all available crypto currencies.
func GetNewTicks(exchange string) ([]Ticker, error) {
	var url string
	if exchange == "" {
		url = "https://api.coinmarketcap.com/v1/ticker/?limit=0"
	} else {
		url = fmt.Sprintf("https://api.coinmarketcap.com/v1/ticker/?limit=0&convert=%s", exchange)
	}
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	ticks := []Ticker{}

	err = json.Unmarshal(body, &ticks)
	if err != nil {
		return nil, err
	}

	return ticks, nil
}

// UnmarshalJSON unmarshals a unix timestamp to time.Time.
func (t *Ticker) UnmarshalJSON(data []byte) error {
	type Alias Ticker
	aux := &struct {
		LastUpdated int64 `json:"last_updated,string"`
		*Alias
	}{
		Alias: (*Alias)(t),
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	t.LastUpdated = time.Unix(aux.LastUpdated, 0)
	return nil
}

func (t *Ticker) MarshalInfluxdbLineProto(exchange string) (models.Point, error) {
	// create measurement name (string)
	name := t.ID
	// create tags
	tagsMap := map[string]string{
		"symbol": t.Symbol,
	}
	tags := models.NewTags(tagsMap)

	// create fields
	fields := make(models.Fields)
	priceFields := t.Price.Fields(exchange)
	volumeFields := t.Volume.Fields(exchange)
	marketCapFields := t.MarketCap.Fields(exchange)

	// consolidate price, volume and marketcap fields into one fields map
	for k, v := range priceFields {
		if _, ok := priceFields[k]; ok {
			fields[k] = v
		}
	}

	for k, v := range volumeFields {
		if _, ok := volumeFields[k]; ok {
			fields[k] = v
		}
	}

	for k, v := range marketCapFields {
		if _, ok := marketCapFields[k]; ok {
			fields[k] = v
		}
	}

	// create point
	point, err := models.NewPoint(name, tags, fields, t.LastUpdated)
	if err != nil {
		return nil, err
	}

	return point, nil
}

// Price is a hacky way to deal with variable price currencies.
type Price struct {
	AUD float64 `json:"price_aud,omitempty,string"`
	BRL float64 `json:"price_brl,omitempty,string"`
	BTC float64 `json:"price_btc,omitempty,string"`
	CAD float64 `json:"price_cad,omitempty,string"`
	CHF float64 `json:"price_chf,omitempty,string"`
	CLP float64 `json:"price_clp,omitempty,string"`
	CNY float64 `json:"price_cny,omitempty,string"`
	CZK float64 `json:"price_czk,omitempty,string"`
	DKK float64 `json:"price_dkk,omitempty,string"`
	EUR float64 `json:"price_eur,omitempty,string"`
	GBP float64 `json:"price_gbp,omitempty,string"`
	HKD float64 `json:"price_hkd,omitempty,string"`
	HUF float64 `json:"price_huf,omitempty,string"`
	IDR float64 `json:"price_idr,omitempty,string"`
	ILS float64 `json:"price_ils,omitempty,string"`
	INR float64 `json:"price_inr,omitempty,string"`
	JPY float64 `json:"price_jpy,omitempty,string"`
	KRW float64 `json:"price_krw,omitempty,string"`
	MXN float64 `json:"price_mxn,omitempty,string"`
	MYR float64 `json:"price_myr,omitempty,string"`
	NOK float64 `json:"price_nok,omitempty,string"`
	NZD float64 `json:"price_nzd,omitempty,string"`
	PHP float64 `json:"price_php,omitempty,string"`
	PKR float64 `json:"price_pkr,omitempty,string"`
	PLN float64 `json:"price_pln,omitempty,string"`
	RUB float64 `json:"price_rub,omitempty,string"`
	SEK float64 `json:"price_sek,omitempty,string"`
	SGD float64 `json:"price_sgd,omitempty,string"`
	THB float64 `json:"price_thb,omitempty,string"`
	TRY float64 `json:"price_try,omitempty,string"`
	TWD float64 `json:"price_twd,omitempty,string"`
	USD float64 `json:"price_usd,omitempty,string"`
	ZAR float64 `json:"price_zar,omitempty,string"`
}

// Fields returns the appropriate values in the Influxdb fields format.
func (p *Price) Fields(reqCurrency string) models.Fields {
	val := reflect.ValueOf(p).Elem()
	if reqCurrency != "" {
		customCurrValue := val.FieldByName(strings.ToUpper(reqCurrency)).Float()

		customCurrStr := strings.ToLower(reqCurrency)
		convertPriceStr := "price_" + customCurrStr
		return models.Fields(map[string]interface{}{
			"price_usd":     p.USD,
			"price_btc":     p.BTC,
			convertPriceStr: customCurrValue,
		})
	}
	return models.Fields(map[string]interface{}{
		"price_usd": p.USD,
		"price_btc": p.BTC,
	})
}

// Volume is a hacky way to deal with variable volume currencies.
type Volume struct {
	AUD float64 `json:"24h_volume_aud,omitempty,string"`
	BRL float64 `json:"24h_volume_brl,omitempty,string"`
	CAD float64 `json:"24h_volume_cad,omitempty,string"`
	CHF float64 `json:"24h_volume_chf,omitempty,string"`
	CLP float64 `json:"24h_volume_clp,omitempty,string"`
	CNY float64 `json:"24h_volume_cny,omitempty,string"`
	CZK float64 `json:"24h_volume_czk,omitempty,string"`
	DKK float64 `json:"24h_volume_dkk,omitempty,string"`
	EUR float64 `json:"24h_volume_eur,omitempty,string"`
	GBP float64 `json:"24h_volume_gbp,omitempty,string"`
	HKD float64 `json:"24h_volume_hkd,omitempty,string"`
	HUF float64 `json:"24h_volume_huf,omitempty,string"`
	IDR float64 `json:"24h_volume_idr,omitempty,string"`
	ILS float64 `json:"24h_volume_ils,omitempty,string"`
	INR float64 `json:"24h_volume_inr,omitempty,string"`
	JPY float64 `json:"24h_volume_jpy,omitempty,string"`
	KRW float64 `json:"24h_volume_krw,omitempty,string"`
	MXN float64 `json:"24h_volume_mxn,omitempty,string"`
	MYR float64 `json:"24h_volume_myr,omitempty,string"`
	NOK float64 `json:"24h_volume_nok,omitempty,string"`
	NZD float64 `json:"24h_volume_nzd,omitempty,string"`
	PHP float64 `json:"24h_volume_php,omitempty,string"`
	PKR float64 `json:"24h_volume_pkr,omitempty,string"`
	PLN float64 `json:"24h_volume_pln,omitempty,string"`
	RUB float64 `json:"24h_volume_rub,omitempty,string"`
	SEK float64 `json:"24h_volume_sek,omitempty,string"`
	SGD float64 `json:"24h_volume_sgd,omitempty,string"`
	THB float64 `json:"24h_volume_thb,omitempty,string"`
	TRY float64 `json:"24h_volume_try,omitempty,string"`
	TWD float64 `json:"24h_volume_twd,omitempty,string"`
	USD float64 `json:"24h_volume_usd,omitempty,string"`
	ZAR float64 `json:"24h_volume_zar,omitempty,string"`
}

// Fields returns the appropriate values in the Influxdb fields format.
func (v *Volume) Fields(reqCurrency string) models.Fields {
	val := reflect.ValueOf(v).Elem()
	if reqCurrency != "" {
		customCurrValue := val.FieldByName(strings.ToUpper(reqCurrency)).Float()

		customCurrStr := strings.ToLower(reqCurrency)
		convertVolumeStr := "24h_volume_" + customCurrStr
		return models.Fields(map[string]interface{}{
			"24h_volume_usd": v.USD,
			convertVolumeStr: customCurrValue,
		})
	}
	return models.Fields(map[string]interface{}{
		"24h_volume_usd": v.USD,
	})
}

// MarketCap is a hacky way to deal with variable marketcap currencies.
type MarketCap struct {
	AUD float64 `json:"market_cap_aud,omitempty,string"`
	BRL float64 `json:"market_cap_brl,omitempty,string"`
	CAD float64 `json:"market_cap_cad,omitempty,string"`
	CHF float64 `json:"market_cap_chf,omitempty,string"`
	CLP float64 `json:"market_cap_clp,omitempty,string"`
	CNY float64 `json:"market_cap_cny,omitempty,string"`
	CZK float64 `json:"market_cap_czk,omitempty,string"`
	DKK float64 `json:"market_cap_dkk,omitempty,string"`
	EUR float64 `json:"market_cap_eur,omitempty,string"`
	GBP float64 `json:"market_cap_gbp,omitempty,string"`
	HKD float64 `json:"market_cap_hkd,omitempty,string"`
	HUF float64 `json:"market_cap_huf,omitempty,string"`
	IDR float64 `json:"market_cap_idr,omitempty,string"`
	ILS float64 `json:"market_cap_ils,omitempty,string"`
	INR float64 `json:"market_cap_inr,omitempty,string"`
	JPY float64 `json:"market_cap_jpy,omitempty,string"`
	KRW float64 `json:"market_cap_krw,omitempty,string"`
	MXN float64 `json:"market_cap_mxn,omitempty,string"`
	MYR float64 `json:"market_cap_myr,omitempty,string"`
	NOK float64 `json:"market_cap_nok,omitempty,string"`
	NZD float64 `json:"market_cap_nzd,omitempty,string"`
	PHP float64 `json:"market_cap_php,omitempty,string"`
	PKR float64 `json:"market_cap_pkr,omitempty,string"`
	PLN float64 `json:"market_cap_pln,omitempty,string"`
	RUB float64 `json:"market_cap_rub,omitempty,string"`
	SEK float64 `json:"market_cap_sek,omitempty,string"`
	SGD float64 `json:"market_cap_sgd,omitempty,string"`
	THB float64 `json:"market_cap_thb,omitempty,string"`
	TRY float64 `json:"market_cap_try,omitempty,string"`
	TWD float64 `json:"market_cap_twd,omitempty,string"`
	USD float64 `json:"market_cap_usd,omitempty,string"`
	ZAR float64 `json:"market_cap_zar,omitempty,string"`
}

// Fields returns the appropriate values in the Influxdb fields format.
func (mc *MarketCap) Fields(reqCurrency string) models.Fields {
	val := reflect.ValueOf(mc).Elem()
	if reqCurrency != "" {
		customCurrValue := val.FieldByName(strings.ToUpper(reqCurrency)).Float()

		customCurrStr := strings.ToLower(reqCurrency)
		convertMarketCapStr := "market_cap_" + customCurrStr
		return models.Fields(map[string]interface{}{
			"market_cap_usd":    mc.USD,
			convertMarketCapStr: customCurrValue,
		})
	}
	return models.Fields(map[string]interface{}{
		"market_cap_usd": mc.USD,
	})
}
