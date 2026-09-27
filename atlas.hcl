data {
  src = "file://migrations/api"
}

env "local" {
  url = getenv("DATABASE_URL")
  src = "file://migrations/api"
}
