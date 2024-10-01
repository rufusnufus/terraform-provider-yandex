package resource_cluster

import (
	"context"
	"fmt"
	"regexp"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework-validators/mapvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/yandex-cloud/go-genproto/yandex/cloud/logging/v1"
	ycsdk "github.com/yandex-cloud/go-sdk"

	provider_config "github.com/yandex-cloud/terraform-provider-yandex/yandex-framework/provider/config"
)

const (
	yandexAirflowClusterCreateTimeout = 30 * time.Minute
	yandexAirflowClusterDeleteTimeout = 15 * time.Minute
	yandexAirflowClusterUpdateTimeout = 60 * time.Minute

	adminPasswordStubOnImport = "<real value unknown because resource was imported>"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &airflowClusterResource{}
var _ resource.ResourceWithImportState = &airflowClusterResource{}
var _ resource.ResourceWithValidateConfig = &airflowClusterResource{}

func NewResource() resource.Resource {
	return &airflowClusterResource{}
}

type airflowClusterResource struct {
	providerConfig *provider_config.Config
}

// Metadata implements resource.Resource.
func (a *airflowClusterResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_airflow_cluster"
}

// Configure implements resource.Resource.
func (a *airflowClusterResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	providerConfig, ok := req.ProviderData.(*provider_config.Config)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *provider_config.Config, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	a.providerConfig = providerConfig
}

// ImportState implements resource.ResourceWithImportState.
func (a *airflowClusterResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)

	adminPassword := path.Root("admin_password")
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, adminPassword, adminPasswordStubOnImport)...)
}

// Create implements resource.Resource.
func (a *airflowClusterResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan ClusterModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	createClusterRequest, diags := buildCreateClusterRequest(ctx, &plan, &a.providerConfig.ProviderState)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Debug(ctx, fmt.Sprintf("Create Airflow cluster request: %+v", createClusterRequest))

	createTimeout, diags := plan.Timeouts.Create(ctx, yandexAirflowClusterCreateTimeout)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, createTimeout)
	defer cancel()

	clusterID, d := createCluster(ctx, a.providerConfig.SDK, &resp.Diagnostics, createClusterRequest)
	resp.Diagnostics.Append(d)
	if resp.Diagnostics.HasError() {
		return
	}

	plan.Id = types.StringValue(clusterID)
	diags = updateState(ctx, a.providerConfig.SDK, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)

	tflog.Debug(ctx, "Finished creating Airflow cluster", clusterIDLogField(clusterID))
}

// Delete implements resource.Resource.
func (a *airflowClusterResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state ClusterModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	clusterID := state.Id.ValueString()
	tflog.Debug(ctx, "Deleting Airflow cluster", clusterIDLogField(clusterID))

	deleteTimeout, diags := state.Timeouts.Delete(ctx, yandexAirflowClusterDeleteTimeout)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, deleteTimeout)
	defer cancel()

	d := deleteCluster(ctx, a.providerConfig.SDK, clusterID)
	resp.Diagnostics.Append(d)

	tflog.Debug(ctx, "Finished deleting Airflow cluster", clusterIDLogField(clusterID))
}

// Read implements resource.Resource.
func (a *airflowClusterResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state ClusterModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	clusterID := state.Id.ValueString()
	tflog.Debug(ctx, "Reading Airflow cluster", clusterIDLogField(clusterID))
	cluster, d := getClusterByID(ctx, a.providerConfig.SDK, clusterID)
	resp.Diagnostics.Append(d)
	if resp.Diagnostics.HasError() {
		return
	}

	if cluster == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	diags = clusterToState(ctx, cluster, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	tflog.Debug(ctx, "Finished reading Airflow cluster", clusterIDLogField(clusterID))
}

// Update implements resource.Resource.
func (a *airflowClusterResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan ClusterModel
	var state ClusterModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Updating Airflow cluster", clusterIDLogField(state.Id.ValueString()))

	updateTimeout, diags := plan.Timeouts.Update(ctx, yandexAirflowClusterUpdateTimeout)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, updateTimeout)
	defer cancel()

	tflog.Debug(ctx, fmt.Sprintf("Update Airflow cluster state: %+v", state))
	tflog.Debug(ctx, fmt.Sprintf("Update Airflow cluster plan: %+v", plan))

	updateReq, diags := buildUpdateClusterRequest(ctx, &state, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Debug(ctx, fmt.Sprintf("Update Airflow cluster request: %+v", updateReq))

	d := updateCluster(ctx, a.providerConfig.SDK, updateReq)
	resp.Diagnostics.Append(d)
	if resp.Diagnostics.HasError() {
		return
	}

	diags = updateState(ctx, a.providerConfig.SDK, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
	tflog.Debug(ctx, "Finished updating Airflow cluster", clusterIDLogField(state.Id.ValueString()))
}

// Schema implements resource.Resource.
func (a *airflowClusterResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = ClusterResourceSchema(ctx)
	resp.Schema.Blocks["timeouts"] = timeouts.Block(ctx, timeouts.Opts{
		Create: true,
		Update: true,
		Delete: true,
	})
}

func (r *airflowClusterResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var cluster ClusterModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &cluster)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if !cluster.Logging.IsNull() {
		// both folder_id and log_group_id are specified or both are not specified
		if cluster.Logging.FolderId.IsNull() == cluster.Logging.LogGroupId.IsNull() {
			resp.Diagnostics.AddAttributeError(
				path.Root("logging"),
				"Invalid Airflow cluster logging configuration",
				"Exactly one of the attributes `folder_id` and `log_group_id` must be specified",
			)
			return
		}
	}
}

func airflowConfigValidator() validator.Map {
	return mapvalidator.KeysAre(stringvalidator.RegexMatches(
		regexp.MustCompile(`^[^\.]*$`),
		"must not contain dots",
	))
}

func allowedLogLevels() []string {
	allowedLevels := make([]string, 0, len(logging.LogLevel_Level_value))
	for levelName, val := range logging.LogLevel_Level_value {
		if val == 0 {
			continue
		}
		allowedLevels = append(allowedLevels, levelName)
	}
	return allowedLevels
}

func logLevelValidator() validator.String {
	return stringvalidator.OneOf(allowedLogLevels()...)
}

func updateState(ctx context.Context, sdk *ycsdk.SDK, state *ClusterModel) diag.Diagnostics {
	var diags diag.Diagnostics
	clusterID := state.Id.ValueString()
	tflog.Debug(ctx, "Reading Airflow cluster", clusterIDLogField(clusterID))
	cluster, d := getClusterByID(ctx, sdk, clusterID)
	diags.Append(d)
	if diags.HasError() {
		return diags
	}

	if cluster == nil {
		diags.AddError(
			"Airflow cluster not found",
			fmt.Sprintf("Airflow cluster with id %s not found", clusterID))
		return diags
	}

	dd := clusterToState(ctx, cluster, state)
	diags.Append(dd...)
	return diags
}

func clusterIDLogField(cid string) map[string]interface{} {
	return map[string]interface{}{
		"cluster_id": cid,
	}
}