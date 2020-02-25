package redispool

const (
	// TagProduction is a tag for a production database
	TagProduction = "production"
	// TagCandidate is a tag for a new database which should be checked
	TagCandidate = "candidate"
	// TagTarget is a tag for a potentially new database
	// If TagCandidate is specified then TagTarget will be the same
	TagTarget = "target"

	// fieldDatabaseName is field to store a project name
	fieldDatabaseName = "database:name"
	// fieldDatabaseTag is field  to store a database tag
	fieldDatabaseTag = "database:tag"
)

// DatabaseInfo is a struct with information about redis database
type DatabaseInfo struct {
	Number int
	Name   string
	Tag    string
}

// Databases is a list with information about redis databases
type Databases struct {
	List       []DatabaseInfo
	Production int
	Candidate  int
	Target     int
}
