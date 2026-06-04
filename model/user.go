package model

type User struct {
	ID      string `json:"id" example:"A1"`
	Name    string `json:"name" example:"John Doe"`
	Balance int64  `json:"balance" example:"1000"`
}

type AccountListResponse struct {
	Users []User `json:"users"`
}

func SeedUsers() []User {
	return []User{
		{
			ID:      "A1",
			Name:    "John Doe",
			Balance: 1000,
		},
		{
			ID:      "A2",
			Name:    "Jane Doe",
			Balance: 2000,
		},
		{
			ID:      "A3",
			Name:    "Jim Doe",
			Balance: 3000,
		},
		{
			ID:      "A4",
			Name:    "Jill Doe",
			Balance: 4000,
		},
		{
			ID:      "A5",
			Name:    "Jack Doe",
			Balance: 5000,
		},
	}
}
