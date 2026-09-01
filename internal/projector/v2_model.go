package projector

const (
	V2IRSchema          = "gooo/measurement-boundary/semantic-ir/v2"
	V2FixtureSchema     = "gooo/measurement-boundary/fixture/v2"
	V2CollectionSchema  = "gooo/measurement-boundary/collection/v2"
	V2ReceiptSchema     = "gooo/measurement-boundary/receipt/v2"
	V2EvaluationSchema  = "gooo/measurement-boundary/evaluation/v2"
	V2CorpusSchema      = "gooo/measurement-boundary/corpus/v2"
	V2ConformanceSchema = "gooo/measurement-boundary/conformance/v2"

	V2Closed  V2Decision = "CLOSED"
	V2Unknown V2Decision = "UNKNOWN"
	V2Refuted V2Decision = "REFUTED"
)

type V2Decision string

type V2CausalEvents struct {
	Start string `json:"start"`
	End   string `json:"end"`
}

type V2MeasurementSpec struct {
	MeasurementID          string            `json:"measurement_id"`
	StageID                string            `json:"stage_id"`
	Stage                  string            `json:"stage"`
	Step                   string            `json:"step"`
	CausalEvents           V2CausalEvents    `json:"causal_events"`
	CoveredOperations      []string          `json:"covered_operations"`
	ExcludedOperations     []string          `json:"excluded_operations"`
	ExpectedChildProcesses []string          `json:"expected_child_processes"`
	ChildProcessCoverage   string            `json:"child_process_coverage"`
	Clock                  string            `json:"clock"`
	ResolutionMS           int64             `json:"resolution_ms"`
	RSSProcessTreeScope    string            `json:"rss_process_tree_scope"`
	InputReceiptDigest     string            `json:"input_receipt_digest"`
	OutputReceiptDigest    string            `json:"output_receipt_digest"`
	Unit                   string            `json:"unit"`
	SourceAuthority        string            `json:"source_authority"`
	ObservationMethod      string            `json:"observation_method"`
	Scope                  string            `json:"scope"`
	IdentityDigests        map[string]string `json:"identity_digests"`
	Direction              string            `json:"direction"`
	NullablePolicy         string            `json:"nullable_policy"`
	ConflictPrecedence     []V2Decision      `json:"conflict_precedence"`
}

type V2OptionalObservation struct {
	ObservationID             string     `json:"observation_id"`
	Source                    string     `json:"source"`
	StageID                   string     `json:"stage_id"`
	Step                      string     `json:"step"`
	ActualMainLockWallMS      int64      `json:"actual_main_lock_wall_ms"`
	ProductReceiptBaselineMS  int64      `json:"product_receipt_baseline_wall_ms"`
	ProductReceiptCandidateMS int64      `json:"product_receipt_candidate_wall_ms"`
	Decision                  V2Decision `json:"decision"`
	Reason                    string     `json:"reason"`
	Acceptance                string     `json:"acceptance"`
	RequiredGate              bool       `json:"required_gate"`
	ImmutableInput            bool       `json:"immutable_input"`
}

type V2SemanticIR struct {
	Schema               string                  `json:"schema"`
	SourcePath           string                  `json:"source_path"`
	SourceDigest         string                  `json:"source_digest"`
	Namespace            string                  `json:"namespace"`
	Measurements         []V2MeasurementSpec     `json:"measurements"`
	OptionalObservations []V2OptionalObservation `json:"optional_observations,omitempty"`
	Digest               string                  `json:"digest"`
}

type V2Fixture struct {
	Schema               string             `json:"schema"`
	CaseID               string             `json:"case_id"`
	Name                 string             `json:"name"`
	CollectorAuthorities []string           `json:"collector_authorities,omitempty"`
	RuntimeAuthority     V2RuntimeAuthority `json:"runtime_authority"`
	Samples              []V2Sample         `json:"samples"`
}

