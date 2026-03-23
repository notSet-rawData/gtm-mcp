package gtm

// boolPtr returns a pointer to the given bool value.
func boolPtr(b bool) *bool {
	return &b
}
// Parameter represents a GTM parameter structure.
// Used in tags, triggers, and variables.
type Parameter struct {
	Type  string      `json:"type"`            // "template", "boolean", "integer", "list", "map"
	Key   string      `json:"key"`
	Value string      `json:"value,omitempty"`
	List  []Parameter `json:"list,omitempty"`
	Map   []Parameter `json:"map,omitempty"`
}

// SetupTagInput represents a setup tag reference for tag sequencing.
type SetupTagInput struct {
	TagName            string `json:"tagName"`
	StopOnSetupFailure bool   `json:"stopOnSetupFailure,omitempty"`
}

// TeardownTagInput represents a teardown tag reference for tag sequencing.
type TeardownTagInput struct {
	TagName               string `json:"tagName"`
	StopTeardownOnFailure bool   `json:"stopTeardownOnFailure,omitempty"`
}

// TagInput represents input for creating/updating a tag.
type TagInput struct {
	Name               string             `json:"name"`
	Type               string             `json:"type"`
	FiringTriggerId    []string           `json:"firingTriggerId"`
	BlockingTriggerId  []string           `json:"blockingTriggerId,omitempty"`
	Parameter          []Parameter        `json:"parameter,omitempty"`
	Notes              string             `json:"notes,omitempty"`
	Paused             *bool              `json:"paused,omitempty"`
	TagFiringOption    string             `json:"tagFiringOption,omitempty"`
	SetupTag           []SetupTagInput    `json:"setupTag,omitempty"`
	TeardownTag        []TeardownTagInput `json:"teardownTag,omitempty"`
	ClearSetupTag      bool               `json:"-"` // When true, explicitly clear setup tags
	ClearTeardownTag   bool               `json:"-"` // When true, explicitly clear teardown tags
}

// TriggerInput represents input for creating/updating a trigger.
type TriggerInput struct {
	Name              string      `json:"name"`
	Type              string      `json:"type"`
	Filter            []Condition `json:"filter,omitempty"`
	AutoEventFilter   []Condition `json:"autoEventFilter,omitempty"`
	CustomEventFilter []Condition `json:"customEventFilter,omitempty"`
	EventName         *Parameter  `json:"eventName,omitempty"`
	Parameter         []Parameter `json:"parameter,omitempty"` // For trigger groups: member trigger references
	Notes             string      `json:"notes,omitempty"`
}

// Condition represents a filter condition for triggers.
type Condition struct {
	Type      string      `json:"type"` // "equals", "contains", "startsWith", etc.
	Negate    bool        `json:"negate,omitempty"`
	Parameter []Parameter `json:"parameter"`
}

// VariableInput represents input for creating a variable.
type VariableInput struct {
	Name           string      `json:"name"`
	Type           string      `json:"type"`
	Parameter      []Parameter `json:"parameter,omitempty"`
	Notes          string      `json:"notes,omitempty"`
	ParentFolderID string      `json:"parentFolderId,omitempty"`
}

// VersionInput represents input for creating a version.
type VersionInput struct {
	Name  string `json:"name,omitempty"`
	Notes string `json:"notes,omitempty"`
}

// CreatedTag represents the result of creating a tag.
type CreatedTag struct {
	TagID       string `json:"tagId"`
	Name        string `json:"name"`
	Type        string `json:"type"`
	Path        string `json:"path"`
	Fingerprint string `json:"fingerprint"`
}

// CreatedTrigger represents the result of creating a trigger.
type CreatedTrigger struct {
	TriggerID   string `json:"triggerId"`
	Name        string `json:"name"`
	Type        string `json:"type"`
	Path        string `json:"path"`
	Fingerprint string `json:"fingerprint"`
}

// CreatedVariable represents the result of creating a variable.
type CreatedVariable struct {
	VariableID  string `json:"variableId"`
	Name        string `json:"name"`
	Type        string `json:"type"`
	Path        string `json:"path"`
	Fingerprint string `json:"fingerprint"`
}

