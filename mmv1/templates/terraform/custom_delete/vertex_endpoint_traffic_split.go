_ = userAgent

log.Printf("[DEBUG] Deleting VertexAIEndpointTrafficSplit %q: ", d.Id())

d.Set("traffic_split", map[string]any{})

err = resourceVertexAIEndpointTrafficSplitUpdate(d, meta)
if err != nil {
	d.SetId("")
	return fmt.Errorf("Error trying to delete VertexAIEndpointTrafficSplit: %s", err)
}

return nil
