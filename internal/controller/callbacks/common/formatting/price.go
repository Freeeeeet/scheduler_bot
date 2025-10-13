package formatting

import "fmt"

// FormatPrice форматирует цену из копеек в рубли
func FormatPrice(priceInCents int) string {
	price := float64(priceInCents) / 100
	return fmt.Sprintf("%.2f ₽", price)
}

// FormatPriceShort форматирует цену без копеек если они равны 0
func FormatPriceShort(priceInCents int) string {
	price := float64(priceInCents) / 100
	if priceInCents%100 == 0 {
		return fmt.Sprintf("%.0f ₽", price)
	}
	return fmt.Sprintf("%.2f ₽", price)
}
