// Store the ID now
id, err := replaceVars(d, config, "projects/{{project}}/locations/{{location}}/endpoints/{{endpoint}}")
if err != nil {
	return fmt.Errorf("Error constructing id: %s", err)
}
d.SetId(id)

log.Printf("[DEBUG] Creating VertexAIEndpointTrafficSplit %q: ", d.Id())

err = resourceVertexAIEndpointTrafficSplitUpdate(d, meta)
if err != nil {
	d.SetId("")
	return fmt.Errorf("Error trying to create VertexAIEndpointTrafficSplit: %s", err)
}

return nil
