package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"google.golang.org/api/cloudbilling/v1"
)

var priceGauge = prometheus.NewGaugeVec(prometheus.GaugeOpts{
	Name: "gcp_sku_price",
	Help: "Current temperature of the CPU.",
}, []string{"sku", "description", "region"})

var monitoryRegions = map[string]bool{
	"us-east1":           true,
	"southamerica-east1": true,
}

func main() {
	prometheus.MustRegister(priceGauge)
	go worker()

	http.Handle("/metrics", promhttp.Handler())
	log.Fatal(http.ListenAndServe(":8888", nil))
}

func worker() {
	for {
		tick()
		time.Sleep(time.Hour)
	}
}

func tick() {
	ctx := context.Background()
	cloudbillingService, err := cloudbilling.NewService(ctx)
	if err != nil {
		fmt.Println(err)
		return
	}

	token := ""

	for {
		response, err := cloudbillingService.Services.Skus.List("services/6F81-5844-456A").CurrencyCode("BRL").PageToken(token).Do()
		if err != nil {
			fmt.Println(err)
			break
		}
		for _, sku := range response.Skus {
			measureSKU(sku)

		}
		token = response.NextPageToken
		if token == "" {
			break
		}
	}
}

func measureSKU(sku *cloudbilling.Sku) {
	if len(sku.PricingInfo) == 0 || len(sku.PricingInfo[0].PricingExpression.TieredRates) == 0 {
		return
	}

	price := float64(sku.PricingInfo[0].PricingExpression.TieredRates[0].UnitPrice.Nanos) / float64(1000000000)

	if price == 0 {
		return
	}

	for _, region := range sku.ServiceRegions {
		if monitoryRegions[region] {
			priceGauge.WithLabelValues(sku.SkuId, sku.Description, region).Set(price)
		}
	}
}
