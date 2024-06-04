terraform {
  required_providers {
    streamkap = {
      source = "github.com/streamkap-com/streamkap"
    }
  }
  required_version = ">= 1.0"
}

provider "streamkap" {}

resource "streamkap_destination_snowflake" "example-destination-snowflake" {
  name                             = "test-destination-snowflake"
  snowflake_url_name               = "https://qivxjke-ibb99601.snowflakecomputing.com/"
  snowflake_user_name              = "STREAMKAP_USER_POSTGRESQL"
  snowflake_private_key            = "MIIFHzBJBgkqhkiG9w0BBQ0wPDAbBgkqhkiG9w0BBQwwDgQIippyfarXzysCAggA MB0GCWCGSAFlAwQBKgQQ7h8WBh+iWZO0uQELV8DlTwSCBNANGA5AMEz65aHVvAGw 0AX4DmrfPaonNp9DnfhWUAX93TwDNsALIYY7wcR7cJexIOvc2N1naharTLVT7UGi SrxBlKUaYGVeEYXeYoUvZ3u1MSTcNuwg7jJ64NucFDjrgW0g/6tdQ+MQVREc4/yv hDCK6OWSN10e+YAScvlJ4qKqLu4y4NxeMGDvlpfEnMOdSaKB7SxOlc4VC3nnvGNY WHwPtD94t1kqcM7Wn7feXLsKRoPj7O8PbXejOCrIjJS1ObHhZv2NuWxRooiHA0NB k5MUehVsXWK6V/n/T+r0il3ZX3oNzpUj2GTGYA8cN40P5bTdnlgJFS5p0zoi8YHW hqyARG6woLzfMOJnrtFWwzHjU8iKSItxqkW2VyxsGwQLiXX9rm93FfdXd0DhkWC4 ng6wsoX2KR8bpfPpUgnT86EXe9nIwJi33NrjLgu35ivm3WZ9WVr2USxNhLy++AqE /jEXGJwnGsWae8gj4RGbjyS4XW0AYIB5Dkqfg54UkDO2q/SMffAbskpogimy93Vt 5tLyKsIvSxdN/VGpaL7OLMTguHbsO4q5VKHiIgNgdGq6Ohutd42+J269Jd1ZkCW3 RuYaBRy1ZL3jC+a/V16tWGcZ5HMYHYFwpDvZ0Lj3rhCKQ6z+c9FFPMn/mHq+CQo1 wxxDmIThsYI+qyIY5oOT/WtFP5O3nmQa9ap+vdOIMFPyrKwQrphnT1kUyrHsHi1b pGwWrg24VnE7DnOrX0rq/TkN9yKRm9TkZkucZDJkXvRuF1ZI8douFXwPF0Rd634J aX+bNEeZEWyZKWqc+fdDg8JsvpAOJFP/NKlAn+BeI8QB2AOsUCMd0HWUIInA6m7X rYfu3u40Sr53wAOEqTmmeFnnwstOjcuKY6vLjs2eNQnRCP0yIz0vVUVozgAB7GaN jM0+gReKP8clJ1dED+/tPN+xVI1n/ztlK6WJdWjL71bR616/O3XQRJRvgnMggxmv ko8P48+9FP7b48L/QC9bTuZWx07hGZUL0F8r3GypTS4la0oNXdMbGlb9nYjKLzsq ENXLQxLZWGrIsGkRPXIc7k9RsP97lgWbYne9/J7R26fUmTpy82JXiRRTkq7JDDGX jJagP4EoQhRfwX08jlEyr/eDh6iIBA7l33Mi2Y7HAUSY5X1thLAd5GJvVNUrjJcL 2BHwRKgybTjVRY1RxT8cT2RBsZPiAYH/dVYfTnKFsuW9wl855oYc53+ep78YOuAx WQg6G+G2uX7oE1WzWO0mWmcYO2HRZCuyAJG4yi3ptQTerYVUlpUd3m42xrZaAZz+ tORxDDWcWWBxIjMDYubv7LxKkGKRZZcC3eFzzTdEWt+zlPC5HcXfA4s78lxoaL6k 2w35c1r5KkQOkPvxNXZM4BsVUX5H0qWTHVhm8Zr7F7i0kXjd9KjzfCF2JG5K4Oh+ Cn+dJkffWT/Q9xKzu7B3puMGpQbpkvbJQnnRvP8sMBnb5pkj16AcrwG1PdfPEQrM +aGa5tgCtgBWGmbYaa360y/3rC/yTROBEXFQiKls/QTlUHC6gexD6c+pS4N2AOaZ YrK2nxGCNxajBQj4R0LfIdZiiLl4hHk4oM/ceKvGX1SEh7HS71Pbm+f16k/99lW7 inlaWpA5RverK4nv94Jfj81r/g=="
  snowflake_private_key_passphrase = "admin1234"
  snowflake_database_name          = "STREAMKAP_POSTGRESQL"
  snowflake_schema_name            = "STREAMKAP"
  snowflake_role_name              = "STREAMKAP_ROLE"
}

output "example-destination-snowflake" {
  value = streamkap_destination_snowflake.example-destination-snowflake.id
}