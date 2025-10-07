package ros_topic_watcher

//type Config struct {
//	Alerts []AlertRule `yaml:"alerts"`
//}
//
//type AlertRule struct {
//	TopicName   string `yaml:"topic_name"`
//	Description string `yaml:"description"`
//	Kind        Kind   `yaml:"kind"` // threshold | deadman | composite
//
//	// threshold
//	When *ThresholdExpr `yaml:"when,omitempty"`
//	For  Duration       `yaml:"for,omitempty"`
//
//	// deadman
//	Source    string    `yaml:"source,omitempty"`
//	NoDataFor *Duration `yaml:"no_data_for,omitempty"`
//
//	// composite
//	All []string `yaml:"all,omitempty"`
//	Any []string `yaml:"any,omitempty"`
//
//	// common
//	Severity Severity  `yaml:"severity"`           // info | warn | critical
//	Notify   []string  `yaml:"notify"`             // channels
//	Cooldown *Duration `yaml:"cooldown,omitempty"` // optional override
//	Once     bool      `yaml:"once,omitempty"`     // fire only first time until cleared
//}

type ThresholdExpr struct {
	Left  string   `yaml:"left"`  // e.g., "bms.battery_pct"
	Op    Operator `yaml:"op"`    // < | <= | > | >= | == | !=
	Right float64  `yaml:"right"` // numeric threshold
}

type Kind string

const (
	KindThreshold Kind = "threshold"
	KindDeadman   Kind = "deadman"
	KindComposite Kind = "composite"
)

//func (k *Kind) UnmarshalYAML(n *yaml.Node) error {
//	var s string
//	if err := n.Decode(&s); err != nil {
//		return err
//	}
//	switch Kind(s) {
//	case KindThreshold, KindDeadman, KindComposite:
//		*k = Kind(s)
//		return nil
//	default:
//		return fmt.Errorf("invalid kind: %q", s)
//	}
//}

type Severity string

const (
	SeverityInfo     Severity = "info"
	SeverityWarn     Severity = "warn"
	SeverityCritical Severity = "critical"
)

//func (s *Severity) UnmarshalYAML(n *yaml.Node) error {
//	var v string
//	if err := n.Decode(&v); err != nil {
//		return err
//	}
//	switch Severity(v) {
//	case SeverityInfo, SeverityWarn, SeverityCritical:
//		*s = Severity(v)
//		return nil
//	default:
//		return fmt.Errorf("invalid severity: %q", v)
//	}
//}

type Operator string

const (
	OpLT  Operator = "<"
	OpLTE Operator = "<="
	OpGT  Operator = ">"
	OpGTE Operator = ">="
	OpEQ  Operator = "=="
	OpNEQ Operator = "!="
)

//func (o *Operator) UnmarshalYAML(n *yaml.Node) error {
//	var v string
//	if err := n.Decode(&v); err != nil {
//		return err
//	}
//	switch Operator(v) {
//	case OpLT, OpLTE, OpGT, OpGTE, OpEQ, OpNEQ:
//		*o = Operator(v)
//		return nil
//	default:
//		return fmt.Errorf("invalid operator: %q", v)
//	}
//}
//
//// ----- duration helper -----
//
//// Duration wraps time.Duration to support YAML strings like "10s", "5m".
//type Duration struct {
//	time.Duration
//}
//
//func (d *Duration) UnmarshalYAML(n *yaml.Decoder) error {
//	// Accept string ("10s") or number (seconds)
//	switch n.Tag {
//	case "!!str":
//		var s string
//		if err := n.Decode(&s); err != nil {
//			return err
//		}
//		dd, err := time.ParseDuration(s)
//		if err != nil {
//			return fmt.Errorf("invalid duration %q: %w", s, err)
//		}
//		d.Duration = dd
//		return nil
//	case "!!int", "!!float":
//		var sec float64
//		if err := n.Decode(&sec); err != nil {
//			return err
//		}
//		d.Duration = time.Duration(sec * float64(time.Second))
//		return nil
//	default:
//		return fmt.Errorf("unsupported duration type: %s", n.Tag)
//	}
//}
