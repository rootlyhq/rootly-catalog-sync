package client

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/oapi-codegen/nullable"
	rootly "github.com/rootlyhq/rootly-go"

	"github.com/rootlyhq/rootly-catalog-sync/catalog"
)

// Native resource type constants.
const (
	NativeService       = "service"
	NativeFunctionality = "functionality"
	NativeEnvironment   = "environment"
	NativeTeam          = "team"
)

// Known attribute key constants (used in multiple places).
const (
	attrDescription        = "description"
	attrColor              = "color"
	attrBackstageID        = "backstage_id"
	attrCortexID           = "cortex_id"
	attrOpsgenieID         = "opsgenie_id"
	attrOpsgenieTeamID     = "opsgenie_team_id"
	attrOpslevelID         = "opslevel_id"
	attrPagerdutyID        = "pagerduty_id"
	attrPagerdutyServiceID = "pagerduty_service_id"
	attrPagertreeID        = "pagertree_id"
	attrVictorOpsID        = "victor_ops_id"
	attrServiceNowCiSysID  = "service_now_ci_sys_id"
	attrAlertsEmailEnabled = "alerts_email_enabled"
)

// Known writable attributes per native resource type. Fields matching these
// keys are set directly on the SDK struct; everything else goes into the
// catalog Fields array.
var nativeKnownAttrs = map[string]map[string]bool{
	NativeService: {
		attrDescription: true, attrColor: true, attrBackstageID: true, attrCortexID: true,
		attrOpsgenieID: true, attrOpsgenieTeamID: true, attrOpslevelID: true,
		attrPagerdutyID: true, attrServiceNowCiSysID: true,
		"github_repository_name": true, "github_repository_branch": true,
		"gitlab_repository_name": true, "gitlab_repository_branch": true,
		"kubernetes_deployment_name": true, attrAlertsEmailEnabled: true,
	},
	NativeFunctionality: {
		attrDescription: true, attrColor: true, attrBackstageID: true, attrCortexID: true,
		attrOpsgenieID: true, attrOpsgenieTeamID: true, attrOpslevelID: true,
		attrPagerdutyID: true, attrServiceNowCiSysID: true,
	},
	NativeEnvironment: {
		attrDescription: true, attrColor: true, "position": true,
	},
	NativeTeam: {
		attrDescription: true, attrColor: true, attrBackstageID: true, attrCortexID: true,
		attrOpsgenieID: true, attrOpslevelID: true, attrPagerdutyID: true,
		attrPagerdutyServiceID: true, attrPagertreeID: true, attrVictorOpsID: true,
		attrServiceNowCiSysID: true, attrAlertsEmailEnabled: true,
	},
}

// IsNativeResource returns true if the given type name is a built-in Rootly
// resource type that supports bulk sync (service, functionality, environment, team).
func IsNativeResource(t string) bool {
	switch t {
	case NativeService, NativeFunctionality, NativeEnvironment, NativeTeam:
		return true
	}
	return false
}

// resourceTypePlural returns the plural API name for a native resource type.
func resourceTypePlural(t string) string {
	switch t {
	case NativeService:
		return "services"
	case NativeFunctionality:
		return "functionalities"
	case NativeEnvironment:
		return "environments"
	case NativeTeam:
		return "teams"
	}
	return t + "s"
}

// ---------------------------------------------------------------------------
// List
// ---------------------------------------------------------------------------

// ListNativeResources lists all resources of the given native type, paginating
// through the API until all pages are fetched.
func (c *Client) ListNativeResources(ctx context.Context, resourceType string) ([]catalog.LiveEntity, error) {
	switch resourceType {
	case NativeService:
		return c.listServices(ctx)
	case NativeFunctionality:
		return c.listFunctionalities(ctx)
	case NativeEnvironment:
		return c.listEnvironments(ctx)
	case NativeTeam:
		return c.listTeams(ctx)
	default:
		return nil, fmt.Errorf("unsupported native resource type: %s", resourceType)
	}
}

func (c *Client) listServices(ctx context.Context) ([]catalog.LiveEntity, error) {
	var entities []catalog.LiveEntity
	page := 1
	for {
		pageSize := DefaultPageSize
		resp, err := c.sdk.ListServicesWithResponse(ctx, &rootly.ListServicesParams{
			PageNumber: &page,
			PageSize:   &pageSize,
		})
		if err != nil {
			return nil, fmt.Errorf("listing services: %w", err)
		}
		if resp.ApplicationVndAPIJSON200 == nil {
			return nil, fmt.Errorf("listing services: unexpected status %d", resp.StatusCode())
		}
		data := resp.ApplicationVndAPIJSON200
		for _, r := range data.Data {
			entities = append(entities, serviceToLive(r.ID, r.Attributes))
		}
		currentPage := 0
		if data.Meta.CurrentPage.IsSpecified() && !data.Meta.CurrentPage.IsNull() {
			currentPage = data.Meta.CurrentPage.MustGet()
		}
		if len(data.Data) == 0 || currentPage >= data.Meta.TotalPages {
			break
		}
		page++
	}
	return entities, nil
}

