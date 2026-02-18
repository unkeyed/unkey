package engine

import (
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"sync"

	sentinelv1 "github.com/unkeyed/unkey/gen/proto/sentinel/v1"
)

// regexCache caches compiled regular expressions to avoid recompilation.
type regexCache struct {
	mu    sync.RWMutex
	cache map[string]*regexp.Regexp
}

func newRegexCache() *regexCache {
	//nolint:exhaustruct
	return &regexCache{
		cache: make(map[string]*regexp.Regexp),
	}
}

func (rc *regexCache) get(pattern string) (*regexp.Regexp, error) {
	rc.mu.RLock()
	re, ok := rc.cache[pattern]
	rc.mu.RUnlock()
	if ok {
		return re, nil
	}

	re, err := regexp.Compile(pattern)
	if err != nil {
		return nil, fmt.Errorf("invalid regex %q: %w", pattern, err)
	}

	rc.mu.Lock()
	rc.cache[pattern] = re
	rc.mu.Unlock()
	return re, nil
}

// matchesRequest evaluates all match expressions against the request.
// All expressions must match (AND semantics). An empty list matches all requests.
func matchesRequest(req *http.Request, exprs []*sentinelv1.MatchExpr, rc *regexCache) (bool, error) {
	for _, expr := range exprs {
		matched, err := evalMatchExpr(req, expr, rc)
		if err != nil {
			return false, err
		}
		if !matched {
			return false, nil
		}
	}
	return true, nil
}

func evalMatchExpr(req *http.Request, expr *sentinelv1.MatchExpr, rc *regexCache) (bool, error) {
	switch e := expr.GetExpr().(type) {
	case *sentinelv1.MatchExpr_Path:
		return evalPathMatch(req, e.Path, rc)
	case *sentinelv1.MatchExpr_Method:
		return evalMethodMatch(req, e.Method), nil
	case *sentinelv1.MatchExpr_Header:
		return evalHeaderMatch(req, e.Header, rc)
	case *sentinelv1.MatchExpr_QueryParam:
		return evalQueryParamMatch(req, e.QueryParam, rc)
	default:
		return false, nil
	}
}

func evalPathMatch(req *http.Request, pm *sentinelv1.PathMatch, rc *regexCache) (bool, error) {
	if pm == nil || pm.GetPath() == nil {
		return true, nil
	}
	return evalStringMatch(req.URL.Path, pm.GetPath(), rc)
}

// evalMethodMatch checks if the request method matches any of the specified methods.
// Always case-insensitive per HTTP spec. OR semantics across the list.
func evalMethodMatch(req *http.Request, mm *sentinelv1.MethodMatch) bool {
	if mm == nil || len(mm.GetMethods()) == 0 {
		return true
	}
	for _, m := range mm.GetMethods() {
		if strings.EqualFold(req.Method, m) {
			return true
		}
	}
	return false
}

func evalHeaderMatch(req *http.Request, hm *sentinelv1.HeaderMatch, rc *regexCache) (bool, error) {
	if hm == nil {
		return true, nil
	}
	values := req.Header.Values(hm.GetName())
	switch m := hm.GetMatch().(type) {
	case *sentinelv1.HeaderMatch_Present:
		return m.Present == (len(values) > 0), nil
	case *sentinelv1.HeaderMatch_Value:
		for _, v := range values {
			matched, err := evalStringMatch(v, m.Value, rc)
			if err != nil {
				return false, err
			}
			if matched {
				return true, nil
			}
		}
		return false, nil
	default:
		// No match specified, just check presence
		return len(values) > 0, nil
	}
}

func evalQueryParamMatch(req *http.Request, qm *sentinelv1.QueryParamMatch, rc *regexCache) (bool, error) {
	if qm == nil {
		return true, nil
	}
	values, exists := req.URL.Query()[qm.GetName()]
	switch m := qm.GetMatch().(type) {
	case *sentinelv1.QueryParamMatch_Present:
		return m.Present == exists, nil
	case *sentinelv1.QueryParamMatch_Value:
		for _, v := range values {
			matched, err := evalStringMatch(v, m.Value, rc)
			if err != nil {
				return false, err
			}
			if matched {
				return true, nil
			}
		}
		return false, nil
	default:
		return exists, nil
	}
}

// evalStringMatch evaluates a value against a StringMatch (exact, prefix, or regex).
func evalStringMatch(value string, sm *sentinelv1.StringMatch, rc *regexCache) (bool, error) {
	if sm == nil {
		return true, nil
	}

	switch m := sm.GetMatch().(type) {
	case *sentinelv1.StringMatch_Exact:
		if sm.GetIgnoreCase() {
			return strings.EqualFold(value, m.Exact), nil
		}
		return value == m.Exact, nil

	case *sentinelv1.StringMatch_Prefix:
		if sm.GetIgnoreCase() {
			return len(value) >= len(m.Prefix) && strings.EqualFold(value[:len(m.Prefix)], m.Prefix), nil
		}
		return strings.HasPrefix(value, m.Prefix), nil

	case *sentinelv1.StringMatch_Regex:
		pattern := m.Regex
		if sm.GetIgnoreCase() {
			pattern = "(?i)" + pattern
		}
		re, err := rc.get(pattern)
		if err != nil {
			return false, err
		}
		return re.MatchString(value), nil

	default:
		return true, nil
	}
}
