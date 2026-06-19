package gtm

import (
	"encoding/json"
	"strconv"
	"time"
)

type ListAccountsOutput struct {
	Accounts []Account `json:"accounts"`
}

type ListContainersOutput struct {
	Containers []Container `json:"containers"`
}

type CreatedContainer struct {
	ContainerID   string   `json:"containerId"`
	Name          string   `json:"name"`
	PublicID      string   `json:"publicId"`
	UsageContext  []string `json:"usageContext"`
	Path          string   `json:"path"`
	TagManagerUrl string   `json:"tagManagerUrl,omitempty"`
}

type CreateContainerOutput struct {
	Success   bool             `json:"success"`
	Container CreatedContainer `json:"container"`
	Message   string           `json:"message"`
}

type DeleteContainerOutput struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

type ListWorkspacesOutput struct {
	Workspaces []Workspace `json:"workspaces"`
}

type CreatedWorkspace struct {
	WorkspaceID   string `json:"workspaceId"`
	Name          string `json:"name"`
	Description   string `json:"description,omitempty"`
	Path          string `json:"path"`
	TagManagerUrl string `json:"tagManagerUrl,omitempty"`
}

type CreateWorkspaceOutput struct {
	Success   bool             `json:"success"`
	Workspace CreatedWorkspace `json:"workspace"`
	Message   string           `json:"message"`
}

type DeleteWorkspaceOutput struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

type GetWorkspaceStatusOutput struct {
	Status WorkspaceStatus `json:"status"`
}

type ListTagsOutput struct {
	Tags []Tag `json:"tags"`
}

type GetTagOutput struct {
	Tag Tag `json:"tag"`
}

type CreateTagOutput struct {
	Success bool       `json:"success"`
	Tag     CreatedTag `json:"tag"`
	Message string     `json:"message"`
}

type UpdateTagOutput struct {
	Success bool       `json:"success"`
	Tag     CreatedTag `json:"tag"`
	Message string     `json:"message"`
}

type DeleteTagOutput struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

type ListTriggersOutput struct {
	Triggers []Trigger `json:"triggers"`
}

type GetTriggerOutput struct {
	Trigger Trigger `json:"trigger"`
}

type CreateTriggerOutput struct {
	Success bool           `json:"success"`
	Trigger CreatedTrigger `json:"trigger"`
	Message string         `json:"message"`
}

type UpdateTriggerOutput struct {
	Success bool           `json:"success"`
	Trigger CreatedTrigger `json:"trigger"`
	Message string         `json:"message"`
}

type DeleteTriggerOutput struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

type ListVariablesOutput struct {
	Variables []Variable `json:"variables"`
}

type GetVariableOutput struct {
	Variable Variable `json:"variable"`
}

type CreateVariableOutput struct {
	Success  bool            `json:"success"`
	Variable CreatedVariable `json:"variable"`
	Message  string          `json:"message"`
}

type UpdateVariableOutput struct {
	Success  bool            `json:"success"`
	Variable CreatedVariable `json:"variable"`
	Message  string          `json:"message"`
}

type DeleteVariableOutput struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

type ListClientsOutput struct {
	Clients []ClientInfo `json:"clients"`
}

type GetClientOutput struct {
	Client ClientInfo `json:"client"`
}

type CreateClientOutput struct {
	Success bool          `json:"success"`
	Client  CreatedClient `json:"client"`
	Message string        `json:"message"`
}

type UpdateClientOutput struct {
	Success bool          `json:"success"`
	Client  CreatedClient `json:"client"`
	Message string        `json:"message"`
}

type DeleteClientOutput struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

type ListTransformationsOutput struct {
	Transformations []TransformationInfo `json:"transformations"`
}

type GetTransformationOutput struct {
	Transformation TransformationInfo `json:"transformation"`
}

type CreateTransformationOutput struct {
	Success        bool                  `json:"success"`
	Transformation CreatedTransformation `json:"transformation"`
	Message        string                `json:"message"`
}

