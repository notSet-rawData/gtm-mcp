package gtm

import (
	"context"
	"fmt"

	tagmanager "google.golang.org/api/tagmanager/v2"
)

func (c *Client) CreateTag(ctx context.Context, accountID, containerID, workspaceID string, input *TagInput) (*CreatedTag, error) {
	parent := BuildWorkspacePath(accountID, containerID, workspaceID)

	paused := false
	if input.Paused != nil {
		paused = *input.Paused
	}

	tag := &tagmanager.Tag{
		Name:                         input.Name,
		Type:                         input.Type,
		FiringTriggerId:              input.FiringTriggerId,
		BlockingTriggerId:            input.BlockingTriggerId,
		Parameter:                    toAPIParams(input.Parameter),
		Notes:                        input.Notes,
		Paused:                       paused,
		TagFiringOption:              input.TagFiringOption,
		SetupTag:                     toAPISetupTags(input.SetupTag),
		TeardownTag:                  toAPITeardownTags(input.TeardownTag),
		Priority:                     toAPIParam(input.Priority),
		ParentFolderId:               input.ParentFolderID,
		ScheduleStartMs:              input.ScheduleStartMs,
		ScheduleEndMs:                input.ScheduleEndMs,
		MonitoringMetadata:           toAPIParam(input.MonitoringMetadata),
		MonitoringMetadataTagNameKey: input.MonitoringMetadataTagNameKey,
		ConsentSettings:              toAPIConsentSettings(input.ConsentSettings),
	}

	result, err := c.Service.Accounts.Containers.Workspaces.Tags.Create(parent, tag).Context(ctx).Do()
	if err != nil {
		return nil, mapGoogleError(err)
	}

	return &CreatedTag{
		TagID:       result.TagId,
		Name:        result.Name,
		Type:        result.Type,
		Path:        result.Path,
		Fingerprint: result.Fingerprint,
	}, nil
}

func (c *Client) UpdateTag(ctx context.Context, path string, input *TagInput) (*CreatedTag, error) {
	current, err := c.Service.Accounts.Containers.Workspaces.Tags.Get(path).Context(ctx).Do()
	if err != nil {
		return nil, mapGoogleError(err)
	}

	params := toAPIParams(input.Parameter)
	if params == nil {
		params = current.Parameter
	}
	firingTriggerIds := input.FiringTriggerId
	if firingTriggerIds == nil {
		firingTriggerIds = current.FiringTriggerId
	}
	blockingTriggerIds := input.BlockingTriggerId
	if blockingTriggerIds == nil {
		blockingTriggerIds = current.BlockingTriggerId
	}
	notes := input.Notes
	if notes == "" {
		notes = current.Notes
	}
	tagFiringOption := input.TagFiringOption
	if tagFiringOption == "" {
		tagFiringOption = current.TagFiringOption
	}

	setupTags := toAPISetupTags(input.SetupTag)
	if setupTags == nil && !input.ClearSetupTag {
		setupTags = current.SetupTag
	}
	teardownTags := toAPITeardownTags(input.TeardownTag)
	if teardownTags == nil && !input.ClearTeardownTag {
		teardownTags = current.TeardownTag
	}

	paused := current.Paused
	if input.Paused != nil {
		paused = *input.Paused
	}

	priority := toAPIParam(input.Priority)
	if priority == nil {
		priority = current.Priority
	}

	parentFolderID := input.ParentFolderID
	if parentFolderID == "" {
		parentFolderID = current.ParentFolderId
	}

	scheduleStartMs := input.ScheduleStartMs
	if scheduleStartMs == 0 {
		scheduleStartMs = current.ScheduleStartMs
	}
	scheduleEndMs := input.ScheduleEndMs
	if scheduleEndMs == 0 {
		scheduleEndMs = current.ScheduleEndMs
	}

	monitoringMetadata := toAPIParam(input.MonitoringMetadata)
	if monitoringMetadata == nil {
		monitoringMetadata = current.MonitoringMetadata
	}
	monitoringMetadataTagNameKey := input.MonitoringMetadataTagNameKey
	if monitoringMetadataTagNameKey == "" {
		monitoringMetadataTagNameKey = current.MonitoringMetadataTagNameKey
	}

	consentSettings := toAPIConsentSettings(input.ConsentSettings)
	if consentSettings == nil {
		consentSettings = current.ConsentSettings
	}

	tag := &tagmanager.Tag{
		Name:                         input.Name,
		Type:                         input.Type,
		FiringTriggerId:              firingTriggerIds,
		BlockingTriggerId:            blockingTriggerIds,
		Parameter:                    params,
		Notes:                        notes,
		Paused:                       paused,
		TagFiringOption:              tagFiringOption,
		SetupTag:                     setupTags,
		TeardownTag:                  teardownTags,
		Fingerprint:                  current.Fingerprint,
		Priority:                     priority,
		ParentFolderId:               parentFolderID,
		ScheduleStartMs:              scheduleStartMs,
		ScheduleEndMs:                scheduleEndMs,
		MonitoringMetadata:           monitoringMetadata,
		MonitoringMetadataTagNameKey: monitoringMetadataTagNameKey,
		ConsentSettings:              consentSettings,
	}

	result, err := c.Service.Accounts.Containers.Workspaces.Tags.Update(path, tag).Context(ctx).Do()
	if err != nil {
		return nil, mapGoogleError(err)
	}

	return &CreatedTag{
		TagID:       result.TagId,
		Name:        result.Name,
		Type:        result.Type,
		Path:        result.Path,
		Fingerprint: result.Fingerprint,
	}, nil
}

