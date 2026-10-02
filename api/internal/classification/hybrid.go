package classification

import "context"

func Hybrid(ctx context.Context, in Input, classifier DocumentClassifier, auto, review float64) (Result, bool, error) {
	rule := ClassifyRule(in, review)
	if auto <= 0 || auto >= 1 {
		auto = .9
	}
	if classifier != nil && rule.Confidence < auto {
		candidate, err := classifier.Classify(ctx, in)
		if err == nil {
			Review(&candidate, auto, review)
			return candidate, true, nil
		}
		Review(&rule, auto, review)
		return rule, false, err
	}
	Review(&rule, auto, review)
	return rule, false, nil
}