type V2Sample struct {
	MetricID                  string            `json:"metric_id"`
	StageID                   string            `json:"stage_id"`
	Stage                     string            `json:"stage"`
	Step                      string            `json:"step"`
	StartEvent                string            `json:"start_event"`
	EndEvent                  string            `json:"end_event"`
	EndObserved               bool              `json:"end_observed"`
	StageEntered              bool              `json:"stage_entered"`
	CoveredOperations         []string          `json:"covered_operations"`
	CoveredChildProcesses     []string          `json:"covered_child_processes"`
	ChildProcessCoverage      string            `json:"child_process_coverage"`
	Clock                     string            `json:"clock"`
	ResolutionMS              int64             `json:"resolution_ms"`
	RSSProcessTreeScope       string            `json:"rss_process_tree_scope"`
	InputReceiptDigest        string            `json:"input_receipt_digest"`
	OutputReceiptDigest       string            `json:"output_receipt_digest"`
	Unit                      string            `json:"unit"`
	SourceAuthority           string            `json:"source_authority"`
	ObservationMethod         string            `json:"observation_method"`
	Scope                     string            `json:"scope"`
	IdentityDigests           map[string]string `json:"identity_digests"`
	Direction                 string            `json:"direction"`
	Measured                  bool              `json:"measured"`
	Value                     *int64            `json:"value"`
	WorkUnits                 *int64            `json:"work_units"`
	PeakRSSKiB                *int64            `json:"peak_rss_kib"`
	SourceArtifact            string            `json:"source_artifact"`
	ConsumerArtifacts         []string          `json:"consumer_artifacts"`
	ExternalUtilityEvidence   bool              `json:"external_utility_evidence"`
	OutputInsideReadOnlyInput bool              `json:"output_inside_read_only_input"`
	AuthorityEscalation       bool              `json:"authority_escalation"`
	Phase                     string            `json:"phase,omitempty"`
	PairID                    string            `json:"pair_id,omitempty"`
	ScenarioID                string            `json:"scenario_id,omitempty"`
	InputDigest               string            `json:"input_digest,omitempty"`
	ContractDigest            string            `json:"contract_digest,omitempty"`
	FixtureDigest             string            `json:"fixture_digest,omitempty"`
	Toolchain                 string            `json:"toolchain,omitempty"`
	Runner                    string            `json:"runner,omitempty"`
	Job                       string            `json:"job,omitempty"`
	TamperReceipt             bool              `json:"tamper_receipt"`
	TamperConsumer            bool              `json:"tamper_consumer"`
}

type V2RuntimeAuthority struct {
	RepositoryWrites          int `json:"repository_writes"`
	ApplyAuthority            int `json:"apply_authority"`
	CommitAuthority           int `json:"commit_authority"`
	MergeAuthority            int `json:"merge_authority"`
	TagAuthority              int `json:"tag_authority"`
	ReleaseAuthority          int `json:"release_authority"`
	LocalTestExecutions       int `json:"local_test_executions"`
	CrossProjectRequiredGates int `json:"cross_project_required_gates"`
}

type V2CollectorEvidence struct {
	Kind              string             `json:"kind"`
	Generated         bool               `json:"generated"`
	MeasuredOnce      bool               `json:"measured_once"`
	Authority         string             `json:"authority"`
	Authorities       []string           `json:"authorities"`
	IdentityDigest    string             `json:"identity_digest"`
	InputScope        string             `json:"input_scope"`
	OutputScope       string             `json:"output_scope"`
	OperatorAuthority string             `json:"operator_authority"`
	RuntimeAuthority  V2RuntimeAuthority `json:"runtime_authority"`
}

type V2CollectedObservation struct {
	MetricID                  string            `json:"metric_id"`
	StageID                   string            `json:"stage_id"`
	Stage                     string            `json:"stage"`
	Step                      string            `json:"step"`
	StartEvent                string            `json:"start_event"`
	EndEvent                  string            `json:"end_event"`
	EndObserved               bool              `json:"end_observed"`
	StageEntered              bool              `json:"stage_entered"`
	CoveredOperations         []string          `json:"covered_operations"`
	CoveredChildProcesses     []string          `json:"covered_child_processes"`
	ChildProcessCoverage      string            `json:"child_process_coverage"`
	Clock                     string            `json:"clock"`
	ResolutionMS              int64             `json:"resolution_ms"`
	RSSProcessTreeScope       string            `json:"rss_process_tree_scope"`
	InputReceiptDigest        string            `json:"input_receipt_digest"`
	OutputReceiptDigest       string            `json:"output_receipt_digest"`
	Unit                      string            `json:"unit"`
	SourceAuthority           string            `json:"source_authority"`
	ObservationMethod         string            `json:"observation_method"`
	Scope                     string            `json:"scope"`
	IdentityDigests           map[string]string `json:"identity_digests"`
	Direction                 string            `json:"direction"`
	Measured                  bool              `json:"measured"`
	Value                     *int64            `json:"value"`
	WorkUnits                 *int64            `json:"work_units"`
	PeakRSSKiB                *int64            `json:"peak_rss_kib"`
	SourceArtifact            string            `json:"source_artifact"`
	ConsumerArtifacts         []string          `json:"consumer_artifacts"`
	ExternalUtilityEvidence   bool              `json:"external_utility_evidence"`
	OutputInsideReadOnlyInput bool              `json:"output_inside_read_only_input"`
	AuthorityEscalation       bool              `json:"authority_escalation"`
	Phase                     string            `json:"phase,omitempty"`
	PairID                    string            `json:"pair_id,omitempty"`
	ScenarioID                string            `json:"scenario_id,omitempty"`
	InputDigest               string            `json:"input_digest,omitempty"`
	ContractDigest            string            `json:"contract_digest,omitempty"`
	FixtureDigest             string            `json:"fixture_digest,omitempty"`
	Toolchain                 string            `json:"toolchain,omitempty"`
	Runner                    string            `json:"runner,omitempty"`
	Job                       string            `json:"job,omitempty"`
	ReceiptDigest             string            `json:"receipt_digest"`
}

