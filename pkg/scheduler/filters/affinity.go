package filters

import (
	"fmt"
	"strings"

	"github.com/coco-sandbox/coco/pkg/scheduler"
)

type AffinityFilter struct {
	rules []AffinityRule
}

type AffinityRule struct {
	Label      string
	Value      string
	Preference AffinityPreference
	Weight     int
}

type AffinityPreference int

const (
	Preferred AffinityPreference = iota
	Required
	PreferredNot
	RequiredNot
)

func NewAffinityFilter() *AffinityFilter {
	return &AffinityFilter{
		rules: make([]AffinityRule, 0),
	}
}

func (f *AffinityFilter) AddRule(rule AffinityRule) {
	f.rules = append(f.rules, rule)
}

func (f *AffinityFilter) Filter(node *scheduler.NodeEntry, req *ScheduleRequest) bool {
	if len(f.rules) == 0 {
		return true
	}

	for _, rule := range f.rules {
		nodeValue, exists := node.Labels[rule.Label]
		if !exists {
			if rule.Preference == Required || rule.Preference == RequiredNot {
				return false
			}
			continue
		}

		matches := nodeValue == rule.Value
		if !matches && rule.Preference == Required {
			return false
		}
	}

	return true
}

func (f *AffinityFilter) Name() string {
	return "affinity"
}

type NodeAffinity struct {
	NodeSelector map[string]string
	Weight      int
	Preference  AffinityPreference
}

func ParseAffinity(affinityStr string) (*AffinityFilter, error) {
	f := NewAffinityFilter()

	rules := strings.Split(affinityStr, ",")
	for _, rule := range rules {
		rule = strings.TrimSpace(rule)
		if rule == "" {
			continue
		}

		var pref AffinityPreference
		if strings.HasPrefix(rule, "-") {
			pref = RequiredNot
			rule = rule[1:]
		} else if strings.HasPrefix(rule, "~") {
			pref = PreferredNot
			rule = rule[1:]
		} else if strings.HasPrefix(rule, "+") {
			pref = Preferred
			rule = rule[1:]
		} else {
			pref = Required
		}

		parts := strings.SplitN(rule, "=", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid affinity rule: %s", rule)
		}

		f.AddRule(AffinityRule{
			Label:      parts[0],
			Value:      parts[1],
			Preference: pref,
		})
	}

	return f, nil
}
