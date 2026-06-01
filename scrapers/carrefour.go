package scrapers

import (
	"encoding/json"
	"fmt"

	"ratoneando/cores/api"
	"ratoneando/products"
	"ratoneando/utils/logger"
)

func Carrefour(query string) ([]products.Schema, error) {
	return api.Core(api.CoreProps[ResponseStructure, RawProduct]{
		Query:         query,
		BaseUrl:       "https://www.carrefour.com.ar",
		SearchPattern: func(q string) string { return "/api/catalog_system/pub/products/search/?ft=" + q },
		Source:        "carrefour",
		Normalizer: func(response ResponseStructure) []RawProduct {
			var normalizedProducts []RawProduct
			for _, rawProduct := range response {
				var productData ProductData
				// Carrefour no devuelve ProductData; lo procesamos igual con Unit vacío.
				if len(rawProduct.ProductData) > 0 {
					err := json.Unmarshal([]byte(rawProduct.ProductData[0]), &productData)
					if err != nil {
						logger.LogWarn(fmt.Sprintf("Error unmarshalling product data: %s", err))
					}
				}
				normalizedProducts = append(normalizedProducts, RawProduct{
					ResponseProduct: rawProduct,
					ProductData:     productData,
				})
			}
			return normalizedProducts
		},
		Extractor: func(rawProduct RawProduct) products.ExtendedSchema {
			var image string
			if len(rawProduct.Items) > 0 && len(rawProduct.Items[0].Images) > 0 {
				image = rawProduct.Items[0].Images[0].ImageUrl
			}
			var price, listPrice float64
			var unavailable bool
			if len(rawProduct.Items) > 0 && len(rawProduct.Items[0].Sellers) > 0 {
				offer := rawProduct.Items[0].Sellers[0].CommertialOffer
				price = offer.Price
				listPrice = offer.ListPrice
				unavailable = !offer.IsAvailable
			} else {
				unavailable = true
			}
			return products.ExtendedSchema{
				ID:          rawProduct.ProductId,
				Source:      "carrefour",
				Name:        rawProduct.ProductName,
				Link:        rawProduct.Link,
				Image:       image,
				Unavailable: unavailable,
				Price:       price,
				ListPrice:   listPrice,
				Unit:        rawProduct.MeasurementUnitUn,
				UnitFactor:  rawProduct.UnitMultiplierUn,
			}
		},
	})
}
