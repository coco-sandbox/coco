package validators

import (
	"fmt"
)

type ExecValidator struct{}

func NewExecValidator() *ExecValidator {
	return &ExecValidator{}
}

func (v *ExecValidator) ValidateCreate(req interface {
	GetSandboxID() string
	GetCommand() []string
}) error {
	if req.GetSandboxID() == "" {
		return fmt.Errorf("sandbox_id is required")
	}

	if len(req.GetCommand()) == 0 {
		return fmt.Errorf("command is required")
	}

	return nil
}

func (v *ExecValidator) ValidateResize(req interface {
	GetSessionID() string
	GetWidth() uint32
	GetHeight() uint32
}) error {
	if req.GetSessionID() == "" {
		return fmt.Errorf("session_id is required")
	}

	if req.GetWidth() == 0 || req.GetHeight() == 0 {
		return fmt.Errorf("width and height must be greater than 0")
	}

	return nil
}

func (v *ExecValidator) ValidateInput(req interface {
	GetSessionID() string
	GetData() []byte
}) error {
	if req.GetSessionID() == "" {
		return fmt.Errorf("session_id is required")
	}

	if len(req.GetData()) == 0 {
		return fmt.Errorf("data is required")
	}

	return nil
}
