package projector

const (
	IRSchema         = "gooo/measurement-boundary/semantic-ir/v1"
	CollectionSchema = "gooo/measurement-boundary/collection/v1"
	ReceiptSchema    = "gooo/measurement-boundary/receipt/v1"
	EvaluationSchema = "gooo/measurement-boundary/evaluation/v1"
	CorpusSchema     = "gooo/measurement-boundary/corpus/v1"
	EvidenceSchema   = "gooo/measurement-boundary/ci-evidence/v1"

	Closed  Decision = "CLOSED"
	Unknown Decision = "UNKNOWN"
	Refuted Decision = "REFUTED"
)

type Decision string

type Span struct {
	StartBoundary string `json:"start_boundary"`
	EndBoundary   string `json:"end_boundary"`
}

type MeasurementSpec struct {
	MeasurementID      string            `json:"measurement_id"`
	Stage              string            `json:"stage"`
	Step               string            `json:"step"`
	Span               Span              `json:"span"`
	IncludedOperations []string          `json:"included_operations"`
	ExcludedOperations []string          `json:"excluded_operations"`
	Unit               string            `json:"unit"`
	SourceAuthority    string            `json:"source_authority"`
	ObservationMethod  string            `json:"observation_method"`
	Scope              string            `json:"scope"`
	IdentityDigests    map[string]string `json:"identity_digests"`
	Direction          string            `json:"direction"`
	NullablePolicy     string            `json:"nullable_policy"`
	ConflictPrecedence []Decision        `json:"conflict_precedence"`
}

type SemanticIR struct {
	Schema       string            `json:"schema"`
	SourcePath   string            `json:"source_path"`
	SourceDigest string            `json:"source_digest"`
	Measurements []MeasurementSpec `json:"measurements"`
	Digest       string            `json:"digest"`
}

type Fixture struct {
	Schema  string   `json:"schema"`
	CaseID  string   `json:"case_id"`
	Name    string   `json:"name"`
	Samples []Sample `json:"samples"`
}

type Sample struct {
	MetricID                string            `json:"metric_id"`
	Stage                   string            `json:"stage"`
	Step                    string            `json:"step"`
	StartBoundary           string            `json:"start_boundary"`
	EndBoundary             string            `json:"end_boundary"`
	IncludedOperations      []string          `json:"included_operations"`
	Unit                    string            `json:"unit"`
	SourceAuthority         string            `json:"source_authority"`
	ObservationMethod       string            `json:"observation_method"`
	Scope                   string            `json:"scope"`
	IdentityDigests         map[string]string `json:"identity_digests"`
	Direction               string            `json:"direction"`
	Measured                bool              `json:"measured"`
	Value                   *float64          `json:"value"`
	SourceArtifact          string            `json:"source_artifact"`
	ConsumerArtifacts       []string          `json:"consumer_artifacts"`
	ExternalUtilityEvidence bool              `json:"external_utility_evidence"`
	TamperReceipt           bool              `json:"tamper_receipt"`
	TamperConsumer          bool              `json:"tamper_consumer"`
	Contradiction           bool              `json:"contradiction"`
	Phase                   string            `json:"phase,omitempty"`
	PairID                  string            `json:"pair_id,omitempty"`
}

type Collection struct {
	Schema        string                 `json:"schema"`
	IRDigest      string                 `json:"ir_digest"`
	FixtureDigest string                 `json:"fixture_digest"`
	Collector     CollectorEvidence      `json:"collector"`
	Observations  []CollectedObservation `json:"observations"`
	Receipts      []Receipt              `json:"receipts"`
	Consumers     []ConsumerArtifact     `json:"consumers"`
	Digest        string                 `json:"digest"`
}