type V2Receipt struct {
	Schema                    string            `json:"schema"`
	MetricID                  string            `json:"metric_id"`
	StageID                   string            `json:"stage_id"`
	Stage                     string            `json:"stage"`
	Step                      string            `json:"step"`
	CausalEvents              V2CausalEvents    `json:"causal_events"`
	EndObserved               bool              `json:"end_observed"`
	StageEntered              bool              `json:"stage_entered"`
	CoveredOperations         []string          `json:"covered_operations"`
	CoveredChildProcesses     []string          `json:"covered_child_processes"`
	ChildProcessCoverage      string            `json:"child_process_coverage"`
	Clock                     string            `json:"clock"`
	ResolutionMS              int64             `json:"resolution_ms"`
	RSSProcessTreeScope       string            `json:"rss_process_tree_scope"`
	InputReceiptDigest        string            `json:"input_receipt_digest"`
	OutputReceiptDigest       string            `json:"output_receipt_digest"`
	Unit                      string            `json:"unit"`
	Scope                     string            `json:"scope"`
	SourceAuthority           string            `json:"source_authority"`
	IdentityDigests           map[string]string `json:"identity_digests"`
	SourceArtifact            string            `json:"source_artifact"`
	Measured                  bool              `json:"measured"`
	Value                     *int64            `json:"value"`
	WorkUnits                 *int64            `json:"work_units"`
	PeakRSSKiB                *int64            `json:"peak_rss_kib"`
	ObservationMethod         string            `json:"observation_method"`
	Direction                 string            `json:"direction"`
	ExternalUtilityEvidence   bool              `json:"external_utility_evidence"`
	OutputInsideReadOnlyInput bool              `json:"output_inside_read_only_input"`
	AuthorityEscalation       bool              `json:"authority_escalation"`
	ScenarioID                string            `json:"scenario_id,omitempty"`
	InputDigest               string            `json:"input_digest,omitempty"`
	ContractDigest            string            `json:"contract_digest,omitempty"`
	FixtureDigest             string            `json:"fixture_digest,omitempty"`
	Toolchain                 string            `json:"toolchain,omitempty"`
	Runner                    string            `json:"runner,omitempty"`
	Job                       string            `json:"job,omitempty"`
	Phase                     string            `json:"phase,omitempty"`
	PairID                    string            `json:"pair_id,omitempty"`
	ObservationDigest         string            `json:"observation_digest"`
	ReceiptDigest             string            `json:"receipt_digest"`
}

type V2ConsumerArtifact struct {
	Name              string   `json:"name"`
	MetricID          string   `json:"metric_id"`
	StageID           string   `json:"stage_id"`
	CoveredOperations []string `json:"covered_operations"`
	ReceiptDigest     string   `json:"receipt_digest"`
}

type V2Collection struct {
	Schema        string                   `json:"schema"`
	IRDigest      string                   `json:"ir_digest"`
	FixtureDigest string                   `json:"fixture_digest"`
	Collector     V2CollectorEvidence      `json:"collector"`
	Observations  []V2CollectedObservation `json:"observations"`
	Receipts      []V2Receipt              `json:"receipts"`
	Consumers     []V2ConsumerArtifact     `json:"consumers"`
	Digest        string                   `json:"digest"`
}

type V2UnknownFrontier struct {
	Stage         string   `json:"stage"`
	Step          string   `json:"step"`
	Reason        string   `json:"reason"`
	UnknownClass  string   `json:"unknown_class"`
	NextOperation string   `json:"next_operation"`
	BlockedBy     []string `json:"blocked_by"`
}

type V2ObservedValue struct {
	Authority      string `json:"authority"`
	SourceArtifact string `json:"source_artifact"`
	Value          *int64 `json:"value"`
	WorkUnits      *int64 `json:"work_units"`
	PeakRSSKiB     *int64 `json:"peak_rss_kib"`
	Unit           string `json:"unit"`
	Scope          string `json:"scope"`
	StageID        string `json:"stage_id"`
	ReceiptDigest  string `json:"receipt_digest"`
}

type V2Improvement struct {
	State   V2Decision         `json:"state"`
	Reason  string             `json:"reason"`
	PairID  string             `json:"pair_id,omitempty"`
	Before  *int64             `json:"before,omitempty"`
	After   *int64             `json:"after,omitempty"`
	Delta   *int64             `json:"delta,omitempty"`
	Unknown *V2UnknownFrontier `json:"unknown,omitempty"`
}

