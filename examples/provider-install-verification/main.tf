terraform {
  required_providers {
    dependencytrack = {
      source = "registry.terraform.io/futurice/dependencytrack"
    }
  }
}

provider "dependencytrack" {
  host = "http://localhost:8081"
  api_key = "odt_2gbKwjq2j0RT9kESnq8J6LUwxQ4IfFeB"
}

resource "dependencytrack_team" "main" {
  name = "bar"
}

resource "dependencytrack_team_permission" "main" {
  team_id = dependencytrack_team.main.id
  name = "BOM_UPLOAD"
}

resource "dependencytrack_team_permission" "main2" {
  team_id = dependencytrack_team.main.id
  name = "ACCESS_MANAGEMENT"
}

resource "dependencytrack_project" "main" {
  name = "foo"
  classifier = "APPLICATION"
}

resource "dependencytrack_project" "sub" {
  name = "bar"
  classifier = "APPLICATION"
  parent_id = dependencytrack_project.main.id
}

resource "dependencytrack_project" "sub2" {
  name = "baz"
  classifier = "APPLICATION"
  # parent_id = dependencytrack_project.main.id
}

resource "dependencytrack_acl_mapping" "main" {
  team_id = dependencytrack_team.main.id
  project_id = dependencytrack_project.main.id
}

data "dependencytrack_notification_publisher" "main" {
  name = "Outbound Webhook"
}

resource "dependencytrack_notification_rule" "main" {
  name = "foo"
  scope = "PORTFOLIO"
  notification_level = "INFORMATIONAL"
  publisher_id = data.dependencytrack_notification_publisher.main.id
  publisher_config = jsonencode({
    destination = "http://localhost:8080"
  })
  notify_on = ["NEW_VULNERABILITY"]
}

output "team" {
  value = dependencytrack_team.main
}

output "project" {
  value = dependencytrack_project.main
}

output "publisher" {
  value = data.dependencytrack_notification_publisher.main
}

output "rule" {
  value = dependencytrack_notification_rule.main
}
