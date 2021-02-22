terraform {
  required_providers {
    bitwarden = {
      version = "0.2"
      source  = "localhost.localdomain/patricol/bitwarden" // TODO adjust when adding to registry etc.
    }
  }
}

locals {
  creds = jsondecode(file("../creds.json")) // TODO toggle for local development?
}

provider "bitwarden" {
  email = local.creds["email"]
  master_password = local.creds["master_password"]
  client_id = local.creds["client_id"]
  client_secret = local.creds["client_secret"]
  user_id = local.creds["user_id"]
#  two_step_method = local.creds["two_step_method"]
  server = local.creds["server"]
}

data "bitwarden_items" "test" {}

