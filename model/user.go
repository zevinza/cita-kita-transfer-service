package model

type User struct {
	ID      string
	Name    string
	Balance int64
}

func SeedUsers() []User {
	return []User{}
}
