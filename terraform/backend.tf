terraform {
  backend "s3" {
    bucket = "serverlessmate-artificats"
    key    = "state/serverlessmate.tfstate"
    region = "us-east-1"
  }
}