type UpdateTransformationOutput struct {
	Success        bool                  `json:"success"`
	Transformation CreatedTransformation `json:"transformation"`
	Message        string                `json:"message"`
}

type DeleteTransformationOutput struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

type ListEnvironmentsOutput struct {
	Environments []Environment `json:"environments"`
}

type GetEnvironmentOutput struct {
	Environment Environment `json:"environment"`
}

type CreateEnvironmentOutput struct {
	Success     bool        `json:"success"`
	Environment Environment `json:"environment"`
	Message     string      `json:"message"`
}

type UpdateEnvironmentOutput struct {
	Success     bool        `json:"success"`
	Environment Environment `json:"environment"`
	Message     string      `json:"message"`
}

type DeleteEnvironmentOutput struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

type ListUserPermissionsOutput struct {
	Permissions []UserPermission `json:"permissions"`
}

type GetUserPermissionOutput struct {
	Permission UserPermission `json:"permission"`
}

type CreateUserPermissionOutput struct {
	Success    bool           `json:"success"`
	Permission UserPermission `json:"permission"`
	Message    string         `json:"message"`
}

type UpdateUserPermissionOutput struct {
	Success    bool           `json:"success"`
	Permission UserPermission `json:"permission"`
	Message    string         `json:"message"`
}

type DeleteUserPermissionOutput struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

type ListFoldersOutput struct {
	Folders []Folder `json:"folders"`
}

type GetFolderEntitiesOutput struct {
	Entities FolderEntities `json:"entities"`
}

type CreateFolderOutput struct {
	Success bool   `json:"success"`
	Folder  Folder `json:"folder"`
	Message string `json:"message"`
}

type UpdateFolderOutput struct {
	Success bool   `json:"success"`
	Folder  Folder `json:"folder"`
	Message string `json:"message"`
}

type DeleteFolderOutput struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

type MoveToFolderOutput struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

type FolderAuditEntity struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Type     string `json:"type"`
	FolderID string `json:"folderId,omitempty"`
}

type AuditFolderStructureOutput struct {
	Folders          []Folder            `json:"folders"`
	UnorganizedTags  []FolderAuditEntity `json:"unorganizedTags"`
	UnorganizedTrigs []FolderAuditEntity `json:"unorganizedTriggers"`
	UnorganizedVars  []FolderAuditEntity `json:"unorganizedVariables"`
	Summary          string              `json:"summary"`
	NamingConvention string              `json:"namingConvention"`
}

type ListTemplatesOutput struct {
	Templates []TemplateInfo `json:"templates"`
}

type TemplateInfo struct {
	TemplateID       string                `json:"templateId"`
	Name             string                `json:"name"`
	Type             string                `json:"type"`
	GalleryReference *GalleryReferenceInfo `json:"galleryReference,omitempty"`
	TagManagerUrl    string                `json:"tagManagerUrl,omitempty"`
}

type GalleryReferenceInfo struct {
	Owner             string `json:"owner"`
	Repository        string `json:"repository"`
	Version           string `json:"version,omitempty"`
	GalleryTemplateId string `json:"galleryTemplateId,omitempty"`
}

type GetTemplateOutput struct {
	TemplateID       string                `json:"templateId"`
	Name             string                `json:"name"`
	Type             string                `json:"type"`
	TemplateData     string                `json:"templateData,omitempty"`
	GalleryReference *GalleryReferenceInfo `json:"galleryReference,omitempty"`
	Path             string                `json:"path"`
	Fingerprint      string                `json:"fingerprint"`
	TagManagerUrl    string                `json:"tagManagerUrl,omitempty"`
}

type CreateTemplateOutput struct {
	Success       bool   `json:"success"`
	TemplateID    string `json:"templateId"`
	Name          string `json:"name"`
	Type          string `json:"type"`
	Path          string `json:"path"`
	TagManagerUrl string `json:"tagManagerUrl,omitempty"`
	Message       string `json:"message"`
}