func serviceToLive(id string, s rootly.Service) catalog.LiveEntity {
	ent := catalog.LiveEntity{
		ID:     id,
		Name:   s.Name,
		Fields: make(map[string]string),
	}
	if s.ExternalID.IsSpecified() && !s.ExternalID.IsNull() {
		ent.ExternalID = s.ExternalID.MustGet()
	}
	if s.Description.IsSpecified() && !s.Description.IsNull() {
		ent.Description = s.Description.MustGet()
		ent.Fields[attrDescription] = ent.Description
	}
	// Service has no ManagedBy field in the SDK read model.

	setNullableStr(s.Color, attrColor, ent.Fields)
	setNullableStr(s.BackstageID, attrBackstageID, ent.Fields)
	setNullableStr(s.CortexID, attrCortexID, ent.Fields)
	setNullableStr(s.OpsgenieID, attrOpsgenieID, ent.Fields)
	setNullableStr(s.PagerdutyID, attrPagerdutyID, ent.Fields)
	setNullableStr(s.ServiceNowCiSysID, attrServiceNowCiSysID, ent.Fields)
	setNullableStr(s.GithubRepositoryName, "github_repository_name", ent.Fields)
	setNullableStr(s.GithubRepositoryBranch, "github_repository_branch", ent.Fields)
	setNullableStr(s.GitlabRepositoryName, "gitlab_repository_name", ent.Fields)
	setNullableStr(s.GitlabRepositoryBranch, "gitlab_repository_branch", ent.Fields)
	setNullableStr(s.KubernetesDeploymentName, "kubernetes_deployment_name", ent.Fields)
	setNullableBool(s.AlertsEmailEnabled, attrAlertsEmailEnabled, ent.Fields)

	return ent
}

func (c *Client) listFunctionalities(ctx context.Context) ([]catalog.LiveEntity, error) {
	var entities []catalog.LiveEntity
	page := 1
	for {
		pageSize := DefaultPageSize
		resp, err := c.sdk.ListFunctionalitiesWithResponse(ctx, &rootly.ListFunctionalitiesParams{
			PageNumber: &page,
			PageSize:   &pageSize,
		})
		if err != nil {
			return nil, fmt.Errorf("listing functionalities: %w", err)
		}
		if resp.ApplicationVndAPIJSON200 == nil {
			return nil, fmt.Errorf("listing functionalities: unexpected status %d", resp.StatusCode())
		}
		data := resp.ApplicationVndAPIJSON200
		for _, r := range data.Data {
			entities = append(entities, functionalityToLive(r.ID, r.Attributes))
		}
		currentPage := 0
		if data.Meta.CurrentPage.IsSpecified() && !data.Meta.CurrentPage.IsNull() {
			currentPage = data.Meta.CurrentPage.MustGet()
		}
		if len(data.Data) == 0 || currentPage >= data.Meta.TotalPages {
			break
		}
		page++
	}
	return entities, nil
}

func functionalityToLive(id string, f rootly.Functionality) catalog.LiveEntity {
	ent := catalog.LiveEntity{
		ID:     id,
		Name:   f.Name,
		Fields: make(map[string]string),
	}
	if f.ExternalID.IsSpecified() && !f.ExternalID.IsNull() {
		ent.ExternalID = f.ExternalID.MustGet()
	}
	if f.Description.IsSpecified() && !f.Description.IsNull() {
		ent.Description = f.Description.MustGet()
		ent.Fields[attrDescription] = ent.Description
	}
	if f.ManagedBy != nil {
		ent.ManagedBy = string(*f.ManagedBy)
	}

	setNullableStr(f.Color, attrColor, ent.Fields)
	setNullableStr(f.BackstageID, attrBackstageID, ent.Fields)
	setNullableStr(f.CortexID, attrCortexID, ent.Fields)
	setNullableStr(f.OpsgenieID, attrOpsgenieID, ent.Fields)
	setNullableStr(f.OpsgenieTeamID, attrOpsgenieTeamID, ent.Fields)
	setNullableStr(f.PagerdutyID, attrPagerdutyID, ent.Fields)
	setNullableStr(f.ServiceNowCiSysID, attrServiceNowCiSysID, ent.Fields)

	return ent
}

func (c *Client) listEnvironments(ctx context.Context) ([]catalog.LiveEntity, error) {
	var entities []catalog.LiveEntity
	page := 1
	for {
		pageSize := DefaultPageSize
		resp, err := c.sdk.ListEnvironmentsWithResponse(ctx, &rootly.ListEnvironmentsParams{
			PageNumber: &page,
			PageSize:   &pageSize,
		})
		if err != nil {
			return nil, fmt.Errorf("listing environments: %w", err)
		}
		if resp.ApplicationVndAPIJSON200 == nil {
			return nil, fmt.Errorf("listing environments: unexpected status %d", resp.StatusCode())
		}
		data := resp.ApplicationVndAPIJSON200
		for _, r := range data.Data {
			entities = append(entities, environmentToLive(r.ID, r.Attributes))
		}
		currentPage := 0
		if data.Meta.CurrentPage.IsSpecified() && !data.Meta.CurrentPage.IsNull() {
			currentPage = data.Meta.CurrentPage.MustGet()
		}
		if len(data.Data) == 0 || currentPage >= data.Meta.TotalPages {
			break
		}
		page++
	}
	return entities, nil
}

