package filters

import (
	"fmt"
	"strings"

	"github.com/coco-sandbox/coco/pkg/scheduler"
)

type LabelFilter struct {
	requirements []LabelRequirement
}

type LabelRequirement struct {
	key      string
	operator LabelOperator
	values   []string
}

type LabelOperator int

const (
	LabelOpIn LabelOperator = iota
	LabelOpNotIn
	LabelOpExists
	LabelOpNotExists
	LabelOpEqual
	LabelOpNotEqual
)

func NewLabelFilter() *LabelFilter {
	return &LabelFilter{
		requirements: make([]LabelRequirement, 0),
	}
}

func (f *LabelFilter) AddRequirement(key string, operator LabelOperator, values ...string) {
	f.requirements = append(f.requirements, LabelRequirement{
		key:      key,
		operator: operator,
		values:   values,
	})
}

func (f *LabelFilter) Filter(node *scheduler.NodeEntry, req *ScheduleRequest) bool {
	if len(f.requirements) == 0 {
		return true
	}

	for _, req := range f.requirements {
		if !f.matchRequirement(node, req) {
			return false
		}
	}

	return true
}

func (f *LabelFilter) matchRequirement(node *scheduler.NodeEntry, req LabelRequirement) bool {
	nodeValue, exists := node.Labels[req.key]

	switch req.operator {
	case LabelOpIn:
		if !exists {
			return false
		}
		for _, v := range req.values {
			if nodeValue == v {
				return true
			}
		}
		return false

	case LabelOpNotIn:
		if !exists {
			return true
		}
		for _, v := range req.values {
			if nodeValue == v {
				return false
			}
		}
		return true

	case LabelOpExists:
		return exists

	case LabelOpNotExists:
		return !exists

	case LabelOpEqual:
		if !exists {
			return false
		}
		return nodeValue == req.values[0]

	case LabelOpNotEqual:
		if !exists {
			return true
		}
		return nodeValue != req.values[0]

	default:
		return true
	}
}

func (f *LabelFilter) Name() string {
	return "label"
}

func ParseLabelSelector(selector string) (*LabelFilter, error) {
	f := NewLabelFilter()

	if selector == "" {
		return f, nil
	}

	requirements := strings.Split(selector, ",")
	for _, req := range requirements {
		req = strings.TrimSpace(req)
		if req == "" {
			continue
		}

		var operator LabelOperator
		var key, value string

		if strings.Contains(req, "=") {
			parts := strings.SplitN(req, "=", 2)
			key = parts[0]
			value = parts[1]
			operator = LabelOpEqual
		} else if strings.Contains(req, "!=") {
			parts := strings.SplitN(req, "!=", 2)
			key = parts[0]
			value = parts[1]
			operator = LabelOpNotEqual
		} else if strings.HasSuffix(req, "*") {
			key = strings.TrimSuffix(req, "*")
			operator = LabelOpExists
		} else if strings.HasPrefix(req, "!") {
			key = strings.TrimPrefix(req, "!")
			operator = LabelOpNotExists
		} else {
			key = req
			operator = LabelOpExists
		}

		f.AddRequirement(key, operator, value)
	}

	return f, nil
}

func (f *LabelFilter) String() string {
	var parts []string
	for _, req := range f.requirements {
		switch req.operator {
		case LabelOpIn:
			parts = append(parts, fmt.Sprintf("%s in (%s)", req.key, strings.Join(req.values, ",")))
		case LabelOpNotIn:
			parts = append(parts, fmt.Sprintf("%s notin (%s)", req.key, strings.Join(req.values, ",")))
		case LabelOpExists:
			parts = append(parts, req.key)
		case LabelOpNotExists:
			parts = append(parts, "!"+req.key)
		case LabelOpEqual:
			parts = append(parts, fmt.Sprintf("%s=%s", req.key, req.values[0]))
		case LabelOpNotEqual:
			parts = append(parts, fmt.Sprintf("%s!=%s", req.key, req.values[0]))
		}
	}
	return strings.Join(parts, ",")
}
