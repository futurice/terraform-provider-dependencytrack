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
  active = true
}

resource "dependencytrack_project" "sub" {
  name = "bar"
  classifier = "APPLICATION"
  parent_id = dependencytrack_project.main.id
  active = true
}

resource "dependencytrack_project" "sub2" {
  name = "baz"
  classifier = "APPLICATION"
  # parent_id = dependencytrack_project.main.id
  active = true
}

output "team" {
  value = resource.dependencytrack_team.main
}

output "projecct" {
  value = resource.dependencytrack_project.main
}
