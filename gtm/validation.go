package gtm

import (
	"fmt"
	"regexp"
	"strings"
)

var numericIDPattern = regexp.MustCompile(`^[0-9]+$`)

func ValidateNumericID(name, value string) error {
	if strings.TrimSpace(value) == "" {
		return fmt.Errorf("%s is required", name)
	}
	if !numericIDPattern.MatchString(value) {
		return fmt.Errorf("%s must be numeric (got %q)", name, value)
	}
	return nil
}

func ValidateTagInput(name, tagType string, firingTriggerIDs []string) error {
	if strings.TrimSpace(name) == "" {
		return fmt.Errorf("tag name is required")
	}
	if len(name) > 256 {
		return fmt.Errorf("tag name must be 256 characters or less")
	}
	if strings.TrimSpace(tagType) == "" {
		return fmt.Errorf("tag type is required")
	}
	if len(firingTriggerIDs) == 0 {
		return fmt.Errorf("at least one firing trigger ID is required")
	}
	for _, id := range firingTriggerIDs {
		if strings.TrimSpace(id) == "" {
			return fmt.Errorf("firing trigger ID cannot be empty")
		}
	}
	return nil
}

func ValidateTriggerInput(name, triggerType string) error {
	if strings.TrimSpace(name) == "" {
		return fmt.Errorf("trigger name is required")
	}
	if len(name) > 256 {
		return fmt.Errorf("trigger name must be 256 characters or less")
	}
	if strings.TrimSpace(triggerType) == "" {
		return fmt.Errorf("trigger type is required")
	}
	return nil
}

func ValidateVariableInput(name, varType string) error {
	if strings.TrimSpace(name) == "" {
		return fmt.Errorf("variable name is required")
	}
	if len(name) > 256 {
		return fmt.Errorf("variable name must be 256 characters or less")
	}
	if strings.TrimSpace(varType) == "" {
		return fmt.Errorf("variable type is required")
	}
	return nil
}

func ValidateWorkspacePath(accountID, containerID, workspaceID string) error {
	if err := ValidateNumericID("account ID", accountID); err != nil {
		return err
	}
	if err := ValidateNumericID("container ID", containerID); err != nil {
		return err
	}
	if err := ValidateNumericID("workspace ID", workspaceID); err != nil {
		return err
	}
	return nil
}

func BuildWorkspacePath(accountID, containerID, workspaceID string) string {
	return fmt.Sprintf("accounts/%s/containers/%s/workspaces/%s",
		accountID, containerID, workspaceID)
}

func ValidateContainerPath(accountID, containerID string) error {
	if err := ValidateNumericID("account ID", accountID); err != nil {
		return err
	}
	if err := ValidateNumericID("container ID", containerID); err != nil {
		return err
	}
	return nil
}

func BuildContainerPath(accountID, containerID string) string {
	return fmt.Sprintf("accounts/%s/containers/%s", accountID, containerID)
}

func ValidateClientInput(name, clientType string) error {
	if strings.TrimSpace(name) == "" {
		return fmt.Errorf("client name is required")
	}
	if len(name) > 256 {
		return fmt.Errorf("client name must be 256 characters or less")
	}
	if strings.TrimSpace(clientType) == "" {
		return fmt.Errorf("client type is required")
	}
	return nil
}

func ValidateTransformationInput(name, transformationType string) error {
	if strings.TrimSpace(name) == "" {
		return fmt.Errorf("transformation name is required")
	}
	if len(name) > 256 {
		return fmt.Errorf("transformation name must be 256 characters or less")
	}
	if strings.TrimSpace(transformationType) == "" {
		return fmt.Errorf("transformation type is required (valid values: tf_exclude_params, tf_allow_params, tf_augment_event)")
	}
	return nil
}
