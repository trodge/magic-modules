// Copyright 2024 Google Inc.
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package api

import (
	"fmt"
	"log"
	"regexp"
	"slices"
	"strings"

	"github.com/GoogleCloudPlatform/magic-modules/mmv1/api/resource"
	"github.com/GoogleCloudPlatform/magic-modules/mmv1/google"
)

type TGCResource struct {
	Resource `yaml:",inline"`
	// If true, exclude resource from Terraform Validator
	// (i.e. terraform-provider-conversion)
	ExcludeTgc bool `yaml:"exclude_tgc,omitempty"`

	// If true, include resource in the new package of TGC (terraform-provider-conversion)
	IncludeInTGCNext bool `yaml:"include_in_tgc_next_DO_NOT_USE,omitempty"`

	// Name of the hcl resource block used in TGC
	TgcHclBlockName string `yaml:"tgc_hcl_block_name,omitempty"`

	// The resource kind in CAI.
	// If this is not set, then :name is used instead.
	// For example: compute.googleapis.com/Address has Address for CaiResourceKind,
	// and compute.googleapis.com/GlobalAddress has GlobalAddress for CaiResourceKind.
	// But they have the same api resource type: address
	CaiResourceKind string `yaml:"cai_resource_kind,omitempty"`
}

func (r TGCResource) CaiProductBaseUrl() string {
	version := r.ProductMetadata.VersionObjOrClosest(r.TargetVersionName)
	baseUrl := version.CaiBaseUrl
	if baseUrl == "" {
		baseUrl = version.BaseUrl
	}
	return baseUrl
}

// Gets the CAI product legacy base url.
// For example, https://www.googleapis.com/compute/v1/ for compute
func (r TGCResource) CaiProductLegacyBaseUrl() string {
	version := r.ProductMetadata.VersionObjOrClosest(r.TargetVersionName)
	baseUrl := version.CaiLegacyBaseUrl
	if baseUrl == "" {
		baseUrl = version.CaiBaseUrl
	}
	if baseUrl == "" {
		baseUrl = version.BaseUrl
	}
	return baseUrl
}

// Returns the Cai product backend name from the version base url
// base_url: https://accessapproval.googleapis.com/v1/ -> accessapproval
func (r TGCResource) CaiProductBackendName(caiProductBaseUrl string) string {
	backendUrl := strings.Split(strings.Split(caiProductBaseUrl, "://")[1], ".googleapis.com")[0]
	return strings.ToLower(backendUrl)
}

// Returns the asset type for this resource.
func (r TGCResource) CaiAssetType() string {
	baseURL := r.CaiProductBaseUrl()
	productBackendName := r.CaiProductBackendName(baseURL)
	return fmt.Sprintf("%s.googleapis.com/%s", productBackendName, r.CaiResourceName())
}

// DefineAssetTypeForResourceInProduct marks the AssetType constant for this resource as defined.
// It returns true if this is the first time it's been called for this resource,
// and false otherwise, preventing duplicate definitions.
func (r TGCResource) DefineAssetTypeForResourceInProduct() bool {
	if r.ProductMetadata.ResourcesWithCaiAssetType == nil {
		r.ProductMetadata.ResourcesWithCaiAssetType = make(map[string]struct{}, 1)
	}
	if _, alreadyDefined := r.ProductMetadata.ResourcesWithCaiAssetType[r.CaiResourceType()]; alreadyDefined {
		return false
	}
	r.ProductMetadata.ResourcesWithCaiAssetType[r.CaiResourceType()] = struct{}{}
	return true
}

// Gets the Cai asset name template, which could include version
// For example: //monitoring.googleapis.com/v3/projects/{{project}}/services/{{service_id}}
func (r TGCResource) rawCaiAssetNameTemplate(productBackendName string) string {
	caiBaseUrl := ""
	if r.CaiBaseUrl != "" {
		caiBaseUrl = fmt.Sprintf("%s/{{name}}", r.CaiBaseUrl)
	}
	if caiBaseUrl == "" {
		caiBaseUrl = r.SelfLink
	}
	if caiBaseUrl == "" {
		caiBaseUrl = fmt.Sprintf("%s/{{name}}", r.BaseUrl)
	}
	return fmt.Sprintf("//%s.googleapis.com/%s", productBackendName, caiBaseUrl)
}

