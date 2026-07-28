env "local" {
  src = "file://schema"
  url = "postgres://alt:alt@postgres:5432/alt?sslmode=disable"
  dev = "postgres://alt:alt@schema-dev:5432/alt_schema_dev?sslmode=disable"

  migration {
    dir = "file://migrations"
  }
}