type UpdateTemplateOutput struct {
	Success       bool   `json:"success"`
	TemplateID    string `json:"templateId"`
	Name          string `json:"name"`
	Type          string `json:"type"`
	Path          string `json:"path"`
	Fingerprint   string `json:"fingerprint"`
	TagManagerUrl string `json:"tagManagerUrl,omitempty"`
	Message       string `json:"message"`
}

type DeleteTemplateOutput struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

type ImportGalleryTemplateOutput struct {
	Success  bool         `json:"success"`
	Template TemplateInfo `json:"template"`
	Message  string       `json:"message"`
}

type ListBuiltInVariablesOutput struct {
	BuiltInVariables []BuiltInVariable `json:"builtInVariables"`
}

type EnableBuiltInVariablesOutput struct {
	Success          bool              `json:"success"`
	BuiltInVariables []BuiltInVariable `json:"builtInVariables"`
	Message          string            `json:"message"`
}

type DisableBuiltInVariablesOutput struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

type RevertOutput struct {
	Success bool        `json:"success"`
	Message string      `json:"message"`
	Entity  interface{} `json:"entity,omitempty"`
}

type VersionInfo struct {
	VersionID          string `json:"versionId"`
	Name               string `json:"name"`
	Description        string `json:"description,omitempty"`
	NumTags            string `json:"numTags,omitempty"`
	NumTriggers        string `json:"numTriggers,omitempty"`
	NumVars            string `json:"numVariables,omitempty"`
	NumCustomTemplates string `json:"numCustomTemplates,omitempty"`
	Path               string `json:"path"`
	Deleted            bool   `json:"deleted,omitempty"`
}

type ListVersionsOutput struct {
	Versions []VersionInfo `json:"versions"`
}

type GetVersionOutput struct {
	Version ContainerVersionDetail `json:"version"`
}

type CreateVersionOutput struct {
	Success bool           `json:"success"`
	Version CreatedVersion `json:"version"`
	Message string         `json:"message"`
}

type PublishVersionOutput struct {
	Success bool             `json:"success"`
	Version PublishedVersion `json:"version"`
	Message string           `json:"message"`
}

type SetLatestVersionOutput struct {
	Success bool             `json:"success"`
	Version PublishedVersion `json:"version"`
	Message string           `json:"message"`
}

type EntityChange struct {
	Name   string `json:"name"`
	ID     string `json:"id"`
	Type   string `json:"type,omitempty"`
	Change string `json:"change"` // "added", "modified", "deleted"
}

type CompareVersionsOutput struct {
	VersionA        string         `json:"versionA"`
	VersionB        string         `json:"versionB"`
	TagChanges      []EntityChange `json:"tagChanges,omitempty"`
	TrigChanges     []EntityChange `json:"triggerChanges,omitempty"`
	VarChanges      []EntityChange `json:"variableChanges,omitempty"`
	TemplateChanges []EntityChange `json:"templateChanges,omitempty"`
	ClientChanges   []EntityChange `json:"clientChanges,omitempty"`
	TransChanges    []EntityChange `json:"transformationChanges,omitempty"`
	FolderChanges   []EntityChange `json:"folderChanges,omitempty"`
	Summary         string         `json:"summary"`
}

type VersionDateInfo struct {
	VersionID   string `json:"versionId"`
	Name        string `json:"name,omitempty"`
	Description string `json:"description,omitempty"`
	Fingerprint string `json:"fingerprint,omitempty"`
	Timestamp   string `json:"timestamp"` // Human-readable UTC timestamp
	Path        string `json:"path"`
}

type FindVersionByDateOutput struct {
	TargetDate string          `json:"targetDate"`
	Version    VersionDateInfo `json:"version"`
	Message    string          `json:"message"`
	APICalls   int             `json:"apiCalls"`
}

