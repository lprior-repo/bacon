locals {
  # Template variables for different scraper types
  scraper_configs = {
    amplify = {
      service_type       = "amplify-console"
      resource_type     = "amplify:app"
      data_field        = "amplifyData"
      cache_ttl_ms      = "3600000"  # 1 hour
      ttl_ms           = "86400000"  # 24 hours
      update_type      = "amplify_scrape"
      vertex_transformation = "$map($states.input.data.amplifyApps, function($app) { {'id': $app.appId, 'label': 'AmplifyApp', 'properties': {'name': $app.detailedInfo.name, 'region': $app.region, 'arn': $app.resourceArn}} })"
      edge_transformation = "$reduce($states.input.data.amplifyApps, function($acc, $app) { $append($acc, [{'from': $states.input.accountId, 'to': $app.appId, 'label': 'OWNS', 'properties': {'relationship': 'ownership'}}]) }, [])"
    }
    
    ec2 = {
      service_type       = "ec2-instances"
      resource_type     = "ec2:instance"
      data_field        = "ec2Data"
      cache_ttl_ms      = "1800000"  # 30 minutes
      ttl_ms           = "86400000"  # 24 hours
      update_type      = "ec2_scrape"
      vertex_transformation = "$map($states.input.data.ec2Instances, function($instance) { {'id': $instance.instanceId, 'label': 'EC2Instance', 'properties': {'instanceType': $instance.detailedInfo.InstanceType, 'state': $instance.detailedInfo.State.Name, 'region': $instance.region}} })"
      edge_transformation = "$reduce($states.input.data.ec2Instances, function($acc, $instance) { $append($acc, [{'from': $states.input.accountId, 'to': $instance.instanceId, 'label': 'OWNS', 'properties': {'relationship': 'ownership'}}]) }, [])"
    }
  }
}

# Template data sources
data "template_file" "cache_pattern" {
  for_each = local.scraper_configs
  
  template = file("${path.module}/cache-pattern.json")
  
  vars = {
    NEXT_STATE_ON_ERROR    = "Discover${title(each.key)}ResourcesWithExplorer"
    CACHE_TTL_MS          = each.value.cache_ttl_ms
    NEXT_STATE_ON_CACHE_MISS = "Discover${title(each.key)}ResourcesWithExplorer"
    DATA_FIELD            = each.value.data_field
  }
}

data "template_file" "resource_explorer_pattern" {
  for_each = local.scraper_configs
  
  template = file("${path.module}/resource-explorer-pattern.json")
  
  vars = {
    ERROR_HANDLER_STATE     = "Handle${title(each.key)}ExplorerError"
    NEXT_PROCESSING_STATE   = "Enrich${title(each.key)}ResourcesWithDetails"
  }
}

data "template_file" "dynamodb_storage_pattern" {
  for_each = local.scraper_configs
  
  template = file("${path.module}/dynamodb-storage-pattern.json")
  
  vars = {
    DATA_FIELD = each.value.data_field
    TTL_MS     = each.value.ttl_ms
    NEXT_STATE = "PrepareNeptuneUpdate"
  }
}

data "template_file" "neptune_update_pattern" {
  for_each = local.scraper_configs
  
  template = file("${path.module}/neptune-update-pattern.json")
  
  vars = {
    UPDATE_TYPE           = each.value.update_type
    VERTEX_TRANSFORMATION = each.value.vertex_transformation
    EDGE_TRANSFORMATION   = each.value.edge_transformation
    NEXT_STATE           = "Finalize${title(each.key)}ScraperResults"
  }
}

# Generate complete step function definitions
data "template_file" "complete_scraper_definition" {
  for_each = local.scraper_configs
  
  template = file("${path.module}/scraper-template.json")
  
  vars = {
    SERVICE_TYPE            = each.value.service_type
    RESOURCE_TYPE          = each.value.resource_type
    CACHE_PATTERN          = data.template_file.cache_pattern[each.key].rendered
    RESOURCE_EXPLORER_PATTERN = data.template_file.resource_explorer_pattern[each.key].rendered
    DYNAMODB_STORAGE_PATTERN = data.template_file.dynamodb_storage_pattern[each.key].rendered
    NEPTUNE_UPDATE_PATTERN = data.template_file.neptune_update_pattern[each.key].rendered
  }
}

# Output the generated step function definitions
output "generated_step_functions" {
  description = "Generated Step Function definitions using templates"
  value = {
    for scraper_type, config in local.scraper_configs :
    scraper_type => data.template_file.complete_scraper_definition[scraper_type].rendered
  }
}