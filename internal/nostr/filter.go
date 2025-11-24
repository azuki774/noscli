package nostr

import "time"

// Filter mirrors a standard Nostr REQ filter.
type Filter struct {
	Authors []string
	Kinds   []int
	Since   *time.Time
	Until   *time.Time
	Limit   int
}

func (f Filter) toRequest() map[string]any {
	payload := make(map[string]any)

	if len(f.Authors) > 0 {
		payload["authors"] = f.Authors
	}
	if len(f.Kinds) > 0 {
		payload["kinds"] = f.Kinds
	}
	if f.Since != nil {
		payload["since"] = f.Since.Unix()
	}
	if f.Until != nil {
		payload["until"] = f.Until.Unix()
	}
	if f.Limit > 0 {
		payload["limit"] = f.Limit
	}

	return payload
}
