package scrapers

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"ratoneando/cores/api"
	"ratoneando/products"
	"ratoneando/utils/logger"
)

type CotoResponseProduct struct {
	DetailsAction struct {
		RecordState string `json:"recordState"`
	} `json:"detailsAction"`
	Attributes struct {
		ProductDisplayName  []string `json:"product.displayName"`
		ProductRepositoryId []string `json:"product.repositoryId"`
	} `json:"attributes"`
	Records []struct {
		Attributes struct {
			SkuReferencePrice     []string `json:"sku.referencePrice"`
			SkuActivePrice        []string `json:"sku.activePrice"`
			ProductContent        []string `json:"product.CONTENIDO"`
			ProductMediumImageUrl []string `json:"product.mediumImage.url"`
			SkuQuantity           []string `json:"sku.quantity"`
			ProductDiscounts      []string `json:"product.dtoDescuentos"`
		} `json:"attributes"`
	} `json:"records"`
}

type CotoProductDiscounts []struct {
	PrecioDescuento string `json:"precioDescuento"`
}

type CotoRawProduct struct {
	CotoResponseProduct
	CotoProductDiscounts
}

type CotoResponseStructure struct {
	Contents []struct {
		Main []struct {
			Contents []struct {
				Records []CotoResponseProduct `json:"records"`
			} `json:"contents"`
		} `json:"Main"`
	} `json:"contents"`
}

func Coto(query string) ([]products.Schema, error) {
	return api.Core(api.CoreProps[CotoResponseStructure, CotoRawProduct]{
		Query:         query,
		BaseUrl:       "https://www.cotodigital.com.ar",
		SearchPattern: func(q string) string { return "/sitios/cdigi/categoria?Ntt=" + q + "&format=json" },
		Source:        "coto",
		Normalizer: func(response CotoResponseStructure) []CotoRawProduct {
			var normalizedProducts []CotoRawProduct

			if len(response.Contents) == 0 {
				return normalizedProducts
			}

			// Coto cambió la API: los records pueden estar en distintos índices de Main[]
			// según banners/contenido. Buscamos dinámicamente el primero con records.
			var rawProducts []CotoResponseProduct
			for _, main := range response.Contents[0].Main {
				if len(main.Contents) > 0 && len(main.Contents[0].Records) > 0 {
					rawProducts = main.Contents[0].Records
					break
				}
			}

			for _, rawProduct := range rawProducts {
				if len(rawProduct.Records) == 0 {
					continue
				}

				var productData CotoProductDiscounts
				if len(rawProduct.Records[0].Attributes.ProductDiscounts) > 0 {
					err := json.Unmarshal([]byte(rawProduct.Records[0].Attributes.ProductDiscounts[0]), &productData)
					if err != nil {
						logger.LogWarn(fmt.Sprintf("Error unmarshalling product data: %s", err))
					}
				}

				normalizedProducts = append(normalizedProducts, CotoRawProduct{
					CotoResponseProduct:  rawProduct,
					CotoProductDiscounts: productData,
				})
			}

			return normalizedProducts
		},
		Extractor: func(rawProduct CotoRawProduct) products.ExtendedSchema {
			var listPrice float64
			if len(rawProduct.Records[0].Attributes.SkuActivePrice) > 0 {
				listPrice, _ = strconv.ParseFloat(rawProduct.Records[0].Attributes.SkuActivePrice[0], 64)
			}
			price := listPrice

			if len(rawProduct.CotoProductDiscounts) > 0 {
				precioDescuento, _ := strconv.ParseFloat(rawProduct.CotoProductDiscounts[0].PrecioDescuento, 64)
				if precioDescuento > 0 {
					price = precioDescuento
				}
			}

			var image, id, name string
			if len(rawProduct.Records[0].Attributes.ProductMediumImageUrl) > 0 {
				image = rawProduct.Records[0].Attributes.ProductMediumImageUrl[0]
			}
			if len(rawProduct.CotoResponseProduct.Attributes.ProductRepositoryId) > 0 {
				id = rawProduct.CotoResponseProduct.Attributes.ProductRepositoryId[0]
			}
			if len(rawProduct.CotoResponseProduct.Attributes.ProductDisplayName) > 0 {
				name = rawProduct.CotoResponseProduct.Attributes.ProductDisplayName[0]
			}

			var unavailable bool
			if len(rawProduct.Records[0].Attributes.SkuQuantity) > 0 {
				unavailable = rawProduct.Records[0].Attributes.SkuQuantity[0] == "0"
			}

			return products.ExtendedSchema{
				ID:          id,
				Source:      "coto",
				Name:        name,
				Link:        strings.Replace(rawProduct.DetailsAction.RecordState, "?format=json", "", -1),
				Image:       image,
				Unavailable: unavailable,
				Price:       price,
				ListPrice:   listPrice,
			}
		},
	})
}