type CollectorEvidence struct {
	Kind             string `json:"kind"`
	Generated        bool   `json:"generated"`
	MeasuredOnce     bool   `json:"measured_once"`
	IdentityDigest   string `json:"identity_digest"`
	OutputScope      string `json:"output_scope"`
	RepositoryWrites int    `json:"repository_writes"`
	ApplyAuthority   int    `json:"apply_authority"`
	CommitAuthority  int    `json:"commit_authority"`
	MergeAuthority   int    `json:"merge_authority"`
	TagAuthority     int    `json:"tag_authority"`
	ReleaseAuthority int    `json:"release_authority"`
}

type CollectedObservation struct {
	MetricID                string            `json:"metric_id"`
	Stage                   string            `json:"stage"`
	Step                    string            `json:"step"`
	StartBoundary           string            `json:"start_boundary"`
	EndBoundary             string            `json:"end_boundary"`
	IncludedOperations      []string          `json:"included_operations"`
	Unit                    string            `json:"unit"`
	SourceAuthority         string            `json:"source_authority"`
	ObservationMethod       string            `json:"observation_method"`
	Scope                   string            `json:"scope"`
	IdentityDigests         map[string]string `json:"identity_digests"`
	Direction               string            `json:"direction"`
	Measured                bool              `json:"measured"`
	Value                   *float64          `json:"value"`
	SourceArtifact          string            `json:"source_artifact"`
	ConsumerArtifacts       []string          `json:"consumer_artifacts"`
	ExternalUtilityEvidence bool              `json:"external_utility_evidence"`
	Contradiction           bool              `json:"contradiction"`
	Phase                   string            `json:"phase,omitempty"`
	PairID                  string            `json:"pair_id,omitempty"`
	ReceiptDigest           string            `json:"receipt_digest"`
}

type Receipt struct {
	Schema            string            `json:"schema"`
	MetricID          string            `json:"metric_id"`
	Stage             string            `json:"stage"`
	Step              string            `json:"step"`
	Unit              string            `json:"unit"`
	Scope             string            `json:"scope"`
	SourceAuthority   string            `json:"source_authority"`
	IdentityDigests   map[string]string `json:"identity_digests"`
	SourceArtifact    string            `json:"source_artifact"`
	Measured          bool              `json:"measured"`
	Value             *float64          `json:"value"`
	ObservationDigest string            `json:"observation_digest"`
	ReceiptDigest     string            `json:"receipt_digest"`
}

type ConsumerArtifact struct {
	Name          string `json:"name"`
	MetricID      string `json:"metric_id"`
	ReceiptDigest string `json:"receipt_digest"`
}

type UnknownFrontier struct {
	Stage         string   `json:"stage"`
	Step          string   `json:"step"`
	Reason        string   `json:"reason"`
	UnknownClass  string   `json:"unknown_class"`
	NextOperation string   `json:"next_operation"`
	BlockedBy     []string `json:"blocked_by"`
}

type MetricResult struct {
	MeasurementID     string            `json:"measurement_id"`
	State             Decision          `json:"state"`
	Stage             string            `json:"stage"`
	Step              string            `json:"step"`
	Reason            string            `json:"reason"`
	Value             *float64          `json:"value"`
	Unit              string            `json:"unit"`
	Scope             string            `json:"scope"`
	Authority         string            `json:"authority"`
	Authorities       []string          `json:"authorities"`
	Scopes            []string          `json:"scopes"`
	Units             []string          `json:"units"`
	IdentityDigests   map[string]string `json:"identity_digests"`
	ReceiptDigests    []string          `json:"receipt_digests"`
	ConsumerArtifacts []string          `json:"consumer_artifacts"`
	ObservedValues    []ObservedValue   `json:"observed_values"`
	Unknown           *UnknownFrontier  `json:"unknown,omitempty"`
	Before            *float64          `json:"before,omitempty"`
	After             *float64          `json:"after,omitempty"`
	Delta             *float64          `json:"delta,omitempty"`
}

