resource "streamkap_source_mysql" "example" {
  name      = "some-value"
  connector = "mysql"
  config = {
    "database.hostname.user.defined" = "192.168.3.47"
  }
}
