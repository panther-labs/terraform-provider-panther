resource "panther_user" "example" {
  email       = "alice@example.com"
  given_name  = "Alice"
  family_name = "Smith"

  role = {
    name = "Analyst"
  }
}