func (c *Client) DeleteTag(ctx context.Context, path string) error {
	err := c.Service.Accounts.Containers.Workspaces.Tags.Delete(path).Context(ctx).Do()
	return mapGoogleError(err)
}

func (c *Client) CreateTrigger(ctx context.Context, accountID, containerID, workspaceID string, input *TriggerInput) (*CreatedTrigger, error) {
	parent := BuildWorkspacePath(accountID, containerID, workspaceID)

	trigger := &tagmanager.Trigger{
		Name:               input.Name,
		Type:               input.Type,
		Filter:             toAPIConditions(input.Filter),
		AutoEventFilter:    toAPIConditions(input.AutoEventFilter),
		CustomEventFilter:  toAPIConditions(input.CustomEventFilter),
		Parameter:          toAPIParams(input.Parameter),
		Notes:              input.Notes,
		ParentFolderId:     input.ParentFolderID,
		WaitForTags:        toAPIParam(input.WaitForTags),
		CheckValidation:    toAPIParam(input.CheckValidation),
		WaitForTagsTimeout: toAPIParam(input.WaitForTagsTimeout),
	}

	if input.EventName != nil {
		trigger.EventName = toAPIParam(input.EventName)
	}

	if len(input.AutoEventFilter) > 0 && (input.Type == "linkClick" || input.Type == "formSubmission" || input.Type == "click") {
		if trigger.WaitForTags == nil {
			trigger.WaitForTags = &tagmanager.Parameter{Type: "boolean", Value: "false"}
		}
		if trigger.WaitForTagsTimeout == nil {
			trigger.WaitForTagsTimeout = &tagmanager.Parameter{Type: "integer", Value: "2000"}
		}
		if trigger.CheckValidation == nil {
			trigger.CheckValidation = &tagmanager.Parameter{Type: "boolean", Value: "false"}
		}
	}

	result, err := c.Service.Accounts.Containers.Workspaces.Triggers.Create(parent, trigger).Context(ctx).Do()
	if err != nil {
		return nil, mapGoogleError(err)
	}

	return &CreatedTrigger{
		TriggerID:   result.TriggerId,
		Name:        result.Name,
		Type:        result.Type,
		Path:        result.Path,
		Fingerprint: result.Fingerprint,
	}, nil
}

func (c *Client) DeleteTrigger(ctx context.Context, path string) error {
	err := c.Service.Accounts.Containers.Workspaces.Triggers.Delete(path).Context(ctx).Do()
	return mapGoogleError(err)
}

