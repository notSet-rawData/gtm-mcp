package gtm

func boolPtr(b bool) *bool {
	return &b
}

type Parameter struct {
	Type  string      `json:"type"` // "template", "boolean", "integer", "list", "map"
	Key   string      `json:"key"`
	Value string      `json:"value,omitempty"`
	List  []Parameter `json:"list,omitempty"`
	Map   []Parameter `json:"map,omitempty"`
}

type SetupTagInput struct {
	TagName            string `json:"tagName"`
	StopOnSetupFailure bool   `json:"stopOnSetupFailure,omitempty"`
}

type TeardownTagInput struct {
	TagName               string `json:"tagName"`
	StopTeardownOnFailure bool   `json:"stopTeardownOnFailure,omitempty"`
}

type ConsentSettingInput struct {
	ConsentStatus string     `json:"consentStatus,omitempty"` // notSet, notNeeded, needed
	ConsentType   *Parameter `json:"consentType,omitempty"`   // LIST type parameter with consent types
}

type FormatValueInput struct {
	CaseConversionType      string     `json:"caseConversionType,omitempty"` // none, lowercase, uppercase
	ConvertNullToValue      *Parameter `json:"convertNullToValue,omitempty"`
	ConvertUndefinedToValue *Parameter `json:"convertUndefinedToValue,omitempty"`
	ConvertTrueToValue      *Parameter `json:"convertTrueToValue,omitempty"`
	ConvertFalseToValue     *Parameter `json:"convertFalseToValue,omitempty"`
}

type TagInput struct {
	Name                         string               `json:"name"`
	Type                         string               `json:"type"`
	FiringTriggerId              []string             `json:"firingTriggerId"`
	BlockingTriggerId            []string             `json:"blockingTriggerId,omitempty"`
	Parameter                    []Parameter          `json:"parameter,omitempty"`
	Notes                        string               `json:"notes,omitempty"`
	Paused                       *bool                `json:"paused,omitempty"`
	TagFiringOption              string               `json:"tagFiringOption,omitempty"`
	SetupTag                     []SetupTagInput      `json:"setupTag,omitempty"`
	TeardownTag                  []TeardownTagInput   `json:"teardownTag,omitempty"`
	ClearSetupTag                bool                 `json:"-"` // When true, explicitly clear setup tags
	ClearTeardownTag             bool                 `json:"-"` // When true, explicitly clear teardown tags
	Priority                     *Parameter           `json:"priority,omitempty"`
	ParentFolderID               string               `json:"parentFolderId,omitempty"`
	ScheduleStartMs              int64                `json:"scheduleStartMs,omitempty"`
	ScheduleEndMs                int64                `json:"scheduleEndMs,omitempty"`
	MonitoringMetadata           *Parameter           `json:"monitoringMetadata,omitempty"`
	MonitoringMetadataTagNameKey string               `json:"monitoringMetadataTagNameKey,omitempty"`
	ConsentSettings              *ConsentSettingInput `json:"consentSettings,omitempty"`
}

type TriggerInput struct {
	Name               string      `json:"name"`
	Type               string      `json:"type"`
	Filter             []Condition `json:"filter,omitempty"`
	AutoEventFilter    []Condition `json:"autoEventFilter,omitempty"`
	CustomEventFilter  []Condition `json:"customEventFilter,omitempty"`
	EventName          *Parameter  `json:"eventName,omitempty"`
	Parameter          []Parameter `json:"parameter,omitempty"` // For trigger groups: member trigger references
	Notes              string      `json:"notes,omitempty"`
	WaitForTags        *Parameter  `json:"waitForTags,omitempty"`
	CheckValidation    *Parameter  `json:"checkValidation,omitempty"`
	WaitForTagsTimeout *Parameter  `json:"waitForTagsTimeout,omitempty"`
	ParentFolderID     string      `json:"parentFolderId,omitempty"`
}

type Condition struct {
	Type      string      `json:"type"` // "equals", "contains", "startsWith", etc.
	Negate    bool        `json:"negate,omitempty"`
	Parameter []Parameter `json:"parameter"`
}

type VariableInput struct {
	Name               string            `json:"name"`
	Type               string            `json:"type"`
	Parameter          []Parameter       `json:"parameter,omitempty"`
	Notes              string            `json:"notes,omitempty"`
	ParentFolderID     string            `json:"parentFolderId,omitempty"`
	ScheduleStartMs    int64             `json:"scheduleStartMs,omitempty"`
	ScheduleEndMs      int64             `json:"scheduleEndMs,omitempty"`
	EnablingTriggerId  []string          `json:"enablingTriggerId,omitempty"`
	DisablingTriggerId []string          `json:"disablingTriggerId,omitempty"`
	FormatValue        *FormatValueInput `json:"formatValue,omitempty"`
}