func environmentToLive(id string, e rootly.Environment) catalog.LiveEntity {
	ent := catalog.LiveEntity{
		ID:     id,
		Name:   e.Name,
		Fields: make(map[string]string),
	}
	if e.ExternalID.IsSpecified() && !e.ExternalID.IsNull() {
		ent.ExternalID = e.ExternalID.MustGet()
	}
	if e.Description.IsSpecified() && !e.Description.IsNull() {
		ent.Description = e.Description.MustGet()
		ent.Fields[attrDescription] = ent.Description
	}
	if e.ManagedBy != nil {
		ent.ManagedBy = string(*e.ManagedBy)
	}

	setNullableStr(e.Color, attrColor, ent.Fields)
	setNullableInt(e.Position, "position", ent.Fields)

	return ent
}

func (c *Client) listTeams(ctx context.Context) ([]catalog.LiveEntity, error) {
	var entities []catalog.LiveEntity
	page := 1
	for {
		pageSize := DefaultPageSize
		resp, err := c.sdk.ListTeamsWithResponse(ctx, &rootly.ListTeamsParams{
			PageNumber: &page,
			PageSize:   &pageSize,
		})
		if err != nil {
			return nil, fmt.Errorf("listing teams: %w", err)
		}
		if resp.ApplicationVndAPIJSON200 == nil {
			return nil, fmt.Errorf("listing teams: unexpected status %d", resp.StatusCode())
		}
		data := resp.ApplicationVndAPIJSON200
		for _, r := range data.Data {
			entities = append(entities, teamToLive(r.ID, r.Attributes))
		}
		currentPage := 0
		if data.Meta.CurrentPage.IsSpecified() && !data.Meta.CurrentPage.IsNull() {
			currentPage = data.Meta.CurrentPage.MustGet()
		}
		if len(data.Data) == 0 || currentPage >= data.Meta.TotalPages {
			break
		}
		page++
	}
	return entities, nil
}

func teamToLive(id string, t rootly.Team) catalog.LiveEntity {
	ent := catalog.LiveEntity{
		ID:     id,
		Name:   t.Name,
		Fields: make(map[string]string),
	}
	if t.ExternalID.IsSpecified() && !t.ExternalID.IsNull() {
		ent.ExternalID = t.ExternalID.MustGet()
	}
	if t.Description.IsSpecified() && !t.Description.IsNull() {
		ent.Description = t.Description.MustGet()
		ent.Fields[attrDescription] = ent.Description
	}
	if t.ManagedBy != nil {
		ent.ManagedBy = string(*t.ManagedBy)
	}

	setNullableStr(t.Color, attrColor, ent.Fields)
	setNullableStr(t.BackstageID, attrBackstageID, ent.Fields)
	setNullableStr(t.CortexID, attrCortexID, ent.Fields)
	setNullableStr(t.OpsgenieID, attrOpsgenieID, ent.Fields)
	setNullableStr(t.PagerdutyID, attrPagerdutyID, ent.Fields)
	setNullableStr(t.PagerdutyServiceID, attrPagerdutyServiceID, ent.Fields)
	setNullableStr(t.PagertreeID, attrPagertreeID, ent.Fields)
	setNullableStr(t.VictorOpsID, attrVictorOpsID, ent.Fields)
	setNullableStr(t.ServiceNowCiSysID, attrServiceNowCiSysID, ent.Fields)
	setNullableBool(t.AlertsEmailEnabled, attrAlertsEmailEnabled, ent.Fields)

	return ent
}

// ---------------------------------------------------------------------------
// Nullable helpers
// ---------------------------------------------------------------------------

func setNullableStr(n nullable.Nullable[string], key string, m map[string]string) {
	if n.IsSpecified() && !n.IsNull() {
		if v := n.MustGet(); v != "" {
			m[key] = v
		}
	}
}

func setNullableBool(n nullable.Nullable[bool], key string, m map[string]string) {
	if n.IsSpecified() && !n.IsNull() {
		m[key] = strconv.FormatBool(n.MustGet())
	}
}

func setNullableInt(n nullable.Nullable[int], key string, m map[string]string) {
	if n.IsSpecified() && !n.IsNull() {
		m[key] = strconv.Itoa(n.MustGet())
	}
}

// ---------------------------------------------------------------------------
// Bulk Upsert
// ---------------------------------------------------------------------------

// BulkUpsertNative upserts native resources in batches. For each entity, fields
// matching known attributes are set directly on the SDK struct; remaining fields
// go into the catalog Fields array.
func (c *Client) BulkUpsertNative(ctx context.Context, resourceType string, ents []catalog.DesiredEntity) (*BulkResult, error) {
	if !IsNativeResource(resourceType) {
		return nil, fmt.Errorf("unsupported native resource type: %s", resourceType)
	}

	if err := validateNativeFields(resourceType, ents); err != nil {
		return nil, err
	}

	result := &BulkResult{}
	for i := 0; i < len(ents); i += MaxBatchSize {
		end := i + MaxBatchSize
		if end > len(ents) {
			end = len(ents)
		}
		batch := ents[i:end]

		body, err := c.bulkUpsertNativeBatch(ctx, resourceType, batch)
		if err != nil {
			return nil, err
		}
		result.Succeeded += body.Succeeded
		result.Errors = append(result.Errors, body.Errors...)
	}
	return result, nil
}

