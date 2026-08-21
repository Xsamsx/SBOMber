package supplychain

// RiskType identifies a supply-chain finding category.
type RiskType string

const (
	RiskMalware             RiskType = "malware"
	RiskDependencyConfusion RiskType = "dependency_confusion"
	RiskRegistryMissing     RiskType = "registry_not_found"
)

// RiskFinding describes a malware or dependency-confusion style issue.
type RiskFinding struct {
	Type      RiskType `json:"type"`
	Package   string   `json:"package"`
	Version   string   `json:"version,omitempty"`
	Ecosystem string   `json:"ecosystem"`
	Severity  string   `json:"severity"`
	Message   string   `json:"message"`
	Source    string   `json:"source,omitempty"`
	Reference string   `json:"reference,omitempty"`
	IsDirect  bool     `json:"is_direct"`
}