type ExportContainerOutput struct {
	ExportJSON json.RawMessage `json:"exportJson" jsonschema:"description:GTM container JSON export. This is the raw JSON object ready for import."`
	Format     string          `json:"format" jsonschema:"description:Export format used: ui (SCREAMING_CASE) or api (camelCase)"`
	Summary    string          `json:"summary"`
}

type GetTagTemplatesOutput struct {
	Templates []TagTemplate `json:"templates"`
	Usage     string        `json:"usage"`
}

type GetTriggerTemplatesOutput struct {
	Templates []TriggerTemplate `json:"templates"`
	Usage     string            `json:"usage"`
}

type namedEntity struct {
	Name string
	ID   string
	Type string
	Hash string
}

func tagMap(tags []Tag) map[string]namedEntity {
	m := make(map[string]namedEntity)
	for _, t := range tags {
		data, _ := json.Marshal(t.Parameter)
		m[t.Name] = namedEntity{Name: t.Name, ID: t.TagID, Type: t.Type, Hash: string(data)}
	}
	return m
}

func triggerMap(triggers []Trigger) map[string]namedEntity {
	m := make(map[string]namedEntity)
	for _, t := range triggers {
		data, _ := json.Marshal(t.Parameter)
		m[t.Name] = namedEntity{Name: t.Name, ID: t.TriggerID, Type: t.Type, Hash: string(data)}
	}
	return m
}

func variableMap(variables []Variable) map[string]namedEntity {
	m := make(map[string]namedEntity)
	for _, v := range variables {
		data, _ := json.Marshal(v.Parameter)
		m[v.Name] = namedEntity{Name: v.Name, ID: v.VariableID, Type: v.Type, Hash: string(data)}
	}
	return m
}

func diffEntities(a, b map[string]namedEntity) []EntityChange {
	var changes []EntityChange

	for name, entA := range a {
		if entB, ok := b[name]; !ok {
			changes = append(changes, EntityChange{Name: name, ID: entA.ID, Type: entA.Type, Change: "deleted"})
		} else if entA.Hash != entB.Hash || entA.Type != entB.Type {
			changes = append(changes, EntityChange{Name: name, ID: entB.ID, Type: entB.Type, Change: "modified"})
		}
	}

	for name, entB := range b {
		if _, ok := a[name]; !ok {
			changes = append(changes, EntityChange{Name: name, ID: entB.ID, Type: entB.Type, Change: "added"})
		}
	}

	return changes
}

func templateMap(templates []TemplateInfo) map[string]namedEntity {
	m := make(map[string]namedEntity)
	for _, t := range templates {
		m[t.Name] = namedEntity{Name: t.Name, ID: t.TemplateID, Type: t.Type}
	}
	return m
}

func clientMap(clients []ClientInfo) map[string]namedEntity {
	m := make(map[string]namedEntity)
	for _, c := range clients {
		data, _ := json.Marshal(c.Parameter)
		m[c.Name] = namedEntity{Name: c.Name, ID: c.ClientID, Type: c.Type, Hash: string(data)}
	}
	return m
}

func transformationMap(transformations []TransformationInfo) map[string]namedEntity {
	m := make(map[string]namedEntity)
	for _, t := range transformations {
		data, _ := json.Marshal(t.Parameter)
		m[t.Name] = namedEntity{Name: t.Name, ID: t.TransformationID, Type: t.Type, Hash: string(data)}
	}
	return m
}

func folderMap(folders []Folder) map[string]namedEntity {
	m := make(map[string]namedEntity)
	for _, f := range folders {
		m[f.Name] = namedEntity{Name: f.Name, ID: f.FolderID}
	}
	return m
}

func fingerprintToTime(fp string) (time.Time, error) {
	millis, err := strconv.ParseInt(fp, 10, 64)
	if err != nil {
		return time.Time{}, err
	}
	return time.UnixMilli(millis), nil
}
