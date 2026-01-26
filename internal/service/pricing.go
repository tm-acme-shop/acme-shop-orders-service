package service

import "math"

// OrderTotal represents the pricing breakdown for an order.
type OrderTotal struct {
	Subtotal float64 `json:"subtotal"`
	Tax      float64 `json:"tax"`
	Total    float64 `json:"total"`
}

// CalculateTax computes tax based on subtotal and configured tax rate.
func CalculateTax(subtotal float64, taxRate float64) float64 {
	return math.Round(subtotal*taxRate*100) / 100
}

// CalculateOrderTotal computes the full order breakdown.
func CalculateOrderTotal(subtotal float64, taxRate float64) OrderTotal {
	tax := CalculateTax(subtotal, taxRate)
	return OrderTotal{Subtotal: subtotal, Tax: tax, Total: subtotal + tax}
}