func (c *Client) bulkUpsertNativeBatch(ctx context.Context, resourceType string, batch []catalog.DesiredEntity) (*BulkResult, error) {
	known := nativeKnownAttrs[resourceType]

	switch resourceType {
	case NativeService:
		return c.bulkUpsertServices(ctx, batch, known)
	case NativeFunctionality:
		return c.bulkUpsertFunctionalities(ctx, batch, known)
	case NativeEnvironment:
		return c.bulkUpsertEnvironments(ctx, batch, known)
	case NativeTeam:
		return c.bulkUpsertTeams(ctx, batch, known)
	default:
		return nil, fmt.Errorf("unsupported native resource type: %s", resourceType)
	}
}

func validateNativeFields(resourceType string, ents []catalog.DesiredEntity) error {
	known := nativeKnownAttrs[resourceType]
	unsupported := make(map[string]bool)
	for _, e := range ents {
		for k := range e.Fields {
			if !known[k] {
				unsupported[k] = true
			}
		}
	}
	if len(unsupported) == 0 {
		return nil
	}
	names := make([]string, 0, len(unsupported))
	for k := range unsupported {
		names = append(names, k)
	}
	sort.Strings(names)
	supported := make([]string, 0, len(known))
	for k := range known {
		supported = append(supported, k)
	}
	sort.Strings(supported)
	return fmt.Errorf("unsupported fields for native %s: %s\n  Supported fields: %s\n  Hint: for custom fields, use catalog entities (catalog: \"Name\") instead of native resources (type: %s)",
		resourceType, strings.Join(names, ", "), strings.Join(supported, ", "), resourceType)
}

// catalogFields builds the Fields array for any fields not in the known attrs set.
func catalogFields(ent catalog.DesiredEntity, known map[string]bool) []struct {
	CatalogFieldID    *string `json:"catalog_field_id,omitempty"`
	CatalogPropertyID *string `json:"catalog_property_id,omitempty"`
	Value             string  `json:"value"`
} {
	var fields []struct {
		CatalogFieldID    *string `json:"catalog_field_id,omitempty"`
		CatalogPropertyID *string `json:"catalog_property_id,omitempty"`
		Value             string  `json:"value"`
	}
	for slug, val := range ent.Fields {
		if known[slug] {
			continue
		}
		s := slug
		fields = append(fields, struct {
			CatalogFieldID    *string `json:"catalog_field_id,omitempty"`
			CatalogPropertyID *string `json:"catalog_property_id,omitempty"`
			Value             string  `json:"value"`
		}{
			CatalogFieldID: &s,
			Value:          val,
		})
	}
	return fields
}

func (c *Client) bulkUpsertServices(ctx context.Context, batch []catalog.DesiredEntity, known map[string]bool) (*BulkResult, error) {
	sdkEntities := make([]struct {
		AlertsEmailEnabled nullable.Nullable[bool]   `json:"alerts_email_enabled,omitempty"`
		BackstageID        nullable.Nullable[string] `json:"backstage_id,omitempty"`
		Color              nullable.Nullable[string] `json:"color,omitempty"`
		CortexID           nullable.Nullable[string] `json:"cortex_id,omitempty"`
		Description        nullable.Nullable[string] `json:"description,omitempty"`
		ExternalID         string                    `json:"external_id"`
		Fields             []struct {
			CatalogFieldID    *string `json:"catalog_field_id,omitempty"`
			CatalogPropertyID *string `json:"catalog_property_id,omitempty"`
			Value             string  `json:"value"`
		} `json:"fields,omitempty"`
		GithubRepositoryBranch   nullable.Nullable[string]   `json:"github_repository_branch,omitempty"`
		GithubRepositoryName     nullable.Nullable[string]   `json:"github_repository_name,omitempty"`
		GitlabRepositoryBranch   nullable.Nullable[string]   `json:"gitlab_repository_branch,omitempty"`
		GitlabRepositoryName     nullable.Nullable[string]   `json:"gitlab_repository_name,omitempty"`
		KubernetesDeploymentName nullable.Nullable[string]   `json:"kubernetes_deployment_name,omitempty"`
		Name                     *string                     `json:"name,omitempty"`
		NotifyEmails             nullable.Nullable[[]string] `json:"notify_emails,omitempty"`
		OpsgenieID               nullable.Nullable[string]   `json:"opsgenie_id,omitempty"`
		OpsgenieTeamID           nullable.Nullable[string]   `json:"opsgenie_team_id,omitempty"`
		OpslevelID               nullable.Nullable[string]   `json:"opslevel_id,omitempty"`
		PagerdutyID              nullable.Nullable[string]   `json:"pagerduty_id,omitempty"`
		Position                 nullable.Nullable[int]      `json:"position,omitempty"`
		PublicDescription        nullable.Nullable[string]   `json:"public_description,omitempty"`
		ServiceNowCiSysID        nullable.Nullable[string]   `json:"service_now_ci_sys_id,omitempty"`
		ShowUptime               nullable.Nullable[bool]     `json:"show_uptime,omitempty"`
		ShowUptimeLastDays       nullable.Nullable[int]      `json:"show_uptime_last_days,omitempty"`
	}, len(batch))

	for j, e := range batch {
		sdkEntities[j].ExternalID = e.ExternalID
		sdkEntities[j].Name = &e.Name
		if e.BackstageID != "" {
			sdkEntities[j].BackstageID = nullable.NewNullableWithValue(e.BackstageID)
		}
		setServiceAttrs(&sdkEntities[j], e.Fields)
		sdkEntities[j].Fields = catalogFields(e, known)
	}

	body := rootly.BulkUpsertServices{Entities: sdkEntities}
	resp, err := c.sdk.BulkUpsertServicesWithApplicationVndAPIPlusJSONBodyWithResponse(ctx, body)
	if err != nil {
		return nil, fmt.Errorf("bulk upsert %s: %w", resourceTypePlural("service"), err)
	}
	return parseBulkUpsertResponse(resp.StatusCode(), resp.Body)
}