func (c *Client) UpdateTrigger(ctx context.Context, path string, input *TriggerInput) (*CreatedTrigger, error) {
	current, err := c.Service.Accounts.Containers.Workspaces.Triggers.Get(path).Context(ctx).Do()
	if err != nil {
		return nil, mapGoogleError(err)
	}

	filter := toAPIConditions(input.Filter)
	if filter == nil {
		filter = current.Filter
	}
	autoEventFilter := toAPIConditions(input.AutoEventFilter)
	if autoEventFilter == nil {
		autoEventFilter = current.AutoEventFilter
	}
	customEventFilter := toAPIConditions(input.CustomEventFilter)
	if customEventFilter == nil {
		customEventFilter = current.CustomEventFilter
	}
	params := toAPIParams(input.Parameter)
	if params == nil {
		params = current.Parameter
	}

	notes := input.Notes
	if notes == "" {
		notes = current.Notes
	}

	parentFolderID := input.ParentFolderID
	if parentFolderID == "" {
		parentFolderID = current.ParentFolderId
	}

	waitForTags := toAPIParam(input.WaitForTags)
	if waitForTags == nil {
		waitForTags = current.WaitForTags
	}
	checkValidation := toAPIParam(input.CheckValidation)
	if checkValidation == nil {
		checkValidation = current.CheckValidation
	}
	waitForTagsTimeout := toAPIParam(input.WaitForTagsTimeout)
	if waitForTagsTimeout == nil {
		waitForTagsTimeout = current.WaitForTagsTimeout
	}

	trigger := &tagmanager.Trigger{
		Name:                           input.Name,
		Type:                           input.Type,
		Filter:                         filter,
		AutoEventFilter:                autoEventFilter,
		CustomEventFilter:              customEventFilter,
		Parameter:                      params,
		Notes:                          notes,
		ParentFolderId:                 parentFolderID,
		WaitForTags:                    waitForTags,
		CheckValidation:                checkValidation,
		WaitForTagsTimeout:             waitForTagsTimeout,
		ContinuousTimeMinMilliseconds:  current.ContinuousTimeMinMilliseconds,
		HorizontalScrollPercentageList: current.HorizontalScrollPercentageList,
		Interval:                       current.Interval,
		IntervalSeconds:                current.IntervalSeconds,
		Limit:                          current.Limit,
		MaxTimerLengthSeconds:          current.MaxTimerLengthSeconds,
		Selector:                       current.Selector,
		TotalTimeMinMilliseconds:       current.TotalTimeMinMilliseconds,
		VerticalScrollPercentageList:   current.VerticalScrollPercentageList,
		VisibilitySelector:             current.VisibilitySelector,
		VisiblePercentageMax:           current.VisiblePercentageMax,
		VisiblePercentageMin:           current.VisiblePercentageMin,
	}

	if input.EventName != nil {
		trigger.EventName = toAPIParam(input.EventName)
	} else {
		trigger.EventName = current.EventName
	}

	if len(autoEventFilter) > 0 && (input.Type == "linkClick" || input.Type == "formSubmission" || input.Type == "click") {
		if trigger.WaitForTags == nil || trigger.WaitForTags.Value == "" {
			trigger.WaitForTags = &tagmanager.Parameter{Type: "boolean", Value: "false"}
		}
		if trigger.WaitForTagsTimeout == nil || trigger.WaitForTagsTimeout.Value == "" {
			trigger.WaitForTagsTimeout = &tagmanager.Parameter{Type: "integer", Value: "2000"}
		}
		if trigger.CheckValidation == nil || trigger.CheckValidation.Value == "" {
			trigger.CheckValidation = &tagmanager.Parameter{Type: "boolean", Value: "false"}
		}
	}

	result, err := c.Service.Accounts.Containers.Workspaces.Triggers.Update(path, trigger).Fingerprint(current.Fingerprint).Context(ctx).Do()
	if err != nil {
		return nil, mapGoogleError(err)
	}

	return &CreatedTrigger{
		TriggerID:   result.TriggerId,
		Name:        result.Name,
		Type:        result.Type,
		Path:        result.Path,
		Fingerprint: result.Fingerprint,
	}, nil
}

