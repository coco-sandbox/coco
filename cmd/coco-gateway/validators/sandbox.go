package validators

import (
	"fmt"
	"regexp"
)

type SandboxValidator struct {
	idRegex *regexp.Regexp
}

func NewSandboxValidator() *SandboxValidator {
	return &SandboxValidator{
		idRegex: regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_.-]{0,62}[a-zA-Z0-9]$`),
	}
}

func (v *SandboxValidator) ValidateCreate(req interface {
	GetID() string
	GetImage() string
	GetSpec() interface{}
}) error {
	if req.GetID() != "" && !v.idRegex.MatchString(req.GetID()) {
		return fmt.Errorf("invalid sandbox id format")
	}

	if req.GetImage() == "" {
		return fmt.Errorf("image is required")
	}

	if req.GetSpec() == nil {
		return fmt.Errorf("spec is required")
	}

	return nil
}

func (v *SandboxValidator) ValidateStart(id string) error {
	if id == "" {
		return fmt.Errorf("sandbox id is required")
	}
	return nil
}

func (v *SandboxValidator) ValidateStop(id string) error {
	if id == "" {
		return fmt.Errorf("sandbox id is required")
	}
	return nil
}

func (v *SandboxValidator) ValidateGet(id string) error {
	if id == "" {
		return fmt.Errorf("sandbox id is required")
	}
	return nil
}

func (v *SandboxValidator) ValidateDelete(id string) error {
	if id == "" {
		return fmt.Errorf("sandbox id is required")
	}
	return nil
}
