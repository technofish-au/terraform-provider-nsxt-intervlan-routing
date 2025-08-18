// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure NsxtIntervlanRoutingProvider satisfies various provider interfaces.
var _ provider.Provider = &NsxtIntervlanRoutingProvider{}

// var _ provider.ProviderWithFunctions = &NsxtIntervlanRoutingProvider{}.
var Client http.Client
var Auth AuthResponse
var Host string

type AuthResponse struct {
	Session   string
	Path      string
	Secure    bool
	HttpOnly  bool
	SameSite  string
	XsrfToken string
}

func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &NsxtIntervlanRoutingProvider{
			version: version,
		}
	}
}

// NsxtIntervlanRoutingProvider defines the provider implementation.
type NsxtIntervlanRoutingProvider struct {
	// version is set to the provider version on release, "dev" when the
	// provider is built and ran locally, and "test" when running acceptance
	// testing.
	version string
}

// Metadata returns the provider type name.
func (p *NsxtIntervlanRoutingProvider) Metadata(ctx context.Context, req provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "nsxt-intervlan-routing"
	resp.Version = p.version
}

// NsxtIntervlanRoutingProviderModel describes the provider data model.
type NsxtIntervlanRoutingProviderModel struct {
	NsxtInsecure types.Bool   `tfsdk:"nsxt_insecure"`
	NsxtUsername types.String `tfsdk:"nsxt_username"`
	NsxtPassword types.String `tfsdk:"nsxt_password"`
	NsxtHost     types.String `tfsdk:"nsxt_host"`
}

func (p *NsxtIntervlanRoutingProvider) Schema(ctx context.Context, req provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"allow_insecure": schema.BoolAttribute{
				Optional:    true,
				Description: "Allow insecure SSL connections",
			},
			"username": schema.StringAttribute{
				Optional:    true,
				Description: "The username used to authenticate the API calls to NSX.",
			},
			"password": schema.StringAttribute{
				Optional:    true,
				Sensitive:   true,
				Description: "The password used to authenticate the API calls to NSX.",
			},
			"host": schema.StringAttribute{
				Optional:    true,
				Description: "The hostname or IP address of the NSX API.",
			},
		},
		Blocks:      map[string]schema.Block{},
		Description: "Interface with the NSX API.",
	}
}

func (p *NsxtIntervlanRoutingProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	tflog.Info(ctx, "Configuring NSX InterVLAN Routing client")
	var config NsxtIntervlanRoutingProviderModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if config.NsxtInsecure.IsUnknown() {
		config.NsxtInsecure = types.BoolValue(false)
	}
	if config.NsxtHost.IsUnknown() {
		resp.Diagnostics.AddAttributeError(
			path.Root("host"),
			"Unknown NSX InterVLAN Routing host",
			"The provider cannot create the NSX InterVLAN Routing client as there is an unknown configuration value for the API host. "+
				"Either target apply the source of the value first, set the value statically in the configuration, or use the NSXT_HOST environment variable.",
		)
	}
	if config.NsxtUsername.IsUnknown() {
		resp.Diagnostics.AddAttributeError(
			path.Root("username"),
			"Unknown NSX InterVLAN Routing username",
			"The provider cannot create the NSX InterVLAN Routing client as there is an unknown configuration value for the API username. "+
				"Either target apply the source of the value first, set the value statically in the configuration, or use the NSXT_USERNAME environment variable.",
		)
	}
	if config.NsxtPassword.IsUnknown() {
		resp.Diagnostics.AddAttributeError(
			path.Root("password"),
			"Unknown NSX InterVLAN Routing password",
			"The provider cannot create the NSX InterVLAN Routing client as there is an unknown configuration value for the API password. "+
				"Either target apply the source of the value first, set the value statically in the configuration, or use the NSXT_PASSWORD environment variable.",
		)
	}
	if resp.Diagnostics.HasError() {
		return
	}

	// Default values to environment variables, but override
	// with Terraform configuration value if set.
	insecure := os.Getenv("NSXT_INSECURE")
	hostname := os.Getenv("NSXT_HOSTNAME")
	username := os.Getenv("NSXT_USERNAME")
	password := os.Getenv("NSXT_PASSWORD")

	if !config.NsxtInsecure.IsNull() {
		insecure = config.NsxtInsecure.String()
	}
	if !config.NsxtHost.IsNull() {
		hostname = config.NsxtHost.ValueString()
	}
	if !config.NsxtUsername.IsNull() {
		username = config.NsxtUsername.ValueString()
	}
	if !config.NsxtPassword.IsNull() {
		password = config.NsxtPassword.ValueString()
	}

	// If any of the expected configurations are missing, return
	// errors with provider-specific guidance.

	if insecure == "" {
		resp.Diagnostics.AddAttributeWarning(
			path.Root("insecure"),
			"Missing NSX-T Manager API Insecure (using default value: false)",
			"The provider is using a default value as there is a missing or empty value for the NSX-T Manager API insecure. "+
				"Set the insecure value in the configuration or use the NSXT_INSECURE environment variable. "+
				"If either is already set, ensure the value is not empty.",
		)
		//insecure = "false"
	}
	if hostname == "" {
		resp.Diagnostics.AddAttributeWarning(
			path.Root("host"),
			"Missing NSX-T Manager API Hostname (using default value: 127.0.0.1)",
			"The provider is using a default value as there is a missing or empty value for the NSX-T Manager API hostname. "+
				"Set the host value in the configuration or use the NSXT_HOSTNAME environment variable. "+
				"If either is already set, ensure the value is not empty.",
		)
		hostname = "127.0.0.1"
	}
	if username == "" {
		resp.Diagnostics.AddAttributeWarning(
			path.Root("username"),
			"Missing NSX-T API username (using default value: admin)",
			"The provider is using a default value as there is a missing or empty value for the NSX-T API username. "+
				"Set the username value in the configuration or use the NSXT_USERNAME environment variable. "+
				"If either is already set, ensure the value is not empty.",
		)
		username = "admin"
	}
	if password == "" {
		resp.Diagnostics.AddAttributeWarning(
			path.Root("password"),
			"Missing NSX-T API port (using default value: password)",
			"The provider is using a default value as there is a missing or empty value for the NSX-T API password. "+
				"Set the password value in the configuration or use the NSXT_PASSWORD environment variable. "+
				"If either is already set, ensure the value is not empty.",
		)
		password = "password"
	}
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Creating NSX-T API client")

	// Create the configuration for the NSX-T API Client
	//is_insecure, _ := strconv.ParseBool(insecure)
	Host = hostname

	creds := url.Values{}
	creds.Set("j_username", username)
	creds.Set("j_password", password)
	enc_creds := creds.Encode()

	// Example client configuration for data sources and resources
	Client := &http.Client{
		Timeout: 10 * time.Second,
	}
	request, err := http.NewRequest(
		"POST",
		hostname+"/api/session/create",
		strings.NewReader(enc_creds))
	if err != nil {
		resp.Diagnostics.AddError(
			"Error occurred configuring the client parameters",
			"An unexpected error occurred when configuring the NSX-T API client. "+
				"If the error is not clear, please contact the provider developers.\n\n"+
				"NSX-T Client Error: "+err.Error(),
		)
		return
	}

	response, err := Client.Do(request)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Create NSX-T API Client",
			"An unexpected error occurred when creating the NSX-T API client. "+
				"If the error is not clear, please contact the provider developers.\n\n"+
				"NSX-T Client Error: "+err.Error(),
		)
		return
	}
	defer response.Body.Close()

	if response.StatusCode == http.StatusOK {
		body, err := io.ReadAll(response.Body)
		if err != nil {
			resp.Diagnostics.AddError(
				"Error reading the API response",
				"An unexpected error occurred when reading the NSX-T API client response. "+
					"If the error is not clear, please contact the provider developers.\n\n"+
					"NSX-T Client Error: "+err.Error(),
			)
			return
		}
		Auth := convertBodyToMap(string(body))
		_ = Auth

		// Make the Inventory client available during DataSource and Resource
		// type Configure methods.
		resp.DataSourceData = Client
		resp.ResourceData = Client

		tflog.Info(ctx, "Configured NSX-T client", map[string]any{"success": true})
	} else {
		resp.Diagnostics.AddError(
			"NSX-T API Client returned a non-200 status code",
			"The NSX-T API Client returned a non-200 status code. The response returned "+
				"indicates an error authenticating the client.\n\n"+
				"NSX-T Client Error: "+err.Error(),
		)
		tflog.Info(ctx, "Configured NSX-T client", map[string]any{"success": false})

		return
	}
}

func convertBodyToMap(bodyString string) AuthResponse {
	dataMap := make(map[string]string)
	parts := strings.Split(bodyString, ":")
	for _, part := range parts {
		kv := strings.SplitN(part, "=", 2) // Split only on the first '='
		if len(kv) == 2 {
			key := strings.TrimSpace(kv[0])
			value := strings.TrimSpace(kv[1])
			dataMap[key] = value
		}
	}

	is_secure, _ := strconv.ParseBool(dataMap["secure"])
	is_http_only, _ := strconv.ParseBool(dataMap["http_only"])

	response := AuthResponse{
		Session:   dataMap["JSESSIONID"],
		Path:      dataMap["Path"],
		Secure:    is_secure,
		HttpOnly:  is_http_only,
		SameSite:  dataMap["SameSite"],
		XsrfToken: dataMap["x-xsrf-token"],
	}

	return response
}

func (p *NsxtIntervlanRoutingProvider) Resources(ctx context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		//NewExampleResource,
	}
}

//func (p *NsxtIntervlanRoutingProvider) EphemeralResources(ctx context.Context) []func() ephemeral.EphemeralResource {
//	return []func() ephemeral.EphemeralResource{
//		NewExampleEphemeralResource,
//	}
//}

func (p *NsxtIntervlanRoutingProvider) DataSources(ctx context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		NewSegmentPortsDataSource,
	}
}

//func (p *NsxtIntervlanRoutingProvider) Functions(ctx context.Context) []func() function.Function {
//	return []func() function.Function{
//		NewExampleFunction,
//	}
//}
