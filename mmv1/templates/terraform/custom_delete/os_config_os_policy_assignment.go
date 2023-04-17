billingProject := ""

project, err := getProject(d, config)
if err != nil {
	return fmt.Errorf("Error fetching project for OSPolicyAssignment: %s", err)
}
billingProject = project

url, err := ReplaceVars(d, config, "{{OSConfigBasePath}}projects/{{project}}/locations/{{location}}/osPolicyAssignments/{{name}}")
if err != nil {
	return err
}
url = strings.ReplaceAll(url, "https://osconfig.googleapis.com/v1beta", "https://osconfig.googleapis.com/v1")

var obj map[string]interface{}
log.Printf("[DEBUG] Deleting OSPolicyAssignment %q", d.Id())

// err == nil indicates that the billing_project value was found
if bp, err := getBillingProject(d, config); err == nil {
	billingProject = bp
}

res, err := SendRequestWithTimeout(config, "DELETE", billingProject, url, userAgent, obj, d.Timeout(schema.TimeoutDelete))
if err != nil {
	return handleNotFoundError(err, d, "OSPolicyAssignment")
}

if skipAwaitRollout := d.Get("skip_await_rollout").(bool); !skipAwaitRollout {
	err = OSConfigOperationWaitTime(
		config, res, project, "Deleting OSPolicyAssignment", userAgent,
		d.Timeout(schema.TimeoutDelete))

	if err != nil {
		return err
	}
}

log.Printf("[DEBUG] Finished deleting OSPolicyAssignment %q: %#v", d.Id(), res)
return nil
