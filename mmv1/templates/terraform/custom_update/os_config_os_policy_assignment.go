userAgent, err := generateUserAgentString(d, config.UserAgent)
if err != nil {
	return err
}

billingProject := ""

project, err := getProject(d, config)
if err != nil {
	return fmt.Errorf("Error fetching project for OSPolicyAssignment: %s", err)
}
billingProject = project

obj := make(map[string]interface{})
descriptionProp, err := expandOSConfigOSPolicyAssignmentDescription(d.Get("description"), d, config)
if err != nil {
	return err
} else if v, ok := d.GetOkExists("description"); !isEmptyValue(reflect.ValueOf(v)) && (ok || !reflect.DeepEqual(v, descriptionProp)) {
	obj["description"] = descriptionProp
}
osPoliciesProp, err := expandOSConfigOSPolicyAssignmentOsPolicies(d.Get("os_policies"), d, config)
if err != nil {
	return err
} else if v, ok := d.GetOkExists("os_policies"); !isEmptyValue(reflect.ValueOf(v)) && (ok || !reflect.DeepEqual(v, osPoliciesProp)) {
	obj["osPolicies"] = osPoliciesProp
}
instanceFilterProp, err := expandOSConfigOSPolicyAssignmentInstanceFilter(d.Get("instance_filter"), d, config)
if err != nil {
	return err
} else if v, ok := d.GetOkExists("instance_filter"); !isEmptyValue(reflect.ValueOf(v)) && (ok || !reflect.DeepEqual(v, instanceFilterProp)) {
	obj["instanceFilter"] = instanceFilterProp
}
rolloutProp, err := expandOSConfigOSPolicyAssignmentRollout(d.Get("rollout"), d, config)
if err != nil {
	return err
} else if v, ok := d.GetOkExists("rollout"); !isEmptyValue(reflect.ValueOf(v)) && (ok || !reflect.DeepEqual(v, rolloutProp)) {
	obj["rollout"] = rolloutProp
}

url, err := ReplaceVars(d, config, "{{OSConfigBasePath}}projects/{{project}}/locations/{{location}}/osPolicyAssignments/{{name}}")
if err != nil {
	return err
}
url = strings.ReplaceAll(url, "https://osconfig.googleapis.com/v1beta", "https://osconfig.googleapis.com/v1")

log.Printf("[DEBUG] Updating OSPolicyAssignment %q: %#v", d.Id(), obj)
updateMask := []string{}

if d.HasChange("description") {
	updateMask = append(updateMask, "description")
}

if d.HasChange("os_policies") {
	updateMask = append(updateMask, "osPolicies")
}

if d.HasChange("instance_filter") {
	updateMask = append(updateMask, "instanceFilter")
}

if d.HasChange("rollout") {
	updateMask = append(updateMask, "rollout")
}
// updateMask is a URL parameter but not present in the schema, so ReplaceVars
// won't set it
url, err = AddQueryParams(url, map[string]string{"updateMask": strings.Join(updateMask, ",")})
if err != nil {
	return err
}

// err == nil indicates that the billing_project value was found
if bp, err := getBillingProject(d, config); err == nil {
	billingProject = bp
}

res, err := SendRequestWithTimeout(config, "PATCH", billingProject, url, userAgent, obj, d.Timeout(schema.TimeoutUpdate))

if err != nil {
	return fmt.Errorf("Error updating OSPolicyAssignment %q: %s", d.Id(), err)
} else {
	log.Printf("[DEBUG] Finished updating OSPolicyAssignment %q: %#v", d.Id(), res)
}

if skipAwaitRollout := d.Get("skip_await_rollout").(bool); !skipAwaitRollout {
	err = OSConfigOperationWaitTime(
		config, res, project, "Updating OSPolicyAssignment", userAgent,
		d.Timeout(schema.TimeoutUpdate))

	if err != nil {
		return err
	}
}

return resourceOSConfigOSPolicyAssignmentRead(d, meta)
