
variable "region" { type = "string" }
variable "PROMPIPE_ENDPOINT" { type = "string" }
variable "PROMPIPE_ENDPOINT" { type = "string" }
variable "PROMPIPE_AUTHTOKEN" { type = "string" }

variable "zip_filename" { type = "string" }

provider "aws" {
	region = "${var.region}"
}

resource "aws_lambda_function" "fn" {
	function_name = "GitHub2Prometheus"
	description = "Delivers GitHub repo statistics to prompipe endpoint"

	filename = "${var.zip_filename}"

	handler = "github2prometheus"
	runtime = "go1.x"

	role = "${aws_iam_role.iam_lambda_role.arn}"

	timeout = 30

	environment {
		variables = {
			PROMPIPE_ENDPOINT = "${var.PROMPIPE_ENDPOINT}"
			PROMPIPE_AUTHTOKEN = "${var.PROMPIPE_AUTHTOKEN}"
      GITHUB_ORG = "${var.GITHUB_ORG}"
		}
	}
}

resource "aws_cloudwatch_event_rule" "cw_scheduledevent_rule" {
	name = "GitHub2Prometheus-schedule"
	description = "Scheduled invocation for Lambda fn"
	schedule_expression = "rate(1 hours)"
}

resource "aws_cloudwatch_event_target" "cwlambdatarget" {
	target_id = "LambdaFnInvoke"
	rule = "${aws_cloudwatch_event_rule.cw_scheduledevent_rule.name}"
	arn = "${aws_lambda_function.fn.arn}"
}

resource "aws_lambda_permission" "cloudwatch_scheduler" {
	statement_id = "AllowExecutionFromCloudWatch"
	action = "lambda:InvokeFunction"
	function_name = "${aws_lambda_function.fn.function_name}"
	principal = "events.amazonaws.com"
	source_arn = "${aws_cloudwatch_event_rule.cw_scheduledevent_rule.arn}"
}

resource "aws_iam_role" "iam_lambda_role" {
  name = "GitHub2Prometheus"

  assume_role_policy = <<EOF
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Action": "sts:AssumeRole",
      "Principal": {
        "Service": "lambda.amazonaws.com"
      },
      "Effect": "Allow",
      "Sid": ""
    }
  ]
}
EOF
}
