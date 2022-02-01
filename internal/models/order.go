package models

type Order struct {
	ID       int64   `json:"id"`
	UserID   int64   `json:"user_id"`
	Status   string  `json:"status"`
	Total    float64 `json:"total"`
	Currency string  `json:"currency"`
}

type OrderItem struct {
	ProductID int64   `json:"product_id"`
	Quantity  int     `json:"quantity"`
	Price     float64 `json:"price"`
}
