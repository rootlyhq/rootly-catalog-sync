package catalog

type DesiredEntity struct {
	ExternalID string
	Name       string
	Fields     map[string]string
}

type LiveEntity struct {
	ID          string
	ExternalID  string
	Name        string
	Description string
	ManagedBy   string
	Fields      map[string]string
}
