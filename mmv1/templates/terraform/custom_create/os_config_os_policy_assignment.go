userAgent, err := generateUserAgentString(d, config.UserAgent)
if err != nil {
	return err
}

obj := make(map[string]interface{})
nameProp, err := expandOSConfigOSPolicyAssignmentName(d.Get("name"), d, config)
if err != nil {
	return err
} else if v, ok := d.GetOkExists("name"); !isEmptyValue(reflect.ValueOf(nameProp)) && (ok || !reflect.DeepEqual(v, nameProp)) {
	obj["name"] = nameProp
}
descriptionProp, err := expandOSConfigOSPolicyAssignmentDescription(d.Get("description"), d, config)
if err != nil {
	return err
} else if v, ok := d.GetOkExists("description"); !isEmptyValue(reflect.ValueOf(descriptionProp)) && (ok || !reflect.DeepEqual(v, descriptionProp)) {
	obj["description"] = descriptionProp
}
osPoliciesProp, err := expandOSConfigOSPolicyAssignmentOsPolicies(d.Get("os_policies"), d, config)
if err != nil {
	return err
} else if v, ok := d.GetOkExists("os_policies"); !isEmptyValue(reflect.ValueOf(osPoliciesProp)) && (ok || !reflect.DeepEqual(v, osPoliciesProp)) {
	obj["osPolicies"] = osPoliciesProp
}
instanceFilterProp, err := expandOSConfigOSPolicyAssignmentInstanceFilter(d.Get("instance_filter"), d, config)
if err != nil {
	return err
} else if v, ok := d.GetOkExists("instance_filter"); !isEmptyValue(reflect.ValueOf(instanceFilterProp)) && (ok || !reflect.DeepEqual(v, instanceFilterProp)) {
	obj["instanceFilter"] = instanceFilterProp
}
rolloutProp, err := expandOSConfigOSPolicyAssignmentRollout(d.Get("rollout"), d, config)
if err != nil {
	return err
} else if v, ok := d.GetOkExists("rollout"); !isEmptyValue(reflect.ValueOf(rolloutProp)) && (ok || !reflect.DeepEqual(v, rolloutProp)) {
	obj["rollout"] = rolloutProp
}

url, err := ReplaceVars(d, config, "{{OSConfigBasePath}}projects/{{project}}/locations/{{location}}/osPolicyAssignments?osPolicyAssignmentId={{name}}")
if err != nil {
	return err
}
url = strings.ReplaceAll(url, "https://osconfig.googleapis.com/v1beta", "https://osconfig.googleapis.com/v1")

log.Printf("[DEBUG] Creating new OSPolicyAssignment: %#v", obj)
billingProject := ""

project, err := getProject(d, config)
if err != nil {
	return fmt.Errorf("Error fetching project for OSPolicyAssignment: %s", err)
}
billingProject = project

// err == nil indicates that the billing_project value was found
if bp, err := getBillingProject(d, config); err == nil {
	billingProject = bp
}

res, err := SendRequestWithTimeout(config, "POST", billingProject, url, userAgent, obj, d.Timeout(schema.TimeoutCreate))
if err != nil {
	return fmt.Errorf("Error creating OSPolicyAssignment: %s", err)
}

// Store the ID now
id, err := ReplaceVars(d, config, "projects/{{project}}/locations/{{location}}/osPolicyAssignments/{{name}}")
if err != nil {
	return fmt.Errorf("Error constructing id: %s", err)
}
d.SetId(id)

if skipAwaitRollout := d.Get("skip_await_rollout").(bool); !skipAwaitRollout {
	// Use the resource in the operation response to populate
	// identity fields and d.Id() before read
	var opRes map[string]interface{}
	err = OSConfigOperationWaitTimeWithResponse(
		config, res, &opRes, project, "Creating OSPolicyAssignment", userAgent,
		d.Timeout(schema.TimeoutCreate))
	if err != nil {
		// The resource didn't actually create
		d.SetId("")

		return fmt.Errorf("Error waiting to create OSPolicyAssignment: %s", err)
	}

	if err := d.Set("name", flattenOSConfigOSPolicyAssignmentName(opRes["name"], d, config)); err != nil {
		return err
	}

	// This may have caused the ID to update - update it if so.
	id, err = ReplaceVars(d, config, "projects/{{project}}/locations/{{location}}/osPolicyAssignments/{{name}}")
	if err != nil {
		return fmt.Errorf("Error constructing id: %s", err)
	}
	d.SetId(id)
}

log.Printf("[DEBUG] Finished creating OSPolicyAssignment %q: %#v", d.Id(), res)

return resourceOSConfigOSPolicyAssignmentRead(d, meta)
