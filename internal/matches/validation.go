package matches

// ResolutionVerdict captures the validator's outcome for a pair of markets.
type ResolutionVerdict struct {
	ValidResolution  bool   `json:"ValidResolution"`
	ResolutionReason string `json:"ResolutionReason"`
}

// NewResolutionVerdict builds a verdict struct.
func NewResolutionVerdict(valid bool, reason string) *ResolutionVerdict {
	return &ResolutionVerdict{
		ValidResolution:  valid,
		ResolutionReason: reason,
	}
}
