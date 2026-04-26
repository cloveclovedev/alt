data "external" "env" {
  program = ["python3", "load_env.py"]
}

locals {
  env    = jsondecode(data.external.env)
  db_url = "postgresql://${local.env.NEON_USER}:${local.env.NEON_PASSWORD}@${local.env.NEON_HOST}/${local.env.NEON_DATABASE}?sslmode=require"
}

env "neon" {
  src = "file://schema"
  url = local.db_url
  dev = "docker://postgres/17/dev?search_path=public"

  migration {
    dir              = "file://migrations"
    revisions_schema = "public"
  }

  diff {
    skip {
      drop_table = true
    }
  }
}
