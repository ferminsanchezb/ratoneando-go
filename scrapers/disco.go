package scrapers

import (
	"encoding/json"
	"fmt"

	"ratoneando/cores/api"
	"ratoneando/products"
	"ratoneando/utils/logger"
)

func Disco(query string) ([]products.Schema, error) {
	return api.Core(api.CoreProps[ResponseStructure, RawProduct]{
		Query:         query,
		BaseUrl:       "https://www.disco.com.ar",
		SearchPattern: func(q string) string { return "/api/catalog_system/pub/products/search/?ft=" + q },
		Source:        "disco",
		Normalizer: func(response ResponseStructure) []RawProduct {
			var normalizedProducts []RawProduct
			for _, rawProduct := range response {
				var productData ProductData
				if len(rawProduct.ProductData) == 0 {
					continue
				}
				err := json.Unmarshal([]byte(rawProduct.ProductData[0]), &productData)
				if err != nil {
					logger.LogWarn(fmt.Sprintf("Error unmarshalling product data: %s", err))
					continue
				}
				normalizedProducts = append(normalizedProducts, RawProduct{
					ResponseProduct: rawProduct,
					ProductData:     productData,
				})
			}
			return normalizedProducts
		},
		Extractor: func(rawProduct RawProduct) products.ExtendedSchema {
			return products.ExtendedSchema{
				ID:          rawProduct.ProductId,
				Source:      "disco",
				Name:        rawProduct.ProductName,
				Link:        rawProduct.Link,
				Image:       rawProduct.Items[0].Images[0].ImageUrl,
				Unavailable: !rawProduct.Items[0].Sellers[0].CommertialOffer.IsAvailable,
				Price:       rawProduct.Items[0].Sellers[0].CommertialOffer.Price,
				ListPrice:   rawProduct.Items[0].Sellers[0].CommertialOffer.ListPrice,
				Unit:        rawProduct.MeasurementUnitUn,
				UnitFactor:  rawProduct.UnitMultiplierUn,
			}
		},
	})
}