func (c *Client) CreateVariable(ctx context.Context, accountID, containerID, workspaceID string, input *VariableInput) (*CreatedVariable, error) {
	parent := BuildWorkspacePath(accountID, containerID, workspaceID)

	variable := &tagmanager.Variable{
		Name:               input.Name,
		Type:               input.Type,
		Parameter:          toAPIParams(input.Parameter),
		Notes:              input.Notes,
		ParentFolderId:     input.ParentFolderID,
		ScheduleStartMs:    input.ScheduleStartMs,
		ScheduleEndMs:      input.ScheduleEndMs,
		EnablingTriggerId:  input.EnablingTriggerId,
		DisablingTriggerId: input.DisablingTriggerId,
		FormatValue:        toAPIFormatValue(input.FormatValue),
	}

	result, err := c.Service.Accounts.Containers.Workspaces.Variables.Create(parent, variable).Context(ctx).Do()
	if err != nil {
		return nil, mapGoogleError(err)
	}

	return &CreatedVariable{
		VariableID:  result.VariableId,
		Name:        result.Name,
		Type:        result.Type,
		Path:        result.Path,
		Fingerprint: result.Fingerprint,
	}, nil
}

func (c *Client) UpdateVariable(ctx context.Context, path string, input *VariableInput) (*CreatedVariable, error) {
	current, err := c.Service.Accounts.Containers.Workspaces.Variables.Get(path).Context(ctx).Do()
	if err != nil {
		return nil, mapGoogleError(err)
	}

	params := toAPIParams(input.Parameter)
	if params == nil {
		params = current.Parameter
	}
	notes := input.Notes
	if notes == "" {
		notes = current.Notes
	}
	parentFolderID := input.ParentFolderID
	if parentFolderID == "" {
		parentFolderID = current.ParentFolderId
	}
	scheduleStartMs := input.ScheduleStartMs
	if scheduleStartMs == 0 {
		scheduleStartMs = current.ScheduleStartMs
	}
	scheduleEndMs := input.ScheduleEndMs
	if scheduleEndMs == 0 {
		scheduleEndMs = current.ScheduleEndMs
	}
	enablingTriggerId := input.EnablingTriggerId
	if enablingTriggerId == nil {
		enablingTriggerId = current.EnablingTriggerId
	}
	disablingTriggerId := input.DisablingTriggerId
	if disablingTriggerId == nil {
		disablingTriggerId = current.DisablingTriggerId
	}
	formatValue := toAPIFormatValue(input.FormatValue)
	if formatValue == nil {
		formatValue = current.FormatValue
	}

	variable := &tagmanager.Variable{
		Name:               input.Name,
		Type:               input.Type,
		Parameter:          params,
		Notes:              notes,
		ParentFolderId:     parentFolderID,
		Fingerprint:        current.Fingerprint,
		ScheduleStartMs:    scheduleStartMs,
		ScheduleEndMs:      scheduleEndMs,
		EnablingTriggerId:  enablingTriggerId,
		DisablingTriggerId: disablingTriggerId,
		FormatValue:        formatValue,
	}

	result, err := c.Service.Accounts.Containers.Workspaces.Variables.Update(path, variable).Context(ctx).Do()
	if err != nil {
		return nil, mapGoogleError(err)
	}

	return &CreatedVariable{
		VariableID:  result.VariableId,
		Name:        result.Name,
		Type:        result.Type,
		Path:        result.Path,
		Fingerprint: result.Fingerprint,
	}, nil
}

func (c *Client) DeleteVariable(ctx context.Context, path string) error {
	err := c.Service.Accounts.Containers.Workspaces.Variables.Delete(path).Context(ctx).Do()
	return mapGoogleError(err)
}

func toAPIParams(params []Parameter) []*tagmanager.Parameter {
	if len(params) == 0 {
		return nil
	}
	result := make([]*tagmanager.Parameter, len(params))
	for i, p := range params {
		result[i] = toAPIParam(&p)
	}
	return result
}

func toAPIParam(p *Parameter) *tagmanager.Parameter {
	if p == nil {
		return nil
	}
	param := &tagmanager.Parameter{
		Type:            p.Type,
		Key:             p.Key,
		Value:           p.Value,
		ForceSendFields: []string{"Type", "Key", "Value"},
	}
	if len(p.List) > 0 {
		param.List = toAPIParams(p.List)
	}
	if len(p.Map) > 0 {
		param.Map = toAPIParams(p.Map)
	}
	return param
}

func toAPISetupTags(tags []SetupTagInput) []*tagmanager.SetupTag {
	if len(tags) == 0 {
		return nil
	}
	result := make([]*tagmanager.SetupTag, len(tags))
	for i, t := range tags {
		result[i] = &tagmanager.SetupTag{
			TagName:            t.TagName,
			StopOnSetupFailure: t.StopOnSetupFailure,
		}
	}
	return result
}