// CreatedVersion represents the result of creating a version.
type CreatedVersion struct {
	VersionID     string `json:"containerVersionId"`
	Name          string `json:"name"`
	Path          string `json:"path"`
	CompilerError bool   `json:"compilerError,omitempty"`
}

// BuiltInVariable represents an enabled built-in variable in a workspace.
type BuiltInVariable struct {
	Name string `json:"name"`
	Type string `json:"type"`
	Path string `json:"path"`
}

// ClientInfo represents a GTM client (server-side containers only).
type ClientInfo struct {
	ClientID       string `json:"clientId"`
	Name           string `json:"name"`
	Type           string `json:"type"`
	Priority       int64  `json:"priority,omitempty"`
	// Using any to avoid recursive type cycle in schema generation.
	Parameter      any    `json:"parameter,omitempty"`
	Notes          string `json:"notes,omitempty"`
	ParentFolderID string `json:"parentFolderId,omitempty"`
	Path           string `json:"path"`
	Fingerprint    string `json:"fingerprint"`
}

// ClientInput represents input for creating/updating a client.
type ClientInput struct {
	Name      string      `json:"name"`
	Type      string      `json:"type"`
	Priority  int64       `json:"priority,omitempty"`
	Parameter []Parameter `json:"parameter,omitempty"`
	Notes     string      `json:"notes,omitempty"`
}

// CreatedClient represents the result of creating a client.
type CreatedClient struct {
	ClientID    string `json:"clientId"`
	Name        string `json:"name"`
	Type        string `json:"type"`
	Path        string `json:"path"`
	Fingerprint string `json:"fingerprint"`
}

// TransformationInfo represents a GTM transformation (server-side containers only).
type TransformationInfo struct {
	TransformationID string `json:"transformationId"`
	Name             string `json:"name"`
	Type             string `json:"type"`
	// Using any to avoid recursive type cycle in schema generation.
	Parameter      any    `json:"parameter,omitempty"`
	Notes          string `json:"notes,omitempty"`
	ParentFolderID string `json:"parentFolderId,omitempty"`
	Path           string `json:"path"`
	Fingerprint    string `json:"fingerprint"`
}

// TransformationInput represents input for creating/updating a transformation.
type TransformationInput struct {
	Name      string      `json:"name"`
	Type      string      `json:"type"`
	Parameter []Parameter `json:"parameter,omitempty"`
	Notes     string      `json:"notes,omitempty"`
}

// CreatedTransformation represents the result of creating a transformation.
type CreatedTransformation struct {
	TransformationID string `json:"transformationId"`
	Name             string `json:"name"`
	Type             string `json:"type,omitempty"`
	Path             string `json:"path"`
	Fingerprint      string `json:"fingerprint"`
}

// EnvironmentInput represents input for creating/updating an environment.
type EnvironmentInput struct {
	Name               string `json:"name"`
	Description        string `json:"description,omitempty"`
	ContainerVersionID string `json:"containerVersionId,omitempty"`
	EnableDebug        bool   `json:"enableDebug,omitempty"`
}

// UserPermission represents a GTM user permission entry.
type UserPermission struct {
	PermissionID    string            `json:"permissionId"`
	EmailAddress    string            `json:"emailAddress"`
	AccountID       string            `json:"accountId"`
	AccountAccess   *AccountAccess    `json:"accountAccess,omitempty"`
	ContainerAccess []ContainerAccess `json:"containerAccess,omitempty"`
	Path            string            `json:"path"`
}

// AccountAccess represents the account-level permission.
type AccountAccess struct {
	Permission string `json:"permission"` // noAccess, read, edit, publish, admin
}

// ContainerAccess represents container-level permission for a user.
type ContainerAccess struct {
	ContainerID string `json:"containerId"`
	Permission  string `json:"permission"` // noAccess, read, edit, publish, approve
}

// UserPermissionInput represents input for creating/updating a user permission.
type UserPermissionInput struct {
	EmailAddress    string            `json:"emailAddress"`
	AccountAccess   *AccountAccess    `json:"accountAccess,omitempty"`
	ContainerAccess []ContainerAccess `json:"containerAccess,omitempty"`
}
