variable "REPO" {
  default = "hominsu"
}

variable "AUTHOR_NAME" {
  default = "hominsu"
}

variable "AUTHOR_EMAIL" {
  default = "hominsu@foxmail.com"
}

variable "VERSION" {
  default = ""
}

group "default" {
  targets = [
    "pallas-service",
  ]
}

target "pallas-service" {
  context    = "."
  dockerfile = "app/pallas/service/Dockerfile"
  args       = {
    AUTHOR_NAME       = "${AUTHOR_NAME}"
    AUTHOR_EMAIL      = "${AUTHOR_EMAIL}"
    APP_RELATIVE_PATH = "pallas/service"
  }
  tags = [
    notequal("", VERSION) ? "${REPO}/pallas:${VERSION}" : "",
    "${REPO}/pallas:latest",
  ]
  platforms = ["linux/amd64", "linux/arm64", "linux/arm"]
}