package domain

// User описывает сущность пользователя в нашей системе
type User struct {
	ID           int64
	Email        string
	PasswordHash string // Мы никогда не храним пароль в открытом виде!
}