func setServiceAttrs(ent *struct {
	AlertsEmailEnabled nullable.Nullable[bool]   `json:"alerts_email_enabled,omitempty"`
	BackstageID        nullable.Nullable[string] `json:"backstage_id,omitempty"`
	Color              nullable.Nullable[string] `json:"color,omitempty"`
	CortexID           nullable.Nullable[string] `json:"cortex_id,omitempty"`
	Description        nullable.Nullable[string] `json:"description,omitempty"`
	ExternalID         string                    `json:"external_id"`
	Fields             []struct {
		CatalogFieldID    *string `json:"catalog_field_id,omitempty"`
		CatalogPropertyID *string `json:"catalog_property_id,omitempty"`
		Value             string  `json:"value"`
	} `json:"fields,omitempty"`
	GithubRepositoryBranch   nullable.Nullable[string]   `json:"github_repository_branch,omitempty"`
	GithubRepositoryName     nullable.Nullable[string]   `json:"github_repository_name,omitempty"`
	GitlabRepositoryBranch   nullable.Nullable[string]   `json:"gitlab_repository_branch,omitempty"`
	GitlabRepositoryName     nullable.Nullable[string]   `json:"gitlab_repository_name,omitempty"`
	KubernetesDeploymentName nullable.Nullable[string]   `json:"kubernetes_deployment_name,omitempty"`
	Name                     *string                     `json:"name,omitempty"`
	NotifyEmails             nullable.Nullable[[]string] `json:"notify_emails,omitempty"`
	OpsgenieID               nullable.Nullable[string]   `json:"opsgenie_id,omitempty"`
	OpsgenieTeamID           nullable.Nullable[string]   `json:"opsgenie_team_id,omitempty"`
	OpslevelID               nullable.Nullable[string]   `json:"opslevel_id,omitempty"`
	PagerdutyID              nullable.Nullable[string]   `json:"pagerduty_id,omitempty"`
	Position                 nullable.Nullable[int]      `json:"position,omitempty"`
	PublicDescription        nullable.Nullable[string]   `json:"public_description,omitempty"`
	ServiceNowCiSysID        nullable.Nullable[string]   `json:"service_now_ci_sys_id,omitempty"`
	ShowUptime               nullable.Nullable[bool]     `json:"show_uptime,omitempty"`
	ShowUptimeLastDays       nullable.Nullable[int]      `json:"show_uptime_last_days,omitempty"`
}, fields map[string]string) {
	for k, v := range fields {
		switch k {
		case attrDescription:
			ent.Description = nullable.NewNullableWithValue(v)
		case attrColor:
			ent.Color = nullable.NewNullableWithValue(v)
		case attrBackstageID:
			ent.BackstageID = nullable.NewNullableWithValue(v)
		case attrCortexID:
			ent.CortexID = nullable.NewNullableWithValue(v)
		case attrOpsgenieID:
			ent.OpsgenieID = nullable.NewNullableWithValue(v)
		case attrOpsgenieTeamID:
			ent.OpsgenieTeamID = nullable.NewNullableWithValue(v)
		case attrOpslevelID:
			ent.OpslevelID = nullable.NewNullableWithValue(v)
		case attrPagerdutyID:
			ent.PagerdutyID = nullable.NewNullableWithValue(v)
		case attrServiceNowCiSysID:
			ent.ServiceNowCiSysID = nullable.NewNullableWithValue(v)
		case "github_repository_name":
			ent.GithubRepositoryName = nullable.NewNullableWithValue(v)
		case "github_repository_branch":
			ent.GithubRepositoryBranch = nullable.NewNullableWithValue(v)
		case "gitlab_repository_name":
			ent.GitlabRepositoryName = nullable.NewNullableWithValue(v)
		case "gitlab_repository_branch":
			ent.GitlabRepositoryBranch = nullable.NewNullableWithValue(v)
		case "kubernetes_deployment_name":
			ent.KubernetesDeploymentName = nullable.NewNullableWithValue(v)
		case attrAlertsEmailEnabled:
			if b, err := strconv.ParseBool(v); err == nil {
				ent.AlertsEmailEnabled = nullable.NewNullableWithValue(b)
			}
		}
	}
}