// Gets the Cai asset name template, which doesn't include version
// For example: //monitoring.googleapis.com/projects/{{project}}/services/{{service_id}}
func (r TGCResource) CaiAssetNameTemplate(productBackendName string) string {
	template := r.rawCaiAssetNameTemplate(productBackendName)
	versionRegex, err := regexp.Compile(`\/(v\d[^"]*)\/`)
	if err != nil {
		log.Fatalf("Cannot compile the regular expression: %v", err)
	}

	return versionRegex.ReplaceAllString(template, "/")
}

// Gets the Cai API version
func (r TGCResource) CaiApiVersion(productBackendName, caiProductBaseUrl string) string {
	template := r.rawCaiAssetNameTemplate(productBackendName)

	versionRegex, err := regexp.Compile(`\/(v\d[^"]*)\/`)
	if err != nil {
		log.Fatalf("Cannot compile the regular expression: %v", err)
	}

	apiVersion := strings.ReplaceAll(versionRegex.FindString(template), "/", "")
	if apiVersion != "" {
		return apiVersion
	}

	splits := strings.Split(caiProductBaseUrl, "/")
	for i := 0; i < len(splits); i++ {
		if splits[len(splits)-1-i] != "" {
			return splits[len(splits)-1-i]
		}
	}
	return ""
}

// For example: the uri "projects/{{project}}/schemas/{{name}}"
// The paramerter is "schema" as "project" is not returned.
func (r TGCResource) CaiIamResourceParams() []string {
	resourceUri := strings.ReplaceAll(r.IamResourceUri(), "{{name}}", fmt.Sprintf("{{%s}}", r.IamParentResourceName()))

	return google.Reject(r.ExtractIdentifiers(resourceUri), func(param string) bool {
		return param == "project"
	})
}

// Gets the Cai IAM asset name template
// For example: //monitoring.googleapis.com/v3/projects/{{project}}/services/{{service_id}}
func (r TGCResource) CaiIamAssetNameTemplate(productBackendName string) string {
	iamImportFormat := r.IamImportFormats()
	if len(iamImportFormat) > 0 {
		name := strings.ReplaceAll(iamImportFormat[0], "{{name}}", fmt.Sprintf("{{%s}}", r.IamParentResourceName()))
		name = strings.ReplaceAll(name, "%", "")
		return fmt.Sprintf("//%s.googleapis.com/%s", productBackendName, name)
	}

	caiBaseUrl := r.CaiBaseUrl

	if caiBaseUrl == "" {
		caiBaseUrl = r.SelfLink
	}
	if caiBaseUrl == "" {
		caiBaseUrl = r.BaseUrl
	}
	return fmt.Sprintf("//%s.googleapis.com/%s/{{%s}}", productBackendName, caiBaseUrl, r.IamParentResourceName())
}

// TGC Methods
// ====================
// Lists fields that test.BidirectionalConversion should ignore
func (r TGCResource) TGCTestIgnorePropertiesToStrings(e resource.Examples) []string {
	props := []string{
		"depends_on",
		"count",
		"for_each",
		"provider",
		"lifecycle",
	}
	for _, tp := range r.VirtualFields {
		props = append(props, google.Underscore(tp.Name))
	}
	for _, tp := range r.AllNestedProperties(r.RootProperties()) {
		if tp.UrlParamOnly {
			props = append(props, google.Underscore(tp.Name))
		} else if tp.IsMissingInCai {
			props = append(props, tp.MetadataLineage())
		}
	}
	props = append(props, e.TGCTestIgnoreExtra...)

	slices.Sort(props)
	return props
}

// Filters out computed properties during cai2hcl
func (r TGCResource) ReadPropertiesForTgc() []*Type {
	return google.Reject(r.AllUserProperties(), func(v *Type) bool {
		return v.Output || v.UrlParamOnly
	})
}

// For example, the CAI resource type with product of "google_compute_autoscaler" is "ComputeAutoscalerAssetType".
// The CAI resource type with product of "google_compute_region_autoscaler" is also "ComputeAutoscalerAssetType".
func (r TGCResource) CaiResourceType() string {
	return fmt.Sprintf("%s%s", r.ProductMetadata.Name, r.CaiResourceName())
}

// The API resource type of the resource. Normally, it is the resource name.
// Rarely, it is the API "resource type kind" or CAI "resource kind"
// For example, the CAI resource type of "google_compute_autoscaler" is "Autoscaler".
// The CAI resource type of "google_compute_region_autoscaler" is also "Autoscaler".
func (r TGCResource) CaiResourceName() string {
	if r.CaiResourceKind != "" {
		return r.CaiResourceKind
	}
	if r.ApiResourceTypeKind != "" {
		return r.ApiResourceTypeKind
	}
	return r.Name
}
