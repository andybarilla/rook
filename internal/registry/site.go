package registry

type Site struct {
	Path        string `json:"path"`
	Domain      string `json:"domain"`
	PHPVersion  string `json:"php_version,omitempty"`
	NodeVersion string `json:"node_version,omitempty"`
	TLS         bool   `json:"tls"`
}

type EventType int

const (
	SiteAdded EventType = iota
	SiteRemoved
	SiteUpdated
)

type ChangeEvent struct {
	Type    EventType
	Site    Site
	OldSite *Site
}