func (c *Client) bulkUpsertFunctionalities(ctx context.Context, batch []catalog.DesiredEntity, known map[string]bool) (*BulkResult, error) {
	sdkEntities := make([]struct {
		BackstageID nullable.Nullable[string] `json:"backstage_id,omitempty"`
		Color       nullable.Nullable[string] `json:"color,omitempty"`
		CortexID    nullable.Nullable[string] `json:"cortex_id,omitempty"`
		Description nullable.Nullable[string] `json:"description,omitempty"`
		ExternalID  string                    `json:"external_id"`
		Fields      []struct {
			CatalogFieldID    *string `json:"catalog_field_id,omitempty"`
			CatalogPropertyID *string `json:"catalog_property_id,omitempty"`
			Value             string  `json:"value"`
		} `json:"fields,omitempty"`
		Name               *string                     `json:"name,omitempty"`
		NotifyEmails       nullable.Nullable[[]string] `json:"notify_emails,omitempty"`
		OpsgenieID         nullable.Nullable[string]   `json:"opsgenie_id,omitempty"`
		OpsgenieTeamID     nullable.Nullable[string]   `json:"opsgenie_team_id,omitempty"`
		OpslevelID         nullable.Nullable[string]   `json:"opslevel_id,omitempty"`
		PagerdutyID        nullable.Nullable[string]   `json:"pagerduty_id,omitempty"`
		Position           nullable.Nullable[int]      `json:"position,omitempty"`
		PublicDescription  nullable.Nullable[string]   `json:"public_description,omitempty"`
		ServiceNowCiSysID  nullable.Nullable[string]   `json:"service_now_ci_sys_id,omitempty"`
		ShowUptime         nullable.Nullable[bool]     `json:"show_uptime,omitempty"`
		ShowUptimeLastDays nullable.Nullable[int]      `json:"show_uptime_last_days,omitempty"`
	}, len(batch))

	for j, e := range batch {
		sdkEntities[j].ExternalID = e.ExternalID
		sdkEntities[j].Name = &e.Name
		if e.BackstageID != "" {
			sdkEntities[j].BackstageID = nullable.NewNullableWithValue(e.BackstageID)
		}
		for k, v := range e.Fields {
			switch k {
			case attrDescription:
				sdkEntities[j].Description = nullable.NewNullableWithValue(v)
			case attrColor:
				sdkEntities[j].Color = nullable.NewNullableWithValue(v)
			case attrBackstageID:
				sdkEntities[j].BackstageID = nullable.NewNullableWithValue(v)
			case attrCortexID:
				sdkEntities[j].CortexID = nullable.NewNullableWithValue(v)
			case attrOpsgenieID:
				sdkEntities[j].OpsgenieID = nullable.NewNullableWithValue(v)
			case attrOpsgenieTeamID:
				sdkEntities[j].OpsgenieTeamID = nullable.NewNullableWithValue(v)
			case attrOpslevelID:
				sdkEntities[j].OpslevelID = nullable.NewNullableWithValue(v)
			case attrPagerdutyID:
				sdkEntities[j].PagerdutyID = nullable.NewNullableWithValue(v)
			case attrServiceNowCiSysID:
				sdkEntities[j].ServiceNowCiSysID = nullable.NewNullableWithValue(v)
			}
		}
		sdkEntities[j].Fields = catalogFields(e, known)
	}

	body := rootly.BulkUpsertFunctionalities{Entities: sdkEntities}
	resp, err := c.sdk.BulkUpsertFunctionalitiesWithApplicationVndAPIPlusJSONBodyWithResponse(ctx, body)
	if err != nil {
		return nil, fmt.Errorf("bulk upsert %s: %w", resourceTypePlural("functionality"), err)
	}
	return parseBulkUpsertResponse(resp.StatusCode(), resp.Body)
}

func (c *Client) bulkUpsertEnvironments(ctx context.Context, batch []catalog.DesiredEntity, known map[string]bool) (*BulkResult, error) {
	sdkEntities := make([]struct {
		Color       nullable.Nullable[string] `json:"color,omitempty"`
		Description nullable.Nullable[string] `json:"description,omitempty"`
		ExternalID  string                    `json:"external_id"`
		Fields      []struct {
			CatalogFieldID    *string `json:"catalog_field_id,omitempty"`
			CatalogPropertyID *string `json:"catalog_property_id,omitempty"`
			Value             string  `json:"value"`
		} `json:"fields,omitempty"`
		Name         *string                     `json:"name,omitempty"`
		NotifyEmails nullable.Nullable[[]string] `json:"notify_emails,omitempty"`
		Position     nullable.Nullable[int]      `json:"position,omitempty"`
	}, len(batch))

	for j, e := range batch {
		sdkEntities[j].ExternalID = e.ExternalID
		sdkEntities[j].Name = &e.Name
		for k, v := range e.Fields {
			switch k {
			case attrDescription:
				sdkEntities[j].Description = nullable.NewNullableWithValue(v)
			case attrColor:
				sdkEntities[j].Color = nullable.NewNullableWithValue(v)
			case "position":
				if pos, err := strconv.Atoi(v); err == nil {
					sdkEntities[j].Position = nullable.NewNullableWithValue(pos)
				}
			}
		}
		sdkEntities[j].Fields = catalogFields(e, known)
	}

	body := rootly.BulkUpsertEnvironments{Entities: sdkEntities}
	resp, err := c.sdk.BulkUpsertEnvironmentsWithApplicationVndAPIPlusJSONBodyWithResponse(ctx, body)
	if err != nil {
		return nil, fmt.Errorf("bulk upsert %s: %w", resourceTypePlural("environment"), err)
	}
	return parseBulkUpsertResponse(resp.StatusCode(), resp.Body)
}

