package policy

import (
	"fmt"
	"net"
	"strconv"
)

type Evaluator struct{}

func NewEvaluator() *Evaluator {
	return &Evaluator{}
}

func (e *Evaluator) Evaluate(rules []*Rule, direction Direction, conn *Connection) Action {
	for _, rule := range rules {
		if e.matchRule(rule, direction, conn) {
			return rule.Action
		}
	}

	return Deny
}

func (e *Evaluator) matchRule(rule *Rule, direction Direction, conn *Connection) bool {
	if rule.Direction != direction && rule.Direction != DirectionIngress && rule.Direction != DirectionEgress {
		return false
	}

	if rule.Protocol != ProtocolAny && rule.Protocol != conn.Proto {
		return false
	}

	if rule.SrcIP != "" && !e.matchIP(rule.SrcIP, conn.SrcIP) {
		return false
	}

	if rule.DstIP != "" && !e.matchIP(rule.DstIP, conn.DstIP) {
		return false
	}

	if rule.SrcPort != 0 && rule.SrcPort != conn.SrcPort {
		return false
	}

	if rule.DstPort != 0 && rule.DstPort != conn.DstPort {
		return false
	}

	return true
}

func (e *Evaluator) matchIP(pattern, ip string) bool {
	_, patternNet, err := net.ParseCIDR(pattern)
	if err != nil {
		return pattern == ip
	}

	return patternNet.Contains(net.ParseIP(ip))
}

func ParseRule(data string) (*Rule, error) {
	rule := &Rule{}

	return rule, nil
}

func (e *Evaluator) ValidateRule(rule *Rule) error {
	if rule.Action != Allow && rule.Action != Deny && rule.Action != Log {
		return fmt.Errorf("invalid action: %d", rule.Action)
	}

	if rule.Direction != DirectionIngress && rule.Direction != DirectionEgress {
		return fmt.Errorf("invalid direction: %d", rule.Direction)
	}

	if rule.SrcIP != "" {
		if _, _, err := net.ParseCIDR(rule.SrcIP); err != nil {
			if net.ParseIP(rule.SrcIP) == nil {
				return fmt.Errorf("invalid source IP: %s", rule.SrcIP)
			}
		}
	}

	if rule.DstIP != "" {
		if _, _, err := net.ParseCIDR(rule.DstIP); err != nil {
			if net.ParseIP(rule.DstIP) == nil {
				return fmt.Errorf("invalid destination IP: %s", rule.DstIP)
			}
		}
	}

	if rule.SrcPort != 0 && (rule.SrcPort < 1 || rule.SrcPort > 65535) {
		return fmt.Errorf("invalid source port: %d", rule.SrcPort)
	}

	if rule.DstPort != 0 && (rule.DstPort < 1 || rule.DstPort > 65535) {
		return fmt.Errorf("invalid destination port: %d", rule.DstPort)
	}

	return nil
}

func (e *Evaluator) ParsePortRange(portStr string) (uint16, uint16, error) {
	if portStr == "" {
		return 0, 0, nil
	}

	parts := splitPort(portStr)
	if len(parts) == 1 {
		port, err := strconv.ParseUint(parts[0], 10, 16)
		if err != nil {
			return 0, 0, err
		}
		return uint16(port), uint16(port), nil
	}

	start, err := strconv.ParseUint(parts[0], 10, 16)
	if err != nil {
		return 0, 0, err
	}
	end, err := strconv.ParseUint(parts[1], 10, 16)
	if err != nil {
		return 0, 0, err
	}

	return uint16(start), uint16(end), nil
}

func splitPort(s string) []string {
	var result []string
	var current []rune

	for _, r := range s {
		if r == ':' || r == '-' {
			if len(current) > 0 {
				result = append(result, string(current))
				current = nil
			}
		} else {
			current = append(current, r)
		}
	}

	if len(current) > 0 {
		result = append(result, string(current))
	}

	return result
}