type ObservedValue struct {
	Authority      string   `json:"authority"`
	SourceArtifact string   `json:"source_artifact"`
	Value          *float64 `json:"value"`
	Unit           string   `json:"unit"`
	Scope          string   `json:"scope"`
	ReceiptDigest  string   `json:"receipt_digest"`
}

type Claim struct {
	State         Decision `json:"state"`
	Stage         string   `json:"stage"`
	Step          string   `json:"step"`
	Reason        string   `json:"reason"`
	UnknownClass  string   `json:"unknown_class"`
	NextOperation string   `json:"next_operation"`
	BlockedBy     []string `json:"blocked_by"`
}

type Evaluation struct {
	Schema           string         `json:"schema"`
	IRDigest         string         `json:"ir_digest"`
	CollectionDigest string         `json:"collection_digest"`
	Decision         Decision       `json:"decision"`
	FailClosed       bool           `json:"fail_closed"`
	Claim            Claim          `json:"claim"`
	Metrics          []MetricResult `json:"metrics"`
	ClosedCount      int            `json:"closed_count"`
	UnknownCount     int            `json:"unknown_count"`
	RefutedCount     int            `json:"refuted_count"`
	AggregatePolicy  string         `json:"aggregate_policy"`
}

type Corpus struct {
	Schema string        `json:"schema"`
	Cases  []CorpusEntry `json:"cases"`
}

type CorpusEntry struct {
	Ordinal     int      `json:"ordinal"`
	CaseID      string   `json:"case_id"`
	Path        string   `json:"path"`
	Expected    Decision `json:"expected"`
	Description string   `json:"description"`
}

type ConformanceSummary struct {
	Schema            string            `json:"schema"`
	Total             int               `json:"total"`
	Selected          int               `json:"selected"`
	Executed          int               `json:"executed"`
	Reused            int               `json:"reused"`
	Closed            int               `json:"closed"`
	Unknown           int               `json:"unknown"`
	Refuted           int               `json:"refuted"`
	Tests             []TestVector      `json:"tests"`
	GeneratedEvidence GeneratedEvidence `json:"generated_evidence"`
}

type TestVector struct {
	Ordinal          int      `json:"ordinal"`
	CaseID           string   `json:"case_id"`
	Expected         Decision `json:"expected"`
	Observed         Decision `json:"observed"`
	FixtureDigest    string   `json:"fixture_digest"`
	EvaluationDigest string   `json:"evaluation_digest"`
}

type GeneratedEvidence struct {
	FixedCaseID           string `json:"fixed_case_id"`
	GeneratedCollectorRan bool   `json:"generated_collector_ran"`
	MeasuredOncePerMetric bool   `json:"measured_once_per_metric"`
	ConsumersShareReceipt bool   `json:"consumers_share_receipt"`
	ReceiptDigest         string `json:"receipt_digest"`
}

type StageMeasurement struct {
	Stage      string `json:"stage"`
	WallMS     int64  `json:"wall_ms"`
	PeakRSSKiB int64  `json:"peak_rss_kib"`
}

type Inventory struct {
	RootReadmeExcluded bool  `json:"root_readme_excluded"`
	Directories        int   `json:"directories"`
	RegularFiles       int   `json:"regular_files"`
	TreeBytes          int64 `json:"tree_bytes"`
	GoFiles            int   `json:"go_files"`
	GoPhysicalLines    int64 `json:"go_physical_lines"`
	GoooFiles          int   `json:"gooo_files"`
	GoooPhysicalLines  int64 `json:"gooo_physical_lines"`
	GeneratedArtifacts int   `json:"generated_artifacts"`
	GeneratedBytes     int64 `json:"generated_bytes"`
}

type UtilityState struct {
	ID     string   `json:"id"`
	State  Decision `json:"state"`
	Reason string   `json:"reason"`
}

type ImprovementState struct {
	ID     string   `json:"id"`
	State  Decision `json:"state"`
	Reason string   `json:"reason"`
}