func toAPITeardownTags(tags []TeardownTagInput) []*tagmanager.TeardownTag {
	if len(tags) == 0 {
		return nil
	}
	result := make([]*tagmanager.TeardownTag, len(tags))
	for i, t := range tags {
		result[i] = &tagmanager.TeardownTag{
			TagName:               t.TagName,
			StopTeardownOnFailure: t.StopTeardownOnFailure,
		}
	}
	return result
}

func toAPIConditions(conditions []Condition) []*tagmanager.Condition {
	if len(conditions) == 0 {
		return nil
	}
	result := make([]*tagmanager.Condition, len(conditions))
	for i, c := range conditions {
		params := toAPIParams(c.Parameter)
		if c.Negate {
			params = append(params, &tagmanager.Parameter{
				Type:  "boolean",
				Key:   "negate",
				Value: "true",
			})
		}
		result[i] = &tagmanager.Condition{
			Type:            c.Type,
			Parameter:       params,
			ForceSendFields: []string{"Type", "Parameter"},
		}
	}
	return result
}

func triggerForceSendFields(input *TriggerInput) []string {
	var fields []string
	if len(input.Filter) > 0 {
		fields = append(fields, "Filter")
	}
	if len(input.AutoEventFilter) > 0 {
		fields = append(fields, "AutoEventFilter")
	}
	if len(input.CustomEventFilter) > 0 {
		fields = append(fields, "CustomEventFilter")
	}
	if len(input.Parameter) > 0 {
		fields = append(fields, "Parameter")
	}
	if input.EventName != nil {
		fields = append(fields, "EventName")
	}
	return fields
}

func toAPIConsentSettings(cs *ConsentSettingInput) *tagmanager.TagConsentSetting {
	if cs == nil {
		return nil
	}
	result := &tagmanager.TagConsentSetting{
		ConsentStatus: cs.ConsentStatus,
	}
	if cs.ConsentType != nil {
		result.ConsentType = toAPIParam(cs.ConsentType)
	}
	return result
}

func toAPIFormatValue(fv *FormatValueInput) *tagmanager.VariableFormatValue {
	if fv == nil {
		return nil
	}
	result := &tagmanager.VariableFormatValue{
		CaseConversionType:      fv.CaseConversionType,
		ConvertNullToValue:      toAPIParam(fv.ConvertNullToValue),
		ConvertUndefinedToValue: toAPIParam(fv.ConvertUndefinedToValue),
		ConvertTrueToValue:      toAPIParam(fv.ConvertTrueToValue),
		ConvertFalseToValue:     toAPIParam(fv.ConvertFalseToValue),
	}
	return result
}

func BuildTagPath(accountID, containerID, workspaceID, tagID string) string {
	return fmt.Sprintf("accounts/%s/containers/%s/workspaces/%s/tags/%s",
		accountID, containerID, workspaceID, tagID)
}

func BuildTriggerPath(accountID, containerID, workspaceID, triggerID string) string {
	return fmt.Sprintf("accounts/%s/containers/%s/workspaces/%s/triggers/%s",
		accountID, containerID, workspaceID, triggerID)
}

func BuildVariablePath(accountID, containerID, workspaceID, variableID string) string {
	return fmt.Sprintf("accounts/%s/containers/%s/workspaces/%s/variables/%s",
		accountID, containerID, workspaceID, variableID)
}

func BuildClientPath(accountID, containerID, workspaceID, clientID string) string {
	return fmt.Sprintf("accounts/%s/containers/%s/workspaces/%s/clients/%s",
		accountID, containerID, workspaceID, clientID)
}

func BuildTransformationPath(accountID, containerID, workspaceID, transformationID string) string {
	return fmt.Sprintf("accounts/%s/containers/%s/workspaces/%s/transformations/%s",
		accountID, containerID, workspaceID, transformationID)
}

func BuildEnvironmentPath(accountID, containerID, environmentID string) string {
	return fmt.Sprintf("accounts/%s/containers/%s/environments/%s",
		accountID, containerID, environmentID)
}

func BuildUserPermissionPath(accountID, permissionID string) string {
	return fmt.Sprintf("accounts/%s/permissions/%s", accountID, permissionID)
}