func (c *Client) bulkUpsertTeams(ctx context.Context, batch []catalog.DesiredEntity, known map[string]bool) (*BulkResult, error) {
	sdkEntities := make([]struct {
		AlertsEmailEnabled nullable.Nullable[bool]   `json:"alerts_email_enabled,omitempty"`
		BackstageID        nullable.Nullable[string] `json:"backstage_id,omitempty"`
		Color              nullable.Nullable[string] `json:"color,omitempty"`
		CortexID           nullable.Nullable[string] `json:"cortex_id,omitempty"`
		Description        nullable.Nullable[string] `json:"description,omitempty"`
		ExternalID         string                    `json:"external_id"`
		Fields             []struct {
			CatalogFieldID    *string `json:"catalog_field_id,omitempty"`
			CatalogPropertyID *string `json:"catalog_property_id,omitempty"`
			Value             string  `json:"value"`
		} `json:"fields,omitempty"`
		Name               *string                     `json:"name,omitempty"`
		NotifyEmails       nullable.Nullable[[]string] `json:"notify_emails,omitempty"`
		OpsgenieID         nullable.Nullable[string]   `json:"opsgenie_id,omitempty"`
		OpslevelID         nullable.Nullable[string]   `json:"opslevel_id,omitempty"`
		PagerdutyID        nullable.Nullable[string]   `json:"pagerduty_id,omitempty"`
		PagerdutyServiceID nullable.Nullable[string]   `json:"pagerduty_service_id,omitempty"`
		PagertreeID        nullable.Nullable[string]   `json:"pagertree_id,omitempty"`
		Position           nullable.Nullable[int]      `json:"position,omitempty"`
		ServiceNowCiSysID  nullable.Nullable[string]   `json:"service_now_ci_sys_id,omitempty"`
		VictorOpsID        nullable.Nullable[string]   `json:"victor_ops_id,omitempty"`
	}, len(batch))

	for j, e := range batch {
		sdkEntities[j].ExternalID = e.ExternalID
		sdkEntities[j].Name = &e.Name
		if e.BackstageID != "" {
			sdkEntities[j].BackstageID = nullable.NewNullableWithValue(e.BackstageID)
		}
		for k, v := range e.Fields {
			switch k {
			case attrDescription:
				sdkEntities[j].Description = nullable.NewNullableWithValue(v)
			case attrColor:
				sdkEntities[j].Color = nullable.NewNullableWithValue(v)
			case attrBackstageID:
				sdkEntities[j].BackstageID = nullable.NewNullableWithValue(v)
			case attrCortexID:
				sdkEntities[j].CortexID = nullable.NewNullableWithValue(v)
			case attrOpsgenieID:
				sdkEntities[j].OpsgenieID = nullable.NewNullableWithValue(v)
			case attrOpslevelID:
				sdkEntities[j].OpslevelID = nullable.NewNullableWithValue(v)
			case attrPagerdutyID:
				sdkEntities[j].PagerdutyID = nullable.NewNullableWithValue(v)
			case attrPagerdutyServiceID:
				sdkEntities[j].PagerdutyServiceID = nullable.NewNullableWithValue(v)
			case attrPagertreeID:
				sdkEntities[j].PagertreeID = nullable.NewNullableWithValue(v)
			case attrVictorOpsID:
				sdkEntities[j].VictorOpsID = nullable.NewNullableWithValue(v)
			case attrServiceNowCiSysID:
				sdkEntities[j].ServiceNowCiSysID = nullable.NewNullableWithValue(v)
			case attrAlertsEmailEnabled:
				if b, err := strconv.ParseBool(v); err == nil {
					sdkEntities[j].AlertsEmailEnabled = nullable.NewNullableWithValue(b)
				}
			}
		}
		sdkEntities[j].Fields = catalogFields(e, known)
	}

	body := rootly.BulkUpsertTeams{Entities: sdkEntities}
	resp, err := c.sdk.BulkUpsertGroupsWithApplicationVndAPIPlusJSONBodyWithResponse(ctx, body)
	if err != nil {
		return nil, fmt.Errorf("bulk upsert %s: %w", resourceTypePlural("team"), err)
	}
	return parseBulkUpsertResponse(resp.StatusCode(), resp.Body)
}

