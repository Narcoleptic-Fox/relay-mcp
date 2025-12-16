package memory

// ErrThreadNotFound indicates the thread doesn't exist
type ErrThreadNotFound struct {
	ThreadID string
}

func (e ErrThreadNotFound) Error() string {
	return "thread not found: " + e.ThreadID
}

// ErrThreadExpired indicates the thread has expired
type ErrThreadExpired struct {
	ThreadID string
}

func (e ErrThreadExpired) Error() string {
	return "thread expired: " + e.ThreadID
}