type VersionInput struct {
	Name  string `json:"name,omitempty"`
	Notes string `json:"notes,omitempty"`
}

type CreatedTag struct {
	TagID       string `json:"tagId"`
	Name        string `json:"name"`
	Type        string `json:"type"`
	Path        string `json:"path"`
	Fingerprint string `json:"fingerprint"`
}

type CreatedTrigger struct {
	TriggerID   string `json:"triggerId"`
	Name        string `json:"name"`
	Type        string `json:"type"`
	Path        string `json:"path"`
	Fingerprint string `json:"fingerprint"`
}

type CreatedVariable struct {
	VariableID  string `json:"variableId"`
	Name        string `json:"name"`
	Type        string `json:"type"`
	Path        string `json:"path"`
	Fingerprint string `json:"fingerprint"`
}

type CreatedVersion struct {
	VersionID     string `json:"containerVersionId"`
	Name          string `json:"name"`
	Path          string `json:"path"`
	CompilerError bool   `json:"compilerError,omitempty"`
}

type BuiltInVariable struct {
	Name string `json:"name"`
	Type string `json:"type"`
	Path string `json:"path"`
}

type ClientInfo struct {
	ClientID       string `json:"clientId"`
	Name           string `json:"name"`
	Type           string `json:"type"`
	Priority       int64  `json:"priority,omitempty"`
	Parameter      any    `json:"parameter,omitempty"`
	Notes          string `json:"notes,omitempty"`
	ParentFolderID string `json:"parentFolderId,omitempty"`
	Path           string `json:"path"`
	Fingerprint    string `json:"fingerprint"`
}

type ClientInput struct {
	Name      string      `json:"name"`
	Type      string      `json:"type"`
	Priority  int64       `json:"priority,omitempty"`
	Parameter []Parameter `json:"parameter,omitempty"`
	Notes     string      `json:"notes,omitempty"`
}

type CreatedClient struct {
	ClientID    string `json:"clientId"`
	Name        string `json:"name"`
	Type        string `json:"type"`
	Path        string `json:"path"`
	Fingerprint string `json:"fingerprint"`
}

type TransformationInfo struct {
	TransformationID string `json:"transformationId"`
	Name             string `json:"name"`
	Type             string `json:"type"`
	Parameter        any    `json:"parameter,omitempty"`
	Notes            string `json:"notes,omitempty"`
	ParentFolderID   string `json:"parentFolderId,omitempty"`
	Path             string `json:"path"`
	Fingerprint      string `json:"fingerprint"`
}

type TransformationInput struct {
	Name      string      `json:"name"`
	Type      string      `json:"type"`
	Parameter []Parameter `json:"parameter,omitempty"`
	Notes     string      `json:"notes,omitempty"`
}

type CreatedTransformation struct {
	TransformationID string `json:"transformationId"`
	Name             string `json:"name"`
	Type             string `json:"type,omitempty"`
	Path             string `json:"path"`
	Fingerprint      string `json:"fingerprint"`
}

type EnvironmentInput struct {
	Name               string `json:"name"`
	Description        string `json:"description,omitempty"`
	ContainerVersionID string `json:"containerVersionId,omitempty"`
	EnableDebug        bool   `json:"enableDebug,omitempty"`
}

type UserPermission struct {
	PermissionID    string            `json:"permissionId"`
	EmailAddress    string            `json:"emailAddress"`
	AccountID       string            `json:"accountId"`
	AccountAccess   *AccountAccess    `json:"accountAccess,omitempty"`
	ContainerAccess []ContainerAccess `json:"containerAccess,omitempty"`
	Path            string            `json:"path"`
}

type AccountAccess struct {
	Permission string `json:"permission"` // noAccess, read, edit, publish, admin
}

type ContainerAccess struct {
	ContainerID string `json:"containerId"`
	Permission  string `json:"permission"` // noAccess, read, edit, publish, approve
}

type UserPermissionInput struct {
	EmailAddress    string            `json:"emailAddress"`
	AccountAccess   *AccountAccess    `json:"accountAccess,omitempty"`
	ContainerAccess []ContainerAccess `json:"containerAccess,omitempty"`
}