type V2MetricResult struct {
	MeasurementID         string             `json:"measurement_id"`
	State                 V2Decision         `json:"state"`
	StageID               string             `json:"stage_id"`
	Stage                 string             `json:"stage"`
	Step                  string             `json:"step"`
	CausalEvents          V2CausalEvents     `json:"causal_events"`
	CoveredOperations     []string           `json:"covered_operations"`
	CoveredChildProcesses []string           `json:"covered_child_processes"`
	ChildProcessCoverage  string             `json:"child_process_coverage"`
	Clock                 string             `json:"clock"`
	ResolutionMS          int64              `json:"resolution_ms"`
	RSSProcessTreeScope   string             `json:"rss_process_tree_scope"`
	InputReceiptDigest    string             `json:"input_receipt_digest"`
	OutputReceiptDigest   string             `json:"output_receipt_digest"`
	Reason                string             `json:"reason"`
	Value                 *int64             `json:"value"`
	WorkUnits             *int64             `json:"work_units"`
	PeakRSSKiB            *int64             `json:"peak_rss_kib"`
	Unit                  string             `json:"unit"`
	Scope                 string             `json:"scope"`
	Authority             string             `json:"authority"`
	Authorities           []string           `json:"authorities"`
	Scopes                []string           `json:"scopes"`
	IdentityDigests       map[string]string  `json:"identity_digests"`
	ReceiptDigests        []string           `json:"receipt_digests"`
	ConsumerArtifacts     []string           `json:"consumer_artifacts"`
	ObservedValues        []V2ObservedValue  `json:"observed_values"`
	Improvement           V2Improvement      `json:"improvement"`
	Unknown               *V2UnknownFrontier `json:"unknown,omitempty"`
}

type V2Claim struct {
	State         V2Decision `json:"state"`
	StageID       string     `json:"stage_id"`
	Stage         string     `json:"stage"`
	Step          string     `json:"step"`
	Reason        string     `json:"reason"`
	UnknownClass  string     `json:"unknown_class"`
	NextOperation string     `json:"next_operation"`
	BlockedBy     []string   `json:"blocked_by"`
}

type V2Evaluation struct {
	Schema           string           `json:"schema"`
	IRDigest         string           `json:"ir_digest"`
	CollectionDigest string           `json:"collection_digest"`
	Decision         V2Decision       `json:"decision"`
	FailClosed       bool             `json:"fail_closed"`
	Claim            V2Claim          `json:"claim"`
	Metrics          []V2MetricResult `json:"metrics"`
	ClosedCount      int              `json:"closed_count"`
	UnknownCount     int              `json:"unknown_count"`
	RefutedCount     int              `json:"refuted_count"`
	AggregatePolicy  string           `json:"aggregate_policy"`
}

type V2Corpus struct {
	Schema string          `json:"schema"`
	Cases  []V2CorpusEntry `json:"cases"`
}

type V2CorpusEntry struct {
	Ordinal     int        `json:"ordinal"`
	CaseID      string     `json:"case_id"`
	Path        string     `json:"path"`
	Expected    V2Decision `json:"expected"`
	Description string     `json:"description"`
}

type V2TestVector struct {
	Ordinal          int        `json:"ordinal"`
	CaseID           string     `json:"case_id"`
	Expected         V2Decision `json:"expected"`
	Observed         V2Decision `json:"observed"`
	FixtureDigest    string     `json:"fixture_digest"`
	EvaluationDigest string     `json:"evaluation_digest"`
}

type V2PairVector struct {
	MetricID       string `json:"metric_id"`
	PairID         string `json:"pair_id"`
	ScenarioID     string `json:"scenario_id"`
	InputDigest    string `json:"input_digest"`
	ContractDigest string `json:"contract_digest"`
	FixtureDigest  string `json:"fixture_digest"`
	Toolchain      string `json:"toolchain"`
	Runner         string `json:"runner"`
	Job            string `json:"job"`
	Before         int64  `json:"before"`
	After          int64  `json:"after"`
}

type V2ConformanceSummary struct {
	Schema               string                  `json:"schema"`
	Total                int                     `json:"total"`
	Closed               int                     `json:"closed"`
	Unknown              int                     `json:"unknown"`
	Refuted              int                     `json:"refuted"`
	Tests                []V2TestVector          `json:"tests"`
	ControlledPairs      []V2PairVector          `json:"controlled_pairs"`
	OptionalObservations []V2OptionalObservation `json:"optional_observations"`
	RuntimeAuthority     V2RuntimeAuthority      `json:"runtime_authority"`
}
