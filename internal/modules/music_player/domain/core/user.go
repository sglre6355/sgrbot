package core

// UserID uniquely identifies a user.
type UserID string

// ParseUserID converts a string to a UserID.
func ParseUserID(id string) (UserID, error) {
	return UserID(id), nil
}

// String returns the UserID as a string.
func (id UserID) String() string {
	return string(id)
}

// User represents a user with display information.
type User struct {
	ID        UserID
	Name      string
	AvatarURL string
}

// UserRepository defines the interface for retrieving users by ID.
type UserRepository interface {
	// FindByID returns the User for the given ID, or error if not found.
	FindByID(userID UserID) (User, error)
}