func parseBulkUpsertResponse(statusCode int, body []byte) (*BulkResult, error) {
	var parsed struct {
		Data   []json.RawMessage `json:"data"`
		Errors []BulkError       `json:"errors"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, fmt.Errorf("decoding bulk upsert response (status %d): %w: %s", statusCode, err, body)
	}
	if statusCode < 200 || statusCode >= 300 {
		return nil, fmt.Errorf("bulk upsert failed (status %d): %s", statusCode, body)
	}
	return &BulkResult{
		Succeeded: len(parsed.Data),
		Errors:    parsed.Errors,
	}, nil
}

// ---------------------------------------------------------------------------
// Bulk Delete
// ---------------------------------------------------------------------------

// BulkDeleteNative deletes native resources by external ID in batches.
func (c *Client) BulkDeleteNative(ctx context.Context, resourceType string, externalIDs []string) (*BulkResult, error) {
	if !IsNativeResource(resourceType) {
		return nil, fmt.Errorf("unsupported native resource type: %s", resourceType)
	}

	result := &BulkResult{}
	for i := 0; i < len(externalIDs); i += MaxBatchSize {
		end := i + MaxBatchSize
		if end > len(externalIDs) {
			end = len(externalIDs)
		}
		batch := externalIDs[i:end]

		br, err := c.bulkDeleteNativeBatch(ctx, resourceType, batch)
		if err != nil {
			return nil, err
		}
		result.DeletedExternalIDs = append(result.DeletedExternalIDs, br.DeletedExternalIDs...)
		result.NotFoundExternalIDs = append(result.NotFoundExternalIDs, br.NotFoundExternalIDs...)
	}
	return result, nil
}

func (c *Client) bulkDeleteNativeBatch(ctx context.Context, resourceType string, batch []string) (*BulkResult, error) {
	switch resourceType {
	case NativeService:
		return c.bulkDeleteServices(ctx, batch)
	case NativeFunctionality:
		return c.bulkDeleteFunctionalities(ctx, batch)
	case NativeEnvironment:
		return c.bulkDeleteEnvironments(ctx, batch)
	case NativeTeam:
		return c.bulkDeleteTeams(ctx, batch)
	default:
		return nil, fmt.Errorf("unsupported native resource type: %s", resourceType)
	}
}

func (c *Client) bulkDeleteServices(ctx context.Context, batch []string) (*BulkResult, error) {
	body := rootly.BulkDestroyServices{ExternalIDs: batch}
	resp, err := c.sdk.BulkDeleteServicesWithApplicationVndAPIPlusJSONBodyWithResponse(ctx, body)
	if err != nil {
		return nil, fmt.Errorf("bulk delete services: %w", err)
	}
	if resp.ApplicationVndAPIJSON200 != nil && resp.ApplicationVndAPIJSON200.Data != nil {
		return &BulkResult{
			DeletedExternalIDs:  resp.ApplicationVndAPIJSON200.Data.DeletedExternalIDs,
			NotFoundExternalIDs: resp.ApplicationVndAPIJSON200.Data.NotFoundExternalIDs,
		}, nil
	}
	return parseBulkDeleteResponse(resp.Body)
}

func (c *Client) bulkDeleteFunctionalities(ctx context.Context, batch []string) (*BulkResult, error) {
	var body rootly.BulkDestroyFunctionalities
	if err := body.FromBulkDestroyFunctionalities0(rootly.BulkDestroyFunctionalities0{
		ExternalIDs: batch,
	}); err != nil {
		return nil, fmt.Errorf("building bulk delete body: %w", err)
	}
	resp, err := c.sdk.BulkDeleteFunctionalitiesWithApplicationVndAPIPlusJSONBodyWithResponse(ctx, body)
	if err != nil {
		return nil, fmt.Errorf("bulk delete functionalities: %w", err)
	}
	if resp.ApplicationVndAPIJSON200 != nil && resp.ApplicationVndAPIJSON200.Data != nil {
		return &BulkResult{
			DeletedExternalIDs:  resp.ApplicationVndAPIJSON200.Data.DeletedExternalIDs,
			NotFoundExternalIDs: resp.ApplicationVndAPIJSON200.Data.NotFoundExternalIDs,
		}, nil
	}
	return parseBulkDeleteResponse(resp.Body)
}

func (c *Client) bulkDeleteEnvironments(ctx context.Context, batch []string) (*BulkResult, error) {
	var body rootly.BulkDestroyEnvironments
	if err := body.FromBulkDestroyEnvironments0(rootly.BulkDestroyEnvironments0{
		ExternalIDs: batch,
	}); err != nil {
		return nil, fmt.Errorf("building bulk delete body: %w", err)
	}
	resp, err := c.sdk.BulkDeleteEnvironmentsWithApplicationVndAPIPlusJSONBodyWithResponse(ctx, body)
	if err != nil {
		return nil, fmt.Errorf("bulk delete environments: %w", err)
	}
	if resp.ApplicationVndAPIJSON200 != nil && resp.ApplicationVndAPIJSON200.Data != nil {
		return &BulkResult{
			DeletedExternalIDs:  resp.ApplicationVndAPIJSON200.Data.DeletedExternalIDs,
			NotFoundExternalIDs: resp.ApplicationVndAPIJSON200.Data.NotFoundExternalIDs,
		}, nil
	}
	return parseBulkDeleteResponse(resp.Body)
}

func (c *Client) bulkDeleteTeams(ctx context.Context, batch []string) (*BulkResult, error) {
	var body rootly.BulkDestroyTeams
	if err := body.FromBulkDestroyTeams0(rootly.BulkDestroyTeams0{
		ExternalIDs: batch,
	}); err != nil {
		return nil, fmt.Errorf("building bulk delete body: %w", err)
	}
	resp, err := c.sdk.BulkDeleteGroupsWithApplicationVndAPIPlusJSONBodyWithResponse(ctx, body)
	if err != nil {
		return nil, fmt.Errorf("bulk delete teams: %w", err)
	}
	if resp.ApplicationVndAPIJSON200 != nil && resp.ApplicationVndAPIJSON200.Data != nil {
		return &BulkResult{
			DeletedExternalIDs:  resp.ApplicationVndAPIJSON200.Data.DeletedExternalIDs,
			NotFoundExternalIDs: resp.ApplicationVndAPIJSON200.Data.NotFoundExternalIDs,
		}, nil
	}
	return parseBulkDeleteResponse(resp.Body)
}

func parseBulkDeleteResponse(body []byte) (*BulkResult, error) {
	var parsed struct {
		DeletedExternalIDs  []string `json:"deleted_external_ids"`
		NotFoundExternalIDs []string `json:"not_found_external_ids"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, fmt.Errorf("decoding bulk delete response: %w", err)
	}
	return &BulkResult{
		DeletedExternalIDs:  parsed.DeletedExternalIDs,
		NotFoundExternalIDs: parsed.NotFoundExternalIDs,
	}, nil
}
